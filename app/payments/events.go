package payments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/jobs"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Provider is payment_events.provider for AbacatePay.
const Provider = "abacatepay"

// Outcome says what a webhook delivery did.
type Outcome string

// Webhook outcomes.
const (
	OutcomeDuplicate     Outcome = "duplicate"      // event id already recorded
	OutcomeIgnored       Outcome = "ignored"        // event type or charge status the app does not act on
	OutcomeUnknownCharge Outcome = "unknown_charge" // no reservation for the charge
	OutcomeAlreadyPaid   Outcome = "already_paid"   // another event already paid the reservation
	OutcomePaid          Outcome = "paid"
	OutcomeRefundDue     Outcome = "refund_due" // money received but no number assigned
)

// Handle records a verified webhook event and applies it in one
// transaction: the event row (unique event_id makes redeliveries a no-op),
// the reservation and number state, the ledger rows and the queued emails
// commit together or not at all. Call it only after the webhook secret and
// signature were checked.
func (s *Service) Handle(ctx context.Context, ev abacatepay.Event, payload []byte) (Outcome, error) {
	var (
		outcome Outcome
		after   []queue.Job
	)

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		after = nil

		recorded, err := s.record(ctx, tx, ev, payload)
		if err != nil || !recorded {
			outcome = OutcomeDuplicate
			return err
		}

		outcome, after, err = s.apply(ctx, tx, ev)
		if err != nil {
			return err
		}

		p := tx.Dialect().Placeholder
		_, err = tx.ExecContext(ctx, "UPDATE payment_events SET processed_at = "+p(1)+", updated_at = "+p(2)+
			" WHERE event_id = "+p(3), s.Now().UTC(), s.Now().UTC(), ev.ID)

		return err
	})
	if err != nil {
		return "", fmt.Errorf("handle %s event %s: %w", ev.Type, ev.ID, err)
	}

	for _, j := range after {
		if err := s.dispatch(ctx, nil, j); err != nil {
			s.Log.ErrorContext(ctx, "payments: dispatch after commit", "job", j.Kind(), "err", err)
		}
	}

	return outcome, nil
}

// record inserts the event; false means the event id was already stored.
func (s *Service) record(ctx context.Context, tx *database.Tx, ev abacatepay.Event, payload []byte) (bool, error) {
	p := tx.Dialect().Placeholder
	now := s.Now().UTC()

	out, err := tx.ExecContext(ctx, "INSERT INTO payment_events (provider, event_id, event_type, payload, received_at, created_at, updated_at)"+
		" VALUES ("+p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", "+p(5)+", "+p(6)+", "+p(7)+") ON CONFLICT (event_id) DO NOTHING",
		Provider, ev.ID, ev.Type, string(payload), now, now, now)
	if err != nil {
		return false, fmt.Errorf("record event: %w", err)
	}

	n, err := out.RowsAffected()

	return n == 1, err
}

func (s *Service) apply(ctx context.Context, tx *database.Tx, ev abacatepay.Event) (Outcome, []queue.Job, error) {
	if ev.Type != abacatepay.EventTransparentCompleted || ev.ChargeStatus != abacatepay.StatusPaid {
		return OutcomeIgnored, nil, nil
	}

	res, byCharge, found, err := s.lockReservation(ctx, tx, ev)
	if err != nil || !found {
		if found || err != nil {
			return "", nil, err
		}

		s.Log.WarnContext(ctx, "payments: paid charge without reservation", "charge_id", ev.ChargeID, "event_id", ev.ID)

		return OutcomeUnknownCharge, nil, nil
	}

	p := tx.Dialect().Placeholder
	if _, err := tx.ExecContext(ctx, "UPDATE payment_events SET reservation_id = "+p(1)+" WHERE event_id = "+p(2), res.ID, ev.ID); err != nil {
		return "", nil, err
	}

	raffle, err := orm.QueryOf[models.Raffle](tx).Find(ctx, res.RaffleID)
	if err != nil {
		return "", nil, fmt.Errorf("load raffle %d: %w", res.RaffleID, err)
	}

	now := s.Now().UTC()

	switch {
	case res.Status == models.ReservationPaid && byCharge:
		return OutcomeAlreadyPaid, nil, nil
	case !byCharge:
		return s.refundDue(ctx, tx, raffle, res, ev, "charge does not match the reservation")
	case ev.AmountCents != int64(res.AmountCents):
		return s.refundDue(ctx, tx, raffle, res, ev, fmt.Sprintf("amount mismatch: expected %d", res.AmountCents))
	case res.Status != models.ReservationPending && res.Status != models.ReservationAwaitingPayment:
		return s.refundDue(ctx, tx, raffle, res, ev, "late payment: reservation "+res.Status)
	case reservations.Expired(res, now):
		return s.refundDue(ctx, tx, raffle, res, ev, "late payment: reservation expired")
	}

	paid, err := s.markPaid(ctx, tx, raffle, res, now)
	if err != nil {
		return "", nil, err
	}

	if !paid {
		return s.refundDue(ctx, tx, raffle, res, ev, "number no longer held by the reservation")
	}

	var after []queue.Job

	if j, err := s.dispatchTx(ctx, tx, jobs.SendPaymentEmails{ReservationID: res.ID}); err != nil {
		return "", nil, err
	} else if j != nil {
		after = append(after, j)
	}

	var left int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM raffle_numbers WHERE raffle_id = "+p(1)+
		" AND status <> '"+models.NumberPaid+"'", raffle.ID).Scan(&left); err != nil {
		return "", nil, fmt.Errorf("count unpaid numbers: %w", err)
	}

	if left == 0 {
		j, err := s.dispatchTx(ctx, tx, jobs.RunDraw{RaffleID: raffle.ID, Trigger: models.DrawSoldOut})
		if err != nil {
			// The draw handler lives in another module; a payment must
			// never roll back because of it, the deadline sweeper still
			// draws the raffle.
			s.Log.ErrorContext(ctx, "payments: dispatch sold-out draw", "raffle_id", raffle.ID, "err", err)
		} else if j != nil {
			after = append(after, j)
		}
	}

	return OutcomePaid, after, nil
}

// lockReservation finds the reservation of the paid charge and, on
// Postgres, locks its row until the transaction ends. byCharge is false
// when it was found only through externalId (the stored charge differs).
//
// workaround(vukem v0.22.0): orm.Query has no ForUpdate(), the lock is a raw SELECT … FOR UPDATE.
func (s *Service) lockReservation(ctx context.Context, tx *database.Tx, ev abacatepay.Event) (models.Reservation, bool, bool, error) {
	p := tx.Dialect().Placeholder

	lock := ""
	if tx.Dialect().Name() == "postgres" {
		lock = " FOR UPDATE"
	}

	byCharge := true

	var id int64

	err := tx.QueryRowContext(ctx, "SELECT id FROM reservations WHERE charge_id = "+p(1)+lock, ev.ChargeID).Scan(&id)
	if isNoRows(err) {
		byCharge = false

		ext, convErr := strconv.ParseInt(ev.ExternalID, 10, 64)
		if convErr != nil || ext < 1 {
			return models.Reservation{}, false, false, nil
		}

		err = tx.QueryRowContext(ctx, "SELECT id FROM reservations WHERE id = "+p(1)+lock, ext).Scan(&id)
		if isNoRows(err) {
			return models.Reservation{}, false, false, nil
		}
	}

	if err != nil {
		return models.Reservation{}, false, false, fmt.Errorf("lock reservation: %w", err)
	}

	res, err := orm.QueryOf[models.Reservation](tx).Find(ctx, id)
	if err != nil {
		return res, false, false, fmt.Errorf("load reservation %d: %w", id, err)
	}

	return res, byCharge, true, nil
}

// markPaid moves the number reserved → paid and the reservation to paid,
// locks the raffle's price and size, and writes the 90/10 ledger rows. It
// reports false (and changes nothing) when the number is no longer held by
// this reservation.
func (s *Service) markPaid(ctx context.Context, tx *database.Tx, raffle models.Raffle, res models.Reservation, now time.Time) (bool, error) {
	p := tx.Dialect().Placeholder

	out, err := tx.ExecContext(ctx, "UPDATE raffle_numbers SET status = '"+models.NumberPaid+"', paid_at = "+p(1)+", updated_at = "+p(2)+
		" WHERE id = "+p(3)+" AND status = '"+models.NumberReserved+"' AND reservation_id = "+p(4), now, now, res.NumberID, res.ID)
	if err != nil {
		return false, fmt.Errorf("pay number %d: %w", res.Number, err)
	}

	if n, err := out.RowsAffected(); err != nil || n != 1 {
		return false, err
	}

	out, err = tx.ExecContext(ctx, "UPDATE reservations SET status = '"+models.ReservationPaid+"', paid_at = "+p(1)+", updated_at = "+p(2)+
		" WHERE id = "+p(3)+" AND status IN ('"+models.ReservationPending+"','"+models.ReservationAwaitingPayment+"')", now, now, res.ID)
	if err != nil {
		return false, fmt.Errorf("pay reservation %d: %w", res.ID, err)
	}

	if n, err := out.RowsAffected(); err != nil || n != 1 {
		// The number was ours but the reservation changed under us: undo.
		return false, errors.Join(errReservationChanged, err)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE raffles SET locked_at = "+p(1)+" WHERE id = "+p(2)+" AND locked_at IS NULL", now, raffle.ID); err != nil {
		return false, fmt.Errorf("lock raffle %d: %w", raffle.ID, err)
	}

	owner, fee := ledger.Split(int64(res.AmountCents))
	for _, e := range []models.LedgerEntry{
		{UserID: raffle.UserID, RaffleID: &raffle.ID, ReservationID: &res.ID, Kind: models.LedgerSaleOwnerShare, AmountCents: owner, Note: fmt.Sprintf("number %d", res.Number)},
		{UserID: raffle.UserID, RaffleID: &raffle.ID, ReservationID: &res.ID, Kind: models.LedgerPlatformFee, AmountCents: fee, Note: fmt.Sprintf("number %d", res.Number)},
	} {
		if e.AmountCents == 0 {
			continue
		}

		if err := orm.Create(ctx, tx, &e); err != nil {
			return false, fmt.Errorf("ledger %s: %w", e.Kind, err)
		}
	}

	return true, nil
}

var errReservationChanged = errors.New("payments: reservation changed while paying")

// refundDue records money that bought nothing: the platform owes it back
// to the buyer (positive refund_due under the raffle owner, outside the
// owner's balance). The number is never assigned.
func (s *Service) refundDue(ctx context.Context, tx *database.Tx, raffle models.Raffle, res models.Reservation, ev abacatepay.Event, why string) (Outcome, []queue.Job, error) {
	e := models.LedgerEntry{
		UserID: raffle.UserID, RaffleID: &raffle.ID, ReservationID: &res.ID, Kind: models.LedgerRefundDue,
		AmountCents: ev.AmountCents, Note: truncateNote(fmt.Sprintf("charge %s: %s", ev.ChargeID, why)),
	}
	if err := orm.Create(ctx, tx, &e); err != nil {
		return "", nil, fmt.Errorf("ledger refund_due: %w", err)
	}

	s.Log.WarnContext(ctx, "payments: refund due", "reservation_id", res.ID, "event_id", ev.ID, "reason", why)

	return OutcomeRefundDue, nil, nil
}

func truncateNote(s string) string {
	if len(s) > 255 {
		return s[:255]
	}

	return s
}

// dispatchTx enqueues j in tx. Drivers without transactional push (sync,
// memory) return the job to dispatch after commit instead.
func (s *Service) dispatchTx(ctx context.Context, tx *database.Tx, j queue.Job) (queue.Job, error) {
	err := s.dispatch(ctx, tx, j)
	if errors.Is(err, queue.ErrNoTx) {
		return j, nil
	}

	return nil, err
}

func (s *Service) dispatch(ctx context.Context, tx *database.Tx, j queue.Job) error {
	if s.Queue == nil {
		return nil
	}

	q, err := s.Queue()
	if err != nil || q == nil {
		return err
	}

	if tx == nil {
		return q.Dispatch(ctx, j)
	}

	return q.DispatchTx(ctx, tx, j)
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
