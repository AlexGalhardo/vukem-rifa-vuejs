// Package payments is the buyer checkout (confirm a payment link, create
// its PIX charge, poll its status) and the AbacatePay webhook that turns a
// confirmed PIX into a paid number with its ledger rows (PLAN.md §4.3-4.5).
package payments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Checkout states shown to the buyer.
const (
	StateLive    = "live"
	StatePaid    = "paid"
	StateExpired = "expired"
)

// Service runs the buyer checkout and the webhook.
type Service struct {
	DB  *database.DB
	Now func() time.Time
	// Gateway is nil when AbacatePay is not configured: confirming then
	// answers 503 and nothing else breaks.
	Gateway abacatepay.Gateway
	// Queue returns the app queue; nil dispatches nothing (tests).
	Queue func() (*queue.Queue, error)
	Log   *slog.Logger

	res *reservations.Service
}

// New returns a Service.
func New(db *database.DB, now func() time.Time, gw abacatepay.Gateway, q func() (*queue.Queue, error), log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}

	return &Service{DB: db, Now: now, Gateway: gw, Queue: q, Log: log, res: reservations.New(db, now, q)}
}

// Checkout is what the buyer page shows for a payment link.
type Checkout struct {
	State       string
	Reservation models.Reservation
	Raffle      models.Raffle
	// SecondsLeft until the link expires (0 once it is over).
	SecondsLeft int
}

// Load returns the checkout of token. Unknown tokens are a 404.
func (s *Service) Load(ctx context.Context, token string) (Checkout, error) {
	res, err := s.res.ByToken(ctx, token)
	if err != nil {
		return Checkout{}, err
	}

	raffle, err := orm.QueryOf[models.Raffle](s.DB).Find(ctx, res.RaffleID)
	if err != nil {
		return Checkout{}, fmt.Errorf("load raffle %d: %w", res.RaffleID, err)
	}

	now := s.Now().UTC()
	co := Checkout{State: StateLive, Reservation: res, Raffle: raffle}

	switch {
	case res.Status == models.ReservationPaid:
		co.State = StatePaid
	case res.Status == models.ReservationExpired || res.Status == models.ReservationCancelled || reservations.Expired(res, now):
		co.State = StateExpired
	case now.Before(res.ExpiresAt):
		co.SecondsLeft = int(res.ExpiresAt.Sub(now) / time.Second)
	}

	// The PIX must not outlive the raffle either.
	if left := int(raffle.EndsAt.Sub(now) / time.Second); co.State == StateLive && left < co.SecondsLeft {
		co.SecondsLeft = max(left, 0)
	}

	return co, nil
}

// errGone is the expired-link answer.
func errGone(ctx context.Context) error {
	return errs.New(http.StatusGone, i18n.T(ctx, "pay.expired"))
}

// ErrUnavailable marks a confirm that failed because the gateway is not
// configured or did not answer; the buyer may retry.
var ErrUnavailable = errors.New("payments: gateway unavailable")

func unavailable(ctx context.Context, key string) error {
	return &errs.HTTPError{Status: http.StatusServiceUnavailable, Message: i18n.T(ctx, key), Err: ErrUnavailable}
}

// Confirm creates the PIX charge of a live link (or returns the existing
// one: confirming twice is idempotent) and moves the reservation to
// awaiting_payment. The QR code expires with the reservation.
func (s *Service) Confirm(ctx context.Context, token string) (Checkout, error) {
	co, err := s.Load(ctx, token)
	if err != nil {
		return co, err
	}

	res := co.Reservation

	switch {
	case co.State == StateExpired:
		return co, errGone(ctx)
	case co.State == StatePaid:
		return co, nil
	case res.Status == models.ReservationAwaitingPayment && res.ChargeID != nil:
		return co, nil
	case co.SecondsLeft < 1:
		return co, errGone(ctx)
	case s.Gateway == nil:
		return co, unavailable(ctx, "pay.unavailable")
	}

	ch, err := s.Gateway.CreatePix(ctx, abacatepay.PixRequest{
		AmountCents: int64(res.AmountCents),
		ExpiresIn:   co.SecondsLeft,
		Description: description(co.Raffle.Title, res.Number),
		ExternalID:  strconv.FormatInt(res.ID, 10),
	})
	if err != nil {
		s.Log.ErrorContext(ctx, "payments: create pix", "reservation_id", res.ID, "err", err)
		return co, unavailable(ctx, "pay.gateway_error")
	}

	if ch.AmountCents != int64(res.AmountCents) {
		s.Log.ErrorContext(ctx, "payments: pix amount mismatch", "reservation_id", res.ID,
			"want", res.AmountCents, "got", ch.AmountCents)

		return co, unavailable(ctx, "pay.gateway_error")
	}

	now := s.Now().UTC()
	p := s.DB.Dialect().Placeholder

	out, err := s.DB.ExecContext(ctx, "UPDATE reservations SET status = '"+models.ReservationAwaitingPayment+"', charge_id = "+p(1)+
		", br_code = "+p(2)+", br_code_base64 = "+p(3)+", locale = "+p(4)+", updated_at = "+p(5)+
		" WHERE id = "+p(6)+" AND status = '"+models.ReservationPending+"' AND expires_at > "+p(7),
		ch.ID, ch.BrCode, ch.BrCodeBase64, i18n.Locale(ctx), now, res.ID, now)
	if err != nil {
		return co, fmt.Errorf("store charge of reservation %d: %w", res.ID, err)
	}

	if n, err := out.RowsAffected(); err != nil || n != 1 {
		// A concurrent confirm won (or the link just expired): show its state.
		again, err := s.Load(ctx, token)
		if err != nil {
			return again, err
		}

		if again.State == StateExpired || again.Reservation.ChargeID == nil {
			return again, errGone(ctx)
		}

		return again, nil
	}

	return s.Load(ctx, token)
}

func description(title string, number int) string {
	d := fmt.Sprintf("%s #%d", title, number)
	for len(d) > 500 || !utf8.ValidString(d) {
		d = d[:len(d)-1]
	}

	return d
}

// Status is the polled state of a link: live links report the reservation
// status (pending or awaiting_payment).
func (s *Service) Status(ctx context.Context, token string) (string, error) {
	co, err := s.Load(ctx, token)
	if err != nil {
		return "", err
	}

	if co.State == StateLive {
		return co.Reservation.Status, nil
	}

	return co.State, nil
}
