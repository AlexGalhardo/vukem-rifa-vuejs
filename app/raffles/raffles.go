// Package raffles holds the raffle rules: creation with numbers 1..N,
// owner-scoped access, edits locked after the first paid number, delete
// only before that (archive otherwise).
package raffles

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/jobs"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// Service manages raffles.
type Service struct {
	DB  *database.DB
	Now func() time.Time
	// Queue returns the app queue (app.Queue) for the deadline draw job;
	// nil leaves the draw to the every-minute sweeper.
	Queue func() (*queue.Queue, error)
}

// New returns a Service; now is the app clock (fake in tests).
func New(db *database.DB, now func() time.Time, q func() (*queue.Queue, error)) *Service {
	return &Service{DB: db, Now: now, Queue: q}
}

// Summary is a raffle with its number counts, for the dashboard.
type Summary struct {
	models.Raffle

	Available int `json:"available"`
	Reserved  int `json:"reserved"`
	Paid      int `json:"paid"`
}

// Create stores a raffle owned by userID and its numbers 1..NumberCount.
func (s *Service) Create(ctx context.Context, userID int64, in requests.Raffle) (models.Raffle, error) {
	now := s.Now().UTC()
	r := models.Raffle{
		UserID:       userID,
		Slug:         newSlug(),
		Title:        in.Title,
		Description:  in.Description,
		NumberCount:  in.NumberCount,
		PriceReais:   in.PriceReais,
		DurationDays: in.DurationDays,
		StartsAt:     now,
		EndsAt:       now.AddDate(0, 0, in.DurationDays),
		Status:       models.RaffleActive,
	}

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := orm.Create(ctx, tx, &r); err != nil {
			return err
		}

		if err := s.insertNumbers(ctx, tx, r.ID, 1, r.NumberCount); err != nil {
			return err
		}

		return s.scheduleDraw(ctx, tx, r)
	})

	return r, err
}

// scheduleDraw enqueues the deadline draw with tx. The sync driver (tests,
// simple dev setups) cannot delay jobs: the every-minute sweeper draws the
// raffle instead.
func (s *Service) scheduleDraw(ctx context.Context, tx *database.Tx, r models.Raffle) error {
	q, ok := s.queue()
	if !ok {
		return nil
	}

	err := q.DispatchTx(ctx, tx, drawJob(r), queue.Delay(r.EndsAt.Sub(s.Now())))
	if errors.Is(err, queue.ErrInlineDelay) {
		return nil
	}

	return err
}

func drawJob(r models.Raffle) jobs.RunDraw {
	return jobs.RunDraw{RaffleID: r.ID, Trigger: models.DrawDeadline}
}

func (s *Service) queue() (*queue.Queue, bool) {
	if s.Queue == nil {
		return nil, false
	}

	q, err := s.Queue()

	return q, err == nil && q != nil
}

// insertNumbers creates numbers from..to (inclusive) as available, in one
// statement that works on SQLite and Postgres.
func (s *Service) insertNumbers(ctx context.Context, ex database.Executor, raffleID int64, from, to int) error {
	if from > to {
		return nil
	}

	p := ex.Dialect().Placeholder
	now := s.Now().UTC()
	query := "WITH RECURSIVE seq(n) AS (SELECT CAST(" + p(1) + " AS INTEGER) UNION ALL SELECT n + 1 FROM seq WHERE n < " + p(2) + ") " +
		"INSERT INTO raffle_numbers (raffle_id, number, status, created_at, updated_at) " +
		"SELECT " + p(3) + ", n, '" + models.NumberAvailable + "', " + p(4) + ", " + p(5) + " FROM seq"

	if _, err := ex.ExecContext(ctx, query, from, to, raffleID, now, now); err != nil {
		return fmt.Errorf("insert numbers %d..%d of raffle %d: %w", from, to, raffleID, err)
	}

	return nil
}

// ForOwner loads raffle id if userID owns it; any other raffle is not found
// (404, never 403: ids of other owners are not confirmed).
func (s *Service) ForOwner(ctx context.Context, userID, id int64) (models.Raffle, error) {
	r, err := orm.QueryOf[models.Raffle](s.DB).Where("id", id).Where("user_id", userID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return r, errs.New(http.StatusNotFound, i18n.T(ctx, "raffle.not_found"))
	}

	return r, err
}

// BySlug loads a raffle for its public page.
func (s *Service) BySlug(ctx context.Context, slug string) (models.Raffle, error) {
	r, err := orm.QueryOf[models.Raffle](s.DB).Where("slug", slug).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return r, errs.New(http.StatusNotFound, i18n.T(ctx, "raffle.not_found"))
	}

	return r, err
}

// List returns the owner's raffles, newest first, with number counts.
func (s *Service) List(ctx context.Context, userID int64) ([]Summary, error) {
	list, err := orm.QueryOf[models.Raffle](s.DB).Where("user_id", userID).OrderBy("id", orm.Desc).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]Summary, 0, len(list))

	for _, r := range list {
		sum := Summary{Raffle: r}

		counts, err := s.Counts(ctx, r.ID)
		if err != nil {
			return nil, err
		}

		sum.Available, sum.Reserved, sum.Paid = counts[models.NumberAvailable], counts[models.NumberReserved], counts[models.NumberPaid]
		out = append(out, sum)
	}

	return out, nil
}

// Counts returns how many numbers of raffleID are in each status.
func (s *Service) Counts(ctx context.Context, raffleID int64) (map[string]int, error) {
	p := s.DB.Dialect().Placeholder

	rows, err := s.DB.QueryContext(ctx,
		"SELECT status, COUNT(*) FROM raffle_numbers WHERE raffle_id = "+p(1)+" GROUP BY status", raffleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}

	for rows.Next() {
		var (
			status string
			n      int
		)

		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}

		counts[status] = n
	}

	return counts, rows.Err()
}

// Numbers returns the numbers of raffleID in order.
func (s *Service) Numbers(ctx context.Context, raffleID int64) ([]models.RaffleNumber, error) {
	return orm.QueryOf[models.RaffleNumber](s.DB).Where("raffle_id", raffleID).OrderBy("number", orm.Asc).Get(ctx)
}

// Update applies in to r. Title and description always change; number
// count, price and duration only while no number is paid. Every live payment
// link (pending or awaiting payment) is cancelled in the same transaction,
// so no buyer pays on the old terms; a PIX paid after that is a late
// payment (refund_due). It returns how many links it cancelled.
func (s *Service) Update(ctx context.Context, r *models.Raffle, in requests.Raffle) (int, error) {
	if r.Status != models.RaffleActive {
		return 0, errs.New(http.StatusConflict, i18n.T(ctx, "raffle.finished"))
	}

	if r.Locked() {
		v := errs.NewValidation()
		if in.NumberCount != r.NumberCount {
			v = v.Add("number_count", i18n.T(ctx, "raffle.locked_field"))
		}

		if in.PriceReais != r.PriceReais {
			v = v.Add("price_reais", i18n.T(ctx, "raffle.locked_field"))
		}

		if in.DurationDays != r.DurationDays {
			v = v.Add("duration_days", i18n.T(ctx, "raffle.locked_field"))
		}

		if !v.Empty() {
			return 0, v
		}
	}

	cancelled := 0

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		var err error
		if cancelled, err = s.cancelLiveLinks(ctx, tx, r.ID); err != nil {
			return err
		}

		if err := s.resize(ctx, tx, r, in.NumberCount); err != nil {
			return err
		}

		moved := in.DurationDays != r.DurationDays
		r.Title, r.Description = in.Title, in.Description
		r.NumberCount, r.PriceReais, r.DurationDays = in.NumberCount, in.PriceReais, in.DurationDays
		r.EndsAt = r.StartsAt.AddDate(0, 0, in.DurationDays)

		if err := orm.Update(ctx, tx, r); err != nil {
			return err
		}

		// The job for the old deadline stays queued: the draw re-reads
		// ends_at and does nothing when it runs early.
		if !moved || r.Status != models.RaffleActive {
			return nil
		}

		return s.scheduleDraw(ctx, tx, *r)
	})
	if err != nil {
		return 0, err
	}

	return cancelled, nil
}

// cancelLiveLinks cancels every pending or awaiting-payment reservation of
// raffleID and frees its numbers, guarded by status like reservations.release.
func (s *Service) cancelLiveLinks(ctx context.Context, tx *database.Tx, raffleID int64) (int, error) {
	p := tx.Dialect().Placeholder
	now := s.Now().UTC()

	out, err := tx.ExecContext(ctx, "UPDATE reservations SET status = '"+models.ReservationCancelled+"', updated_at = "+p(1)+
		" WHERE raffle_id = "+p(2)+" AND status IN ('"+models.ReservationPending+"','"+models.ReservationAwaitingPayment+"')",
		now, raffleID)
	if err != nil {
		return 0, fmt.Errorf("cancel links of raffle %d: %w", raffleID, err)
	}

	n, err := out.RowsAffected()
	if err != nil {
		return 0, err
	}

	if _, err := tx.ExecContext(ctx, "UPDATE raffle_numbers SET status = '"+models.NumberAvailable+"', reservation_id = NULL, updated_at = "+p(1)+
		" WHERE raffle_id = "+p(2)+" AND status = '"+models.NumberReserved+"'", now, raffleID); err != nil {
		return 0, fmt.Errorf("free numbers of raffle %d: %w", raffleID, err)
	}

	return int(n), nil
}

// resize adds or removes trailing numbers; removing is refused when one of
// them is reserved or paid.
func (s *Service) resize(ctx context.Context, tx *database.Tx, r *models.Raffle, count int) error {
	switch {
	case count > r.NumberCount:
		return s.insertNumbers(ctx, tx, r.ID, r.NumberCount+1, count)
	case count < r.NumberCount:
		p := tx.Dialect().Placeholder

		var busy int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM raffle_numbers WHERE raffle_id = "+p(1)+
			" AND number > "+p(2)+" AND status <> '"+models.NumberAvailable+"'", r.ID, count).Scan(&busy); err != nil {
			return err
		}

		if busy > 0 {
			return errs.NewValidation().Add("number_count", i18n.T(ctx, "raffle.shrink_busy"))
		}

		_, err := tx.ExecContext(ctx, "DELETE FROM raffle_numbers WHERE raffle_id = "+p(1)+" AND number > "+p(2), r.ID, count)

		return err
	}

	return nil
}

// Delete removes r and its numbers, only while no number is paid.
func (s *Service) Delete(ctx context.Context, r *models.Raffle) error {
	if r.Locked() {
		return errs.New(http.StatusConflict, i18n.T(ctx, "raffle.delete_locked"))
	}

	return orm.Delete(ctx, s.DB, r)
}

// Archive hides r from new sales; its numbers, payments and result stay.
func (s *Service) Archive(ctx context.Context, r *models.Raffle) error {
	if r.Status == models.RaffleArchived {
		return nil
	}

	r.Status = models.RaffleArchived
	r.ArchivedAt = orm.NullTime{Null: sql.Null[time.Time]{V: s.Now().UTC(), Valid: true}}

	return orm.Update(ctx, s.DB, r)
}

// newSlug returns a random, URL-safe public identifier (50 bits).
func newSlug() string { return strings.ToLower(rand.Text()[:10]) }
