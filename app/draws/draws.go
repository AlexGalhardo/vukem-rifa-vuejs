package draws

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// ErrUnknownTrigger is returned by Run for a trigger other than
// models.DrawDeadline and models.DrawSoldOut.
var ErrUnknownTrigger = errors.New("draws: unknown trigger")

// Service runs draws.
type Service struct {
	DB  *database.DB
	Now func() time.Time
}

// New returns a Service; now is the app clock (fake in tests).
func New(db *database.DB, now func() time.Time) *Service { return &Service{DB: db, Now: now} }

// Run draws raffle raffleID when trigger is due: the deadline has passed
// (models.DrawDeadline) or every number is paid (models.DrawSoldOut). It
// returns ok=false, without error, when there is nothing to do: the raffle
// is gone, not active (already drawn, closed or archived), or the trigger
// is not due yet (a delayed job running early, a re-dispatched deadline,
// a sold-out signal for a raffle that is not fully paid). Concurrent runs
// are safe: only the run whose guarded UPDATE moves the raffle out of
// "active" draws, and draws.raffle_id is unique as a backstop.
//
// Live reservations are released in the same transaction, so no number
// can be paid after the draw (a late PIX becomes a refund).
func (s *Service) Run(ctx context.Context, raffleID int64, trigger string) (models.Draw, bool, error) {
	if trigger != models.DrawDeadline && trigger != models.DrawSoldOut {
		return models.Draw{}, false, fmt.Errorf("%w: %q", ErrUnknownTrigger, trigger)
	}

	now := s.Now().UTC()
	seed := make([]byte, SeedSize)
	_, _ = rand.Read(seed) // crypto/rand.Read never fails (Go 1.24+)

	var (
		d  models.Draw
		ok bool
	)

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		p := tx.Dialect().Placeholder

		lock := ""
		if tx.Dialect().Name() == "postgres" {
			lock = " FOR UPDATE"
		}

		var (
			status string
			endsAt time.Time
			count  int
		)

		err := tx.QueryRowContext(ctx, "SELECT status, ends_at, number_count FROM raffles WHERE id = "+p(1)+lock, raffleID).
			Scan(&status, &endsAt, &count)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("lock raffle %d: %w", raffleID, err)
		}

		if status != models.RaffleActive {
			return nil
		}

		paid, err := paidNumbers(ctx, tx, raffleID)
		if err != nil {
			return err
		}

		if trigger == models.DrawDeadline && now.Before(endsAt) || trigger == models.DrawSoldOut && len(paid) < count {
			return nil
		}

		final := models.RaffleDrawn
		if len(paid) == 0 {
			final = models.RaffleClosed
		}

		out, err := tx.ExecContext(ctx, "UPDATE raffles SET status = "+p(1)+", updated_at = "+p(2)+
			" WHERE id = "+p(3)+" AND status = '"+models.RaffleActive+"'", final, now, raffleID)
		if err != nil {
			return fmt.Errorf("end raffle %d: %w", raffleID, err)
		}

		if n, err := out.RowsAffected(); err != nil || n != 1 {
			return err // another run ended it first
		}

		if err := releaseLive(ctx, tx, raffleID, now); err != nil {
			return err
		}

		d = models.Draw{
			RaffleID:           raffleID,
			DrawnAt:            now,
			Seed:               hex.EncodeToString(seed),
			ParticipantsCount:  len(paid),
			ParticipantsSHA256: ParticipantsHash(paid),
			Algorithm:          AlgorithmV1,
			Trigger:            trigger,
		}

		if len(paid) > 0 {
			w, err := Pick(seed, paid)
			if err != nil {
				return err
			}

			d.WinningNumber = &w
		}

		if err := orm.Create(ctx, tx, &d); err != nil {
			return fmt.Errorf("store draw of raffle %d: %w", raffleID, err)
		}

		ok = true

		return nil
	})
	if err != nil {
		return models.Draw{}, false, err
	}

	return d, ok, nil
}

func paidNumbers(ctx context.Context, tx *database.Tx, raffleID int64) ([]int, error) {
	p := tx.Dialect().Placeholder

	rows, err := tx.QueryContext(ctx, "SELECT number FROM raffle_numbers WHERE raffle_id = "+p(1)+
		" AND status = '"+models.NumberPaid+"' ORDER BY number", raffleID)
	if err != nil {
		return nil, fmt.Errorf("paid numbers of raffle %d: %w", raffleID, err)
	}
	defer rows.Close()

	var out []int

	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}

		out = append(out, n)
	}

	return out, rows.Err()
}

// releaseLive expires the raffle's live payment links and frees their
// numbers.
func releaseLive(ctx context.Context, tx *database.Tx, raffleID int64, now time.Time) error {
	p := tx.Dialect().Placeholder

	if _, err := tx.ExecContext(ctx, "UPDATE reservations SET status = '"+models.ReservationExpired+"', updated_at = "+p(1)+
		" WHERE raffle_id = "+p(2)+" AND status IN ('"+models.ReservationPending+"','"+models.ReservationAwaitingPayment+"')",
		now, raffleID); err != nil {
		return fmt.Errorf("expire links of raffle %d: %w", raffleID, err)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE raffle_numbers SET status = '"+models.NumberAvailable+"', reservation_id = NULL, updated_at = "+p(1)+
		" WHERE raffle_id = "+p(2)+" AND status = '"+models.NumberReserved+"'", now, raffleID); err != nil {
		return fmt.Errorf("free numbers of raffle %d: %w", raffleID, err)
	}

	return nil
}

// RunDue draws every active raffle that is sold out or past its deadline
// (the every-minute sweeper behind the delayed jobs) and returns how many
// it ended.
func (s *Service) RunDue(ctx context.Context) (int, error) {
	now := s.Now().UTC()

	list, err := s.due(ctx, now)
	if err != nil {
		return 0, err
	}

	n := 0

	for _, d := range list {
		// Past the deadline is always a deadline draw; before it, the query
		// only returned the raffle because it is sold out.
		trigger := models.DrawSoldOut
		if !now.Before(d.endsAt) {
			trigger = models.DrawDeadline
		}

		_, ok, err := s.Run(ctx, d.id, trigger)
		if err != nil {
			return n, err
		}

		if ok {
			n++
		}
	}

	return n, nil
}

type dueRaffle struct {
	id     int64
	endsAt time.Time
}

// due lists the active raffles past their deadline or sold out at now.
func (s *Service) due(ctx context.Context, now time.Time) ([]dueRaffle, error) {
	p := s.DB.Dialect().Placeholder

	rows, err := s.DB.QueryContext(ctx, "SELECT r.id, r.ends_at FROM raffles r WHERE r.status = '"+models.RaffleActive+"'"+
		" AND (r.ends_at <= "+p(1)+" OR NOT EXISTS (SELECT 1 FROM raffle_numbers n WHERE n.raffle_id = r.id"+
		" AND n.status <> '"+models.NumberPaid+"')) ORDER BY r.id", now)
	if err != nil {
		return nil, fmt.Errorf("due raffles: %w", err)
	}
	defer rows.Close()

	var list []dueRaffle

	for rows.Next() {
		var d dueRaffle
		if err := rows.Scan(&d.id, &d.endsAt); err != nil {
			return nil, err
		}

		list = append(list, d)
	}

	return list, rows.Err()
}

// ForRaffle returns the draw of raffleID, if it was drawn or closed.
func (s *Service) ForRaffle(ctx context.Context, raffleID int64) (models.Draw, bool, error) {
	d, err := orm.QueryOf[models.Draw](s.DB).Where("raffle_id", raffleID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return d, false, nil
	}

	return d, err == nil, err
}
