//go:build integration

package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Twenty workers (job retries, sweeper, sold-out signal) racing to draw the
// same raffle: exactly one draw.
func TestPostgres_ConcurrentDrawRunsOnce(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t, postgresEnv(t))
	for _, n := range []int{5, 50, 95} {
		sell(t, f.app, f.svc, f.raffle, n, buyer)
	}

	f.moveDeadline(t, 0)

	const workers = 20

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		wins int
	)

	start := make(chan struct{})
	svc := f.draws()

	for range workers {
		wg.Go(func() {
			<-start

			_, ok, err := svc.Run(t.Context(), f.raffle.ID, models.DrawDeadline)
			if err != nil {
				t.Errorf("Run: %v", err)
			}

			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		})
	}

	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("draws won = %d, want exactly 1", wins)
	}

	vukemtest.AssertDatabaseCount(t, f.app, "draws", 1)

	if got := f.status(t); got != models.RaffleDrawn {
		t.Fatalf("status = %q", got)
	}
}

// PLAN Phase 3 (D15): "serve" runs the database queue worker in-process
// (vukem QUEUE_IN_SERVE) and it processes the delayed ExpireReservation
// and RunDraw jobs.
func TestPostgres_BackgroundWorkerProcessesDelayedJobs(t *testing.T) {
	t.Parallel()

	env := postgresEnv(t)
	env.AppOptions = []vukem.Option{vukem.WithKind(vukem.KindWeb)}
	f := newNumbersFixture(t, env)
	sell(t, f.app, f.svc, f.raffle, 1, buyer)
	postLink(f.c, f.path, 2, buyerForm()).AssertCreated()

	db := f.svc.DB
	ctx := t.Context()

	var queued int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE kind IN ('reservation.expire','raffle.draw')").
		Scan(&queued); err != nil || queued != 3 { // 2 link expiries + the deadline draw
		t.Fatalf("queued jobs = %d, %v; want 3", queued, err)
	}

	// The jobs are not due yet: the worker leaves them alone.
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)

	go func() { done <- f.app.Run(runCtx) }()

	t.Cleanup(func() {
		stop()

		if err := <-done; err != nil {
			t.Errorf("app.Run: %v", err)
		}
	})

	time.Sleep(1500 * time.Millisecond)
	f.show(t).PropEquals("numbers.1.status", "reserved")

	// Past the deadline: the queue follows app.Now (vukem v0.25.0), so both
	// delayed jobs (the link expiry and the deadline draw) become due on the
	// fake clock and the in-process worker runs them.
	f.clock.Advance(31 * 24 * time.Hour)

	waitFor(t, 15*time.Second, func() bool {
		var draws, live int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM draws WHERE raffle_id = $1", f.raffle.ID).Scan(&draws)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reservations WHERE raffle_id = $1 AND status = 'pending'", f.raffle.ID).Scan(&live)

		return draws == 1 && live == 0
	})

	// The session expired with the clock jump: check the database.
	vukemtest.AssertDatabaseHas(t, f.app, "raffle_numbers", map[string]any{"raffle_id": f.raffle.ID, "number": 2, "status": "available"})
	vukemtest.AssertDatabaseHas(t, f.app, "draws", map[string]any{"raffle_id": f.raffle.ID, "winning_number": 1})
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 2, "status": "expired"})
}

// The every-minute sweepers on Postgres: due links expire, overdue raffles
// are drawn.
func TestPostgres_SweepersExpireLinksAndDraw(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t, postgresEnv(t))
	sell(t, f.app, f.svc, f.raffle, 7, buyer)
	postLink(f.c, f.path, 8, buyerForm()).AssertCreated()

	s, err := f.app.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(10 * time.Minute)

	if err := s.RunDue(t.Context(), f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	f.show(t).PropEquals("numbers.7.status", "available").PropEquals("raffle.status", "active")

	f.moveDeadline(t, 0)
	f.clock.Advance(time.Minute)

	if err := s.RunDue(t.Context(), f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	f.show(t).PropEquals("raffle.status", "drawn").PropEquals("draw.winning_number", float64(7))
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Owner rule on Postgres: an edit cancels the live links in its transaction
// (CHECKs and the one-live-link index hold), a paid number still locks the
// price, and the refused edit cancels nothing.
func TestPostgres_EditCancelsLiveLinks(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t, postgresEnv(t))
	postLink(f.c, f.path, 2, buyerForm()).AssertCreated()
	postLink(f.c, f.path, 3, buyerForm()).AssertCreated()

	fields := validRaffle()
	fields["title"] = "Rifa editada"
	f.c.PutJSON(f.path, fields).AssertRedirect(f.path)
	vukemtest.AssertDatabaseMissing(t, f.app, "reservations", map[string]any{"status": "pending"})
	vukemtest.AssertDatabaseMissing(t, f.app, "raffle_numbers", map[string]any{"raffle_id": f.raffle.ID, "status": "reserved"})

	// The numbers can be sold again right away (one live link per number).
	postLink(f.c, f.path, 2, buyerForm()).AssertCreated()

	markPaidPG(t, f, 1)

	fields["price_reais"] = 99
	f.c.WithHeader("Referer", f.path+"/edit").PutJSON(f.path, fields).AssertRedirect(f.path + "/edit")
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 2, "status": "pending"})
}

func markPaidPG(t *testing.T, f numbersFixture, number int) {
	t.Helper()

	now := f.app.Now().UTC()
	if _, err := f.svc.DB.ExecContext(t.Context(), "UPDATE raffle_numbers SET status = 'paid', paid_at = $1 WHERE raffle_id = $2 AND number = $3",
		now, f.raffle.ID, number); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.DB.ExecContext(t.Context(), "UPDATE raffles SET locked_at = $1 WHERE id = $2", now, f.raffle.ID); err != nil {
		t.Fatal(err)
	}
}
