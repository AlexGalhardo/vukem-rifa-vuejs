//go:build integration

package tests

import (
	"net/http"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/tests/pgtest"
)

func postgresEnv(t *testing.T) vukemtest.Options {
	t.Helper()

	return vukemtest.Options{Env: map[string]string{
		"DATABASE_URL":     pgtest.NewDatabase(t),
		"SESSION_DRIVER":   "database",
		"QUEUE_CONNECTION": "database",
		"CACHE_STORE":      "database",
		"RATE_LIMIT_STORE": "cache",
	}}
}

// Twenty owners' requests racing for the same number: exactly one link.
func TestPostgres_ConcurrentReserveSameNumber(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, postgresEnv(t))

	const workers = 20

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		ok, taken int
	)

	start := make(chan struct{})

	for range workers {
		wg.Go(func() {
			<-start

			_, err := f.svc.Reserve(t.Context(), f.raffle, 42, buyer)

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				ok++
			case errs.Status(err) == http.StatusConflict:
				taken++
			default:
				t.Errorf("Reserve: %v", err)
			}
		})
	}

	close(start)
	wg.Wait()

	if ok != 1 || taken != workers-1 {
		t.Fatalf("ok = %d, conflicts = %d; want 1 and %d", ok, taken, workers-1)
	}

	var live int
	if err := f.svc.DB.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM reservations WHERE raffle_id = $1 AND number = 42 AND status IN ('pending','awaiting_payment')",
		f.raffle.ID).Scan(&live); err != nil || live != 1 {
		t.Fatalf("live reservations = %d, %v", live, err)
	}

	// The expiry job was enqueued in the same transaction as the link.
	var queued int
	if err := f.svc.DB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM jobs").Scan(&queued); err != nil || queued < 1 {
		t.Fatalf("queued jobs = %d, %v", queued, err)
	}
}

func TestPostgres_PaymentSchemaConstraints(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, postgresEnv(t))
	db := f.svc.DB
	ctx := t.Context()

	link, err := f.svc.Reserve(ctx, f.raffle, 1, buyer)
	if err != nil {
		t.Fatal(err)
	}

	resID := link.Reservation.ID
	numberID := link.Reservation.NumberID

	// A second live reservation for the same number is refused by the
	// partial unique index even if the guarded UPDATE were bypassed.
	if _, err := db.ExecContext(ctx, `INSERT INTO reservations (raffle_id, number_id, number, buyer_name, buyer_email, buyer_phone,
		token_hash, status, expires_at, amount_cents, created_at, updated_at)
		VALUES ($1, $2, 1, 'X', 'x@example.com', '+5511987654321', 'dup', 'pending', now(), 1000, now(), now())`,
		f.raffle.ID, numberID); err == nil {
		t.Error("second live reservation accepted")
	}

	var userID int64
	if err := db.QueryRowContext(ctx, "SELECT user_id FROM raffles WHERE id = $1", f.raffle.ID).Scan(&userID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO ledger_entries (user_id, raffle_id, reservation_id, kind, amount_cents, note, created_at, updated_at)
		VALUES ($1, $2, $3, 'sale_owner_share', 900, '', now(), now())`, userID, f.raffle.ID, resID); err != nil {
		t.Fatal(err)
	}

	bad := []struct {
		name string
		sql  string
		args []any
	}{
		{"reservation status", "UPDATE reservations SET status = 'bogus' WHERE id = $1", []any{resID}},
		{"reservation amount", "UPDATE reservations SET amount_cents = 50 WHERE id = $1", []any{resID}},
		{"ledger update", "UPDATE ledger_entries SET amount_cents = 1 WHERE user_id = $1", []any{userID}},
		{"ledger delete", "DELETE FROM ledger_entries WHERE user_id = $1", []any{userID}},
		{"ledger kind", "INSERT INTO ledger_entries (user_id, kind, amount_cents, note, created_at, updated_at) VALUES ($1, 'gift', 1, '', now(), now())", []any{userID}},
		{"ledger zero", "INSERT INTO ledger_entries (user_id, kind, amount_cents, note, created_at, updated_at) VALUES ($1, 'payout', 0, '', now(), now())", []any{userID}},
		{"payout amount", "INSERT INTO payouts (user_id, amount_cents, reference, recorded_by, created_at, updated_at) VALUES ($1, 0, 'x', $1, now(), now())", []any{userID}},
		{"draw trigger", "INSERT INTO draws (raffle_id, drawn_at, seed, participants_count, participants_sha256, algorithm, \"trigger\", created_at, updated_at) VALUES ($1, now(), 's', 0, 'h', 'v1', 'manual', now(), now())", []any{f.raffle.ID}},
		{"duplicate event", "INSERT INTO payment_events (provider, event_id, event_type, payload, received_at, created_at, updated_at) VALUES ('abacatepay', 'e1', 't', '{}', now(), now(), now()), ('abacatepay', 'e1', 't', '{}', now(), now(), now())", nil},
	}

	for _, tc := range bad {
		if _, err := db.ExecContext(ctx, tc.sql, tc.args...); err == nil {
			t.Errorf("%s: accepted, want a constraint error", tc.name)
		}
	}
}
