//go:build integration

package tests

import (
	"fmt"
	"maps"
	"net/http"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
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

// Security review fix: a raffle edit racing a payment webhook must never
// deadlock (both lock the raffle first) nor change the price of a raffle
// whose number just got paid (Locked() is read from the locked row).
func TestPostgres_EditRacingAPaymentKeepsTheRules(t *testing.T) {
	t.Parallel()

	for i := range 6 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()

			env := postgresEnv(t)
			maps.Copy(env.Env, payEnv().Env)
			f := newReservationFixture(t, env)
			_, res := f.confirmed(t, 3)
			body := paidEvent(fmt.Sprintf("log_race_%d", i), *res.ChargeID, res.ID, 1000)

			svc := raffles.New(f.svc.DB, f.clock.Now, nil)
			edit := requests.Raffle{Title: f.raffle.Title, Description: f.raffle.Description, NumberCount: f.raffle.NumberCount, PriceReais: 99, DurationDays: f.raffle.DurationDays}

			var (
				wg      sync.WaitGroup
				editErr error
				status  int
			)

			start := make(chan struct{})

			wg.Go(func() {
				<-start

				r := f.raffle
				_, editErr = svc.Update(t.Context(), &r, edit)
			})
			wg.Go(func() {
				<-start

				status = deliverSigned(t, f.app, body).Status()
			})
			close(start)
			wg.Wait()

			if status != http.StatusOK {
				t.Fatalf("webhook = %d", status)
			}

			if editErr != nil && errs.Status(editErr) != http.StatusUnprocessableEntity {
				t.Fatalf("edit: %v (deadlock or 500)", editErr)
			}

			var price int
			var paid int
			if err := f.svc.DB.QueryRowContext(t.Context(), "SELECT price_reais FROM raffles WHERE id = $1", f.raffle.ID).Scan(&price); err != nil {
				t.Fatal(err)
			}

			if err := f.svc.DB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM raffle_numbers WHERE raffle_id = $1 AND status = 'paid'", f.raffle.ID).Scan(&paid); err != nil {
				t.Fatal(err)
			}

			switch {
			case paid == 1 && price != 10:
				t.Fatalf("a number was paid at R$ 10 but the price changed to %d", price)
			case paid == 0 && price != 99:
				t.Fatalf("no number paid but the edit did not apply (price %d, edit err %v)", price, editErr)
			}
		})
	}
}

// The deterministic half of the race: an edit built from a raffle read
// before a payment landed must still see the lock.
func TestPostgres_EditWithAStaleRaffleSeesTheNewLock(t *testing.T) {
	t.Parallel()

	env := postgresEnv(t)
	maps.Copy(env.Env, payEnv().Env)
	f := newReservationFixture(t, env)
	stale := f.raffle

	_, res := f.confirmed(t, 3)
	deliverSigned(t, f.app, paidEvent("log_stale", *res.ChargeID, res.ID, 1000)).AssertOK()

	in := requests.Raffle{Title: stale.Title, Description: stale.Description, NumberCount: stale.NumberCount, PriceReais: 99, DurationDays: stale.DurationDays}
	if _, err := raffles.New(f.svc.DB, f.clock.Now, nil).Update(t.Context(), &stale, in); errs.Status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("edit with a stale raffle = %v, want 422 (locked)", err)
	}
}
