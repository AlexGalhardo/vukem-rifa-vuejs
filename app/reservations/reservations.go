// Package reservations holds payment links: a link reserves one number for
// a buyer for ten minutes (PLAN.md §4.3). Every state change is one
// transaction guarded by the current status, so concurrent requests cannot
// reserve the same number twice or release a number that was paid.
package reservations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
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

// PaymentGrace keeps a reservation with a PIX charge alive a little after
// expires_at, so a payment made in the last seconds still lands before the
// sweeper releases the number.
const PaymentGrace = time.Minute

// Service manages reservations.
type Service struct {
	DB  *database.DB
	Now func() time.Time
	// Queue returns the app queue (app.Queue); nil skips the expiry job and
	// leaves expiry to the sweeper.
	Queue func() (*queue.Queue, error)
}

// New returns a Service.
func New(db *database.DB, now func() time.Time, q func() (*queue.Queue, error)) *Service {
	return &Service{DB: db, Now: now, Queue: q}
}

// Link is a new reservation and its plain token, shown once to the owner.
type Link struct {
	Reservation models.Reservation
	Token       string
}

// HashToken returns the stored form of a payment-link token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// newToken returns 256 random bits, URL-safe.
func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read never fails (Go 1.24+)

	return base64.RawURLEncoding.EncodeToString(b)
}

// ReserveOptions describe who creates a link.
type ReserveOptions struct {
	// Origin is models.OriginOwner (default), OriginPublic or OriginAPI.
	Origin string
	// BuyerUserID links the reservation to the buyer's account.
	BuyerUserID *int64
	// MaxLivePerEmail caps the live links of one buyer email in the raffle
	// (0 = no cap); checked in the reserve transaction.
	MaxLivePerEmail int
	// ConsentVersion is the LGPD notice version the buyer accepted; when
	// set, the reservation records it with the time of consent.
	ConsentVersion string
}

// ErrTooManyLinks is a 429 for a buyer email over ReserveOptions.MaxLivePerEmail.
func errTooManyLinks(ctx context.Context) error {
	return errs.New(http.StatusTooManyRequests, i18n.T(ctx, "link.too_many_for_email"))
}

// Reserve moves number of raffle r from available to reserved and creates
// its payment link (owner origin). A number that is not available is a 409.
func (s *Service) Reserve(ctx context.Context, r models.Raffle, number int, in requests.Buyer) (Link, error) {
	return s.ReserveWith(ctx, r, number, in, ReserveOptions{})
}

// ReserveWith is Reserve with an origin, a buyer account and a per-email cap.
func (s *Service) ReserveWith(ctx context.Context, r models.Raffle, number int, in requests.Buyer, o ReserveOptions) (Link, error) {
	if o.Origin == "" {
		o.Origin = models.OriginOwner
	}

	now := s.Now().UTC()
	if r.Status != models.RaffleActive || !now.Before(r.EndsAt) {
		return Link{}, errs.New(http.StatusConflict, i18n.T(ctx, "link.not_on_sale"))
	}

	if number < 1 || number > r.NumberCount {
		return Link{}, errs.New(http.StatusNotFound, i18n.T(ctx, "link.number_not_found"))
	}

	token := newToken()
	res := models.Reservation{
		RaffleID:    r.ID,
		Number:      number,
		BuyerName:   in.Name,
		BuyerEmail:  in.Email,
		BuyerPhone:  in.Phone,
		TokenHash:   HashToken(token),
		TokenSealed: orm.Encrypted[string]{V: token},
		Status:      models.ReservationPending,
		ExpiresAt:   now.Add(models.ReservationTTL),
		AmountCents: r.PriceReais * 100,
		Origin:      o.Origin,
		BuyerUserID: o.BuyerUserID,
	}

	if o.ConsentVersion != "" {
		v := o.ConsentVersion
		res.ConsentVersion = &v
		res.ConsentAt.V, res.ConsentAt.Valid = now, true
	}

	q, queued := s.queue()

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		p := tx.Dialect().Placeholder

		if o.MaxLivePerEmail > 0 {
			// Serializes this raffle's capped reserves so two concurrent
			// requests cannot both pass the count (no-op on SQLite).
			if _, err := orm.QueryOf[models.Raffle](tx).Where("id", r.ID).LockForUpdate().First(ctx); err != nil {
				return fmt.Errorf("lock raffle %d: %w", r.ID, err)
			}

			var live int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM reservations WHERE raffle_id = "+p(1)+" AND buyer_email = "+p(2)+
				" AND status IN ('"+models.ReservationPending+"','"+models.ReservationAwaitingPayment+"') AND expires_at > "+p(3),
				r.ID, in.Email, now).Scan(&live); err != nil {
				return fmt.Errorf("count live links: %w", err)
			}

			if live >= o.MaxLivePerEmail {
				return errTooManyLinks(ctx)
			}
		}

		var numberID int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM raffle_numbers WHERE raffle_id = "+p(1)+" AND number = "+p(2),
			r.ID, number).Scan(&numberID); err != nil {
			return fmt.Errorf("find number %d: %w", number, err)
		}

		out, err := tx.ExecContext(ctx, "UPDATE raffle_numbers SET status = '"+models.NumberReserved+"', updated_at = "+p(1)+
			" WHERE id = "+p(2)+" AND status = '"+models.NumberAvailable+"'", now, numberID)
		if err != nil {
			return fmt.Errorf("reserve number %d: %w", number, err)
		}

		if n, err := out.RowsAffected(); err != nil || n != 1 {
			return errs.New(http.StatusConflict, i18n.T(ctx, "link.number_unavailable"))
		}

		res.NumberID = numberID
		if err := orm.Create(ctx, tx, &res); err != nil {
			return fmt.Errorf("create reservation: %w", err)
		}

		if _, err := tx.ExecContext(ctx, "UPDATE raffle_numbers SET reservation_id = "+p(1)+" WHERE id = "+p(2),
			res.ID, numberID); err != nil {
			return fmt.Errorf("link reservation: %w", err)
		}

		if !queued {
			return nil
		}

		// The sync driver cannot delay jobs: the every-minute sweeper expires
		// the reservation instead.
		err = q.DispatchTx(ctx, tx, jobs.ExpireReservation{ReservationID: res.ID}, queue.Delay(models.ReservationTTL))
		if errors.Is(err, queue.ErrInlineDelay) {
			return nil
		}

		return err
	})
	if err != nil {
		return Link{}, err
	}

	return Link{Reservation: res, Token: token}, nil
}

func (s *Service) queue() (*queue.Queue, bool) {
	if s.Queue == nil {
		return nil, false
	}

	q, err := s.Queue()

	return q, err == nil && q != nil
}

// ByToken loads the reservation of a payment link. Unknown tokens are 404.
func (s *Service) ByToken(ctx context.Context, token string) (models.Reservation, error) {
	if token == "" || len(token) > 128 {
		return models.Reservation{}, errs.New(http.StatusNotFound, i18n.T(ctx, "link.not_found"))
	}

	res, err := orm.QueryOf[models.Reservation](s.DB).Where("token_hash", HashToken(token)).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return res, errs.New(http.StatusNotFound, i18n.T(ctx, "link.not_found"))
	}

	return res, err
}

// LiveFor returns the live reservation holding number of raffleID, if any.
func (s *Service) LiveFor(ctx context.Context, raffleID int64, number int) (models.Reservation, bool, error) {
	res, err := orm.QueryOf[models.Reservation](s.DB).Where("raffle_id", raffleID).Where("number", number).
		WhereIn("status", models.ReservationPending, models.ReservationAwaitingPayment).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return res, false, nil
	}

	return res, err == nil, err
}

// Cancel releases the live reservation of number (owner action). Nothing
// live is a 409.
func (s *Service) Cancel(ctx context.Context, raffleID int64, number int) error {
	res, ok, err := s.LiveFor(ctx, raffleID, number)
	if err != nil {
		return err
	}

	if !ok {
		return errs.New(http.StatusConflict, i18n.T(ctx, "link.nothing_to_cancel"))
	}

	released, err := s.release(ctx, res, models.ReservationCancelled)
	if err != nil {
		return err
	}

	if !released {
		return errs.New(http.StatusConflict, i18n.T(ctx, "link.nothing_to_cancel"))
	}

	return nil
}

// Expired reports whether res should be released at now: pending links at
// expires_at, links with a PIX charge after PaymentGrace.
func Expired(res models.Reservation, now time.Time) bool {
	switch res.Status {
	case models.ReservationPending:
		return !now.Before(res.ExpiresAt)
	case models.ReservationAwaitingPayment:
		return !now.Before(res.ExpiresAt.Add(PaymentGrace))
	default:
		return false
	}
}

// Expire releases reservation id if it is due; it reports whether it did.
// Not due, already paid or already released is a no-op (the job may run
// early on the sync driver or twice on retries).
func (s *Service) Expire(ctx context.Context, id int64) (bool, error) {
	res, err := orm.QueryOf[models.Reservation](s.DB).Find(ctx, id)
	if errors.Is(err, orm.ErrNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if !Expired(res, s.Now()) {
		return false, nil
	}

	return s.release(ctx, res, models.ReservationExpired)
}

// ExpireDue releases every reservation that is due and returns how many
// (the every-minute sweeper).
func (s *Service) ExpireDue(ctx context.Context) (int, error) {
	now := s.Now().UTC()

	due, err := orm.QueryOf[models.Reservation](s.DB).
		WhereIn("status", models.ReservationPending, models.ReservationAwaitingPayment).
		Where("expires_at", "<=", now).Get(ctx)
	if err != nil {
		return 0, err
	}

	n := 0

	for _, res := range due {
		if !Expired(res, now) {
			continue
		}

		ok, err := s.release(ctx, res, models.ReservationExpired)
		if err != nil {
			return n, err
		}

		if ok {
			n++
		}
	}

	return n, nil
}

// release ends a live reservation with status and frees its number, in one
// transaction guarded by both rows' current status.
func (s *Service) release(ctx context.Context, res models.Reservation, status string) (bool, error) {
	now := s.Now().UTC()
	released := false

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		p := tx.Dialect().Placeholder

		out, err := tx.ExecContext(ctx, "UPDATE reservations SET status = "+p(1)+", updated_at = "+p(2)+
			" WHERE id = "+p(3)+" AND status IN ('"+models.ReservationPending+"','"+models.ReservationAwaitingPayment+"')",
			status, now, res.ID)
		if err != nil {
			return fmt.Errorf("release reservation %d: %w", res.ID, err)
		}

		if n, err := out.RowsAffected(); err != nil || n != 1 {
			return err
		}

		if _, err := tx.ExecContext(ctx, "UPDATE raffle_numbers SET status = '"+models.NumberAvailable+"', reservation_id = NULL, updated_at = "+p(1)+
			" WHERE id = "+p(2)+" AND status = '"+models.NumberReserved+"' AND reservation_id = "+p(3),
			now, res.NumberID, res.ID); err != nil {
			return fmt.Errorf("free number %d: %w", res.Number, err)
		}

		released = true

		return nil
	})

	return released, err
}
