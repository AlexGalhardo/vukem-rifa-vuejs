package tests

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"testing"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/jobs"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/database/seeders"
)

const thirtyDays = 30 * 24 * time.Hour

func (f numbersFixture) draws() *draws.Service { return draws.New(f.svc.DB, f.clock.Now) }

func (f numbersFixture) status(t *testing.T) string {
	t.Helper()

	r, err := orm.QueryOf[models.Raffle](f.svc.DB).Find(t.Context(), f.raffle.ID)
	if err != nil {
		t.Fatal(err)
	}

	return r.Status
}

// moveDeadline puts ends_at at now+d: tests reach the deadline without
// advancing the clock past the session lifetime.
func (f numbersFixture) moveDeadline(t *testing.T, d time.Duration) {
	t.Helper()

	p := f.svc.DB.Dialect().Placeholder
	ends := f.clock.Now().Add(d)
	// Postgres checks ends_at > starts_at.
	if _, err := f.svc.DB.ExecContext(t.Context(), "UPDATE raffles SET starts_at = "+p(1)+", ends_at = "+p(2)+" WHERE id = "+p(3),
		ends.Add(-thirtyDays), ends, f.raffle.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDraw_DeadlinePicksAmongPaidNumbers(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	for _, n := range []int{12, 3, 7} {
		sell(t, f.app, f.svc, f.raffle, n, requests.Buyer{Name: "Comprador", Email: "c@example.com", Phone: "+5511987654321"})
	}

	link, err := f.svc.Reserve(t.Context(), f.raffle, 20, buyer)
	if err != nil {
		t.Fatal(err)
	}

	svc := f.draws()

	if _, ok, err := svc.Run(t.Context(), f.raffle.ID, models.DrawDeadline); err != nil || ok {
		t.Fatalf("Run before the deadline = %v, %v; want a no-op", ok, err)
	}

	f.clock.Advance(thirtyDays)

	d, ok, err := svc.Run(t.Context(), f.raffle.ID, models.DrawDeadline)
	if err != nil || !ok {
		t.Fatalf("Run = %v, %v", ok, err)
	}

	seed, err := hex.DecodeString(d.Seed)
	if err != nil || len(seed) != draws.SeedSize {
		t.Fatalf("seed %q: %v", d.Seed, err)
	}

	want, _ := draws.Pick(seed, []int{3, 7, 12})
	if d.WinningNumber == nil || *d.WinningNumber != want || !slices.Contains([]int{3, 7, 12}, want) {
		t.Fatalf("winner = %v, want %d (recomputed)", d.WinningNumber, want)
	}

	if d.ParticipantsCount != 3 || d.ParticipantsSHA256 != draws.ParticipantsHash([]int{3, 7, 12}) ||
		d.Algorithm != "v1" || d.Trigger != models.DrawDeadline || !d.DrawnAt.Equal(f.clock.Now().UTC()) {
		t.Fatalf("draw = %+v", d)
	}

	if got := f.status(t); got != models.RaffleDrawn {
		t.Fatalf("raffle status = %q, want drawn", got)
	}

	// The live link ended with the raffle: nothing can be paid afterwards.
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"id": link.Reservation.ID, "status": "expired"})
	vukemtest.AssertDatabaseHas(t, f.app, "raffle_numbers", map[string]any{"raffle_id": f.raffle.ID, "number": 20, "status": "available"})

	// Idempotent: a second run (retry, sweeper, sold-out signal) is a no-op.
	for _, trigger := range []string{models.DrawDeadline, models.DrawSoldOut} {
		if _, ok, err := svc.Run(t.Context(), f.raffle.ID, trigger); err != nil || ok {
			t.Fatalf("second Run(%s) = %v, %v", trigger, ok, err)
		}
	}

	vukemtest.AssertDatabaseCount(t, f.app, "draws", 1)
}

func TestDraw_ZeroPaidClosesWithoutWinner(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	f.clock.Advance(thirtyDays)

	d, ok, err := f.draws().Run(t.Context(), f.raffle.ID, models.DrawDeadline)
	if err != nil || !ok || d.WinningNumber != nil || d.ParticipantsCount != 0 ||
		d.ParticipantsSHA256 != draws.ParticipantsHash(nil) {
		t.Fatalf("Run = %+v, %v, %v", d, ok, err)
	}

	if got := f.status(t); got != models.RaffleClosed {
		t.Fatalf("raffle status = %q, want closed", got)
	}
}

func TestDraw_SoldOutNeedsEveryNumberPaid(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	for n := 1; n < f.raffle.NumberCount; n++ {
		markPaid(t, f.app, f.raffle.ID, n)
	}

	svc := f.draws()
	if _, ok, err := svc.Run(t.Context(), f.raffle.ID, models.DrawSoldOut); err != nil || ok {
		t.Fatalf("Run with one number unpaid = %v, %v; want a no-op", ok, err)
	}

	markPaid(t, f.app, f.raffle.ID, f.raffle.NumberCount)

	d, ok, err := svc.Run(t.Context(), f.raffle.ID, models.DrawSoldOut)
	if err != nil || !ok || d.Trigger != models.DrawSoldOut || d.ParticipantsCount != 100 {
		t.Fatalf("Run = %+v, %v, %v", d, ok, err)
	}

	if _, _, err := svc.Run(t.Context(), f.raffle.ID, "manual"); !errors.Is(err, draws.ErrUnknownTrigger) {
		t.Fatalf("unknown trigger: %v", err)
	}
}

// Archived raffles were cancelled by their owner: never drawn, even past
// the deadline.
func TestDraw_ArchivedRafflesAreNotDrawn(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	markPaid(t, f.app, f.raffle.ID, 1)
	f.c.PostJSON(f.path+"/archive", nil).AssertRedirect(f.path)
	f.clock.Advance(thirtyDays)

	if _, ok, err := f.draws().Run(t.Context(), f.raffle.ID, models.DrawDeadline); err != nil || ok {
		t.Fatalf("Run = %v, %v; want a no-op", ok, err)
	}

	if n, err := f.draws().RunDue(t.Context()); err != nil || n != 0 {
		t.Fatalf("RunDue = %d, %v", n, err)
	}

	vukemtest.AssertDatabaseCount(t, f.app, "draws", 0)

	if got := f.status(t); got != models.RaffleArchived {
		t.Fatalf("status = %q", got)
	}
}

// The every-minute sweeper draws overdue and sold-out raffles whose job
// was lost.
func TestDraw_SweeperRunsDueRaffles(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	markPaid(t, f.app, f.raffle.ID, 4)

	other := owner(t, f.app, "outro@example.com")
	soldOutID := createRaffle(t, f.app, other, map[string]any{
		"title": "Rifa pequena", "description": "", "number_count": 10, "price_reais": 5, "duration_days": 30,
	})

	for n := 1; n <= 10; n++ {
		markPaid(t, f.app, soldOutID, n)
	}

	s, err := f.app.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	f.moveDeadline(t, time.Minute)

	if err := s.RunDue(t.Context(), f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "draws", map[string]any{"raffle_id": soldOutID, "trigger": "sold_out"})
	vukemtest.AssertDatabaseCount(t, f.app, "draws", 1)

	f.clock.Advance(time.Minute)

	if err := s.RunDue(t.Context(), f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "draws", map[string]any{"raffle_id": f.raffle.ID, "trigger": "deadline", "winning_number": 4})

	// Owner page: winner's contact; public page: result without personal data.
	f.show(t).PropEquals("raffle.status", "drawn").PropEquals("draw.winning_number", float64(4)).
		PropEquals("winner", nil) // markPaid has no buyer; sold numbers do (TestDraw_ResultPages)
}

func TestDraw_ResultPages(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	sell(t, f.app, f.svc, f.raffle, 8, buyer)
	f.moveDeadline(t, 0)

	d, ok, err := f.draws().Run(t.Context(), f.raffle.ID, models.DrawDeadline)
	if err != nil || !ok {
		t.Fatalf("Run = %v, %v", ok, err)
	}

	f.show(t).PropEquals("draw.winning_number", float64(8)).
		PropEquals("draw.seed", d.Seed).
		PropEquals("winner.buyer_name", buyer.Name).
		PropEquals("winner.buyer_email", buyer.Email).
		PropEquals("winner.buyer_phone", buyer.Phone)

	pub := f.app.Client(t).Get("/r/" + f.raffle.Slug).AssertOK()
	pub.AssertDontSee(buyer.Email).AssertDontSee(buyer.Name).AssertDontSee(buyer.Phone)
	pub.AssertInertia("Raffles/Public").
		PropEquals("raffle.status", "drawn").
		PropEquals("draw.winning_number", float64(8)).
		PropEquals("draw.seed", d.Seed).
		PropEquals("draw.participants_count", float64(1)).
		PropEquals("draw.participants_sha256", draws.ParticipantsHash([]int{8})).
		PropEquals("draw.algorithm", "v1").
		PropEquals("draw.trigger", "deadline").
		HasProp("draw.drawn_at").
		MissingProp("winner").
		MissingProp("buyers")
}

// Creating a raffle queues its deadline draw; changing the duration queues
// a new one (the old job finds the deadline not due and does nothing).
func TestDraw_JobQueuedAtDeadline(t *testing.T) {
	t.Parallel()

	fake := queue.NewFake()
	f := newNumbersFixture(t, vukemtest.Options{AppOptions: []vukem.Option{vukem.WithQueueDriver(fake)}})

	queued := queue.Dispatched[jobs.RunDraw](fake)
	if len(queued) != 1 || queued[0].RaffleID != f.raffle.ID || queued[0].Trigger != models.DrawDeadline {
		t.Fatalf("queued draws = %+v", queued)
	}

	// The queue delays from the wall clock; the fixture's clock starts at it.
	at := fake.Messages()[len(fake.Messages())-1].AvailableAt
	if d := at.Sub(f.raffle.EndsAt); d < -time.Minute || d > time.Minute {
		t.Fatalf("draw job available at %v, deadline %v", at, f.raffle.EndsAt)
	}

	fields := validRaffle()
	fields["duration_days"] = 7
	f.c.PutJSON(f.path, fields).AssertRedirect(f.path)

	if queued = queue.Dispatched[jobs.RunDraw](fake); len(queued) != 2 {
		t.Fatalf("queued draws after a duration change = %d, want 2", len(queued))
	}

	fields["title"] = "Só o título"
	f.c.PutJSON(f.path, fields).AssertRedirect(f.path)

	if queued = queue.Dispatched[jobs.RunDraw](fake); len(queued) != 2 {
		t.Fatalf("queued draws after a title change = %d, want 2", len(queued))
	}
}

// The registered handler runs the draw (sync driver: dispatch runs it).
func TestDraw_JobHandler(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	markPaid(t, f.app, f.raffle.ID, 10)

	q, err := f.app.Queue()
	if err != nil {
		t.Fatal(err)
	}

	if err := q.Dispatch(t.Context(), jobs.RunDraw{RaffleID: f.raffle.ID, Trigger: models.DrawDeadline}); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseCount(t, f.app, "draws", 0)

	f.moveDeadline(t, 0)

	if err := q.Dispatch(t.Context(), jobs.RunDraw{RaffleID: f.raffle.ID, Trigger: models.DrawDeadline}); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "draws", map[string]any{"raffle_id": f.raffle.ID, "winning_number": 10})
}

// The e2e demo seeder: a drawn raffle among its paid numbers, idempotent.
func TestDraw_DemoSeeder(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	for range 2 {
		if err := app.Seed(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	vukemtest.AssertDatabaseCount(t, app, "draws", 1)

	page := app.Client(t).Get("/r/"+seeders.DrawDemoSlug).AssertOK().AssertDontSee("Pessoa Demo").
		AssertInertia("Raffles/Public").PropEquals("raffle.status", "drawn").
		PropEquals("draw.participants_count", float64(3)).
		PropEquals("draw.participants_sha256", draws.ParticipantsHash(seeders.DrawDemoPaid))

	w, _ := page.Prop("draw.winning_number")
	if !slices.ContainsFunc(seeders.DrawDemoPaid, func(n int) bool { return fmt.Sprint(w) == strconv.Itoa(n) }) {
		t.Fatalf("winner = %v, want one of %v", w, seeders.DrawDemoPaid)
	}
}
