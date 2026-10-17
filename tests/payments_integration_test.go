//go:build integration

package tests

import (
	"fmt"
	"maps"
	"net/http"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Concurrent deliveries of the same event (retries racing) and of new
// events for the same charge (dashboard resends) pay the number once.
func TestPostgres_ConcurrentWebhookDeliveries(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, mergeEnv(postgresEnv(t), payEnv().Env))
	_, res := f.confirmed(t, 30)

	const same, resends = 10, 5

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses = map[int]int{}
	)

	start := make(chan struct{})

	for i := range same + resends {
		id := "log_race"
		if i >= same {
			id = fmt.Sprintf("log_resend_%d", i)
		}

		body := paidEvent(id, *res.ChargeID, res.ID, 1000)

		wg.Go(func() {
			<-start

			status := deliverSigned(t, f.app, body).Status()

			mu.Lock()
			statuses[status]++
			mu.Unlock()
		})
	}

	close(start)
	wg.Wait()

	for status, n := range statuses {
		if status != http.StatusOK && status != http.StatusConflict {
			t.Errorf("%d deliveries answered %d", n, status)
		}
	}

	if n, sum := ledgerSum(t, f, models.LedgerSaleOwnerShare); n != 1 || sum != 900 {
		t.Fatalf("owner share = %d rows, %d", n, sum)
	}

	if n, sum := ledgerSum(t, f, models.LedgerPlatformFee); n != 1 || sum != 100 {
		t.Fatalf("platform fee = %d rows, %d", n, sum)
	}

	if f.numberStatus(t, 30) != models.NumberPaid {
		t.Fatal("number not paid")
	}

	// The emails job was enqueued in the payment transaction.
	var queued int
	if err := f.svc.DB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM jobs WHERE payload::text LIKE '%reservation_id%' AND kind = 'payment.emails'").
		Scan(&queued); err != nil || queued != 1 {
		t.Fatalf("queued payment emails = %d, %v", queued, err)
	}
}

// A double submit (or two admins) cannot pay out the same balance twice.
func TestPostgres_ConcurrentPayoutsNeverExceedTheBalance(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, mergeEnv(postgresEnv(t), payEnv().Env))
	f.paidSale(t, 31)
	makeAdmin(t, f.app, "admin@example.com")

	const workers = 10

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses = map[int]int{}
	)

	start := make(chan struct{})

	for i := range workers {
		c := loginJSON(t, f.app, "admin@example.com")

		wg.Go(func() {
			<-start

			status := c.PostJSON("/admin/payouts", map[string]any{
				"user_id": f.raffle.UserID, "amount_cents": 900, "reference": fmt.Sprintf("race-%d", i),
			}).Status()

			mu.Lock()
			statuses[status]++
			mu.Unlock()
		})
	}

	close(start)
	wg.Wait()

	if statuses[http.StatusSeeOther]+statuses[http.StatusFound]+statuses[http.StatusOK] != 1 ||
		statuses[http.StatusUnprocessableEntity] != workers-1 {
		t.Fatalf("statuses = %v; want one success and %d 422", statuses, workers-1)
	}

	if got := f.balance(t); got != 0 {
		t.Fatalf("balance = %d, want 0", got)
	}

	var payouts int
	if err := f.svc.DB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM payouts").Scan(&payouts); err != nil || payouts != 1 {
		t.Fatalf("payouts = %d, %v", payouts, err)
	}
}

// mergeEnv adds payEnv's variables to opts (e.g. the Postgres options).
func mergeEnv(opts vukemtest.Options, env map[string]string) vukemtest.Options {
	o := opts
	o.Env = maps.Clone(opts.Env)

	if o.Env == nil {
		o.Env = map[string]string{}
	}

	maps.Copy(o.Env, env)

	return o
}
