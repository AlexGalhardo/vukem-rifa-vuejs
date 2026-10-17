package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// numbersFixture is an owner's Inertia client with one raffle (100
// numbers, R$ 10) on a fake clock.
type numbersFixture struct {
	app    *vukemtest.App
	clock  *vukemtest.FakeClock
	c      *vukemtest.Client
	raffle models.Raffle
	path   string
	svc    *reservations.Service
}

func newNumbersFixture(t *testing.T, opts ...vukemtest.Options) numbersFixture {
	t.Helper()

	// Real start time: the test cookie jar drops cookies expired by the wall clock.
	clock := vukemtest.NewFakeClock(time.Now().UTC().Truncate(time.Second))

	var o vukemtest.Options
	if len(opts) > 0 {
		o = opts[0]
	}

	o.Clock = clock.Now
	app := vukemtest.New(t, bootstrap.New, o)
	c := owner(t, app, "dono@example.com")
	id := createRaffle(t, app, c, validRaffle())

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	r, err := orm.QueryOf[models.Raffle](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	return numbersFixture{
		app: app, clock: clock, c: c, raffle: r, path: fmt.Sprintf("/raffles/%d", id),
		svc: reservations.New(db, clock.Now, app.Queue),
	}
}

func buyerForm() map[string]any {
	return map[string]any{"buyer_name": "Maria Silva", "buyer_email": "Maria@Example.com", "buyer_phone": "(11) 98765-4321"}
}

// postLink calls the link endpoint the way the page does (fetch, JSON).
func postLink(c *vukemtest.Client, path string, number any, body map[string]any) *vukemtest.Response {
	res := c.WithHeader("Accept", "application/json").PostJSON(fmt.Sprintf("%s/numbers/%v/link", path, number), body)
	c.WithHeader("Accept", "text/html, application/xhtml+xml")

	return res
}

func (f numbersFixture) show(t *testing.T) *vukemtest.InertiaPage {
	t.Helper()

	return f.c.Get(f.path).AssertOK().AssertInertia("Raffles/Show")
}

func TestNumbers_LinkIsShownOnceAndBuyerOnlyToOwner(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	res := postLink(f.c, f.path, 7, buyerForm()).AssertCreated().
		AssertJSONPath("number", float64(7)).
		AssertJSONPath("buyer_phone", "+5511987654321")

	raw, _ := res.JSONPath("url")
	url, _ := raw.(string)

	token, ok := strings.CutPrefix(url, "http://localhost:3000/pay/")
	if !ok || len(token) < 43 {
		t.Fatalf("url = %q, want APP_URL/pay/<token>", url)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 7, "token_hash": reservations.HashToken(token)})
	vukemtest.AssertDatabaseMissing(t, f.app, "reservations", map[string]any{"token_hash": token})

	page := f.c.Get(f.path).AssertOK()
	// No token prop or column (shared catalogs may say "token" in prose).
	page.AssertDontSee(token).AssertDontSee(reservations.HashToken(token)).AssertDontSee(`\"token\"`).AssertDontSee("token_hash")
	page.AssertInertia("Raffles/Show").
		PropEquals("numbers.6.status", "reserved").
		HasProp("buyers", 1).
		PropEquals("buyers.0.number", float64(7)).
		PropEquals("buyers.0.status", "pending").
		PropEquals("buyers.0.buyer_name", "Maria Silva").
		PropEquals("buyers.0.buyer_email", "maria@example.com").
		PropEquals("buyers.0.buyer_phone", "+5511987654321").
		PropEquals("buyers.0.expires_at", f.clock.Now().Add(10*time.Minute).Format(time.RFC3339)).
		PropEquals("draw", nil)

	pub := f.app.Client(t).Get("/r/" + f.raffle.Slug).AssertOK()
	pub.AssertDontSee("Maria").AssertDontSee("maria@example.com").AssertDontSee("5511987654321")
	pub.AssertInertia("Raffles/Public").PropEquals("statuses.6", "reserved").MissingProp("buyers")

	// The number is taken until the link ends.
	postLink(f.c, f.path, 7, buyerForm()).AssertStatus(409)
}

func TestNumbers_LinkValidatesBuyer(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	bad := buyerForm()
	bad["buyer_phone"], bad["buyer_email"], bad["buyer_name"] = "123", "nope", "M"

	postLink(f.c, f.path, 1, bad).AssertStatus(422).AssertJSONValidationErrors("buyer_name", "buyer_email", "buyer_phone")
	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
	f.show(t).PropEquals("numbers.0.status", "available")
}

func TestNumbers_OwnerCancelsLink(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	postLink(f.c, f.path, 9, buyerForm()).AssertCreated()

	f.c.PostJSON(f.path+"/numbers/9/cancel", nil).AssertRedirect(f.path)
	f.show(t).PropEquals("numbers.8.status", "available").HasProp("buyers", 0)
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 9, "status": "cancelled"})

	f.c.PostJSON(f.path+"/numbers/9/cancel", nil).AssertStatus(409)

	// A new link can be generated right away.
	postLink(f.c, f.path, 9, buyerForm()).AssertCreated()
}

// PLAN Phase 3: after ten minutes the sweeper frees the number and the
// manage page shows it available again.
func TestNumbers_LinkExpiresAfterTenMinutes(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	postLink(f.c, f.path, 3, buyerForm()).AssertCreated()

	s, err := f.app.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(10*time.Minute - time.Second)

	if err := s.RunDue(t.Context(), f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	f.show(t).PropEquals("numbers.2.status", "reserved").HasProp("buyers", 1)

	f.clock.Advance(time.Second)

	if err := s.RunDue(t.Context(), f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	f.show(t).PropEquals("numbers.2.status", "available").HasProp("buyers", 0)
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 3, "status": "expired"})

	postLink(f.c, f.path, 3, buyerForm()).AssertCreated()
}

func TestNumbers_PaidNumberShowsBuyerToOwner(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	sell(t, f.app, f.svc, f.raffle, 5, buyer)

	f.show(t).PropEquals("numbers.4.status", "paid").
		PropEquals("buyers.0.status", "paid").
		PropEquals("buyers.0.buyer_name", buyer.Name).
		HasProp("buyers.0.paid_at")

	f.app.Client(t).Get("/r/" + f.raffle.Slug).AssertOK().AssertDontSee(buyer.Email).AssertDontSee(buyer.Name)
}

// sell generates a link for number and marks it paid, as the payment
// webhook will (payments stream).
func sell(t *testing.T, app *vukemtest.App, svc *reservations.Service, r models.Raffle, number int, b requests.Buyer) {
	t.Helper()

	link, err := svc.Reserve(t.Context(), r, number, b)
	if err != nil {
		t.Fatal(err)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	now := app.Now().UTC()
	p := db.Dialect().Placeholder

	if _, err := db.ExecContext(t.Context(), "UPDATE reservations SET status = 'paid', paid_at = "+p(1)+" WHERE id = "+p(2),
		now, link.Reservation.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), "UPDATE raffle_numbers SET status = 'paid', paid_at = "+p(1)+" WHERE id = "+p(2),
		now, link.Reservation.NumberID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), "UPDATE raffles SET locked_at = "+p(1)+" WHERE id = "+p(2)+" AND locked_at IS NULL",
		now, r.ID); err != nil {
		t.Fatal(err)
	}
}
