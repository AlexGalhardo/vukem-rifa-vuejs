package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Owner request: anyone with the public raffle link can buy a free number
// without an account (name, email, phone, LGPD consent), then pays on the
// same /pay/{token} page as owner-generated links.

func publicBuyer(email string) map[string]any {
	return map[string]any{
		"buyer_name": "Paula Pública", "buyer_email": email, "buyer_phone": "(31) 99876-5432",
		"consent": true, "website": "",
	}
}

func (f numbersFixture) publicReserve(c *vukemtest.Client, number int, body map[string]any) *vukemtest.Response {
	return c.WithHeader("Referer", "/r/"+f.raffle.Slug).
		PostJSON(fmt.Sprintf("/r/%s/numbers/%d/reserve", f.raffle.Slug, number), body)
}

func payPathOf(t *testing.T, res *vukemtest.Response) string {
	t.Helper()

	loc := res.Header("Location")
	if !strings.HasPrefix(loc, "/pay/") || len(loc) < len("/pay/")+43 {
		t.Fatalf("status %d, Location %q; want a redirect to /pay/<token>; body %.300s", res.Status(), loc, res.Body())
	}

	return loc
}

func TestPublicCheckout_FreeNumberGoesToThePayPage(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	visitor := f.app.Client(t).Inertia("")

	page := visitor.Get("/r/" + f.raffle.Slug).AssertOK().AssertInertia("Raffles/Public")
	page.PropEquals("can_buy", true)

	pay := payPathOf(t, f.publicReserve(visitor, 12, publicBuyer("Paula@Example.com")))
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{
		"number": 12, "origin": "public", "buyer_email": "paula@example.com", "status": "pending",
	})
	// LGPD: the consent is recorded with the version of the notice shown.
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 12, "consent_version": controllers.PublicConsentVersion})
	vukemtest.AssertDatabaseMissing(t, f.app, "reservations", map[string]any{"number": 12, "consent_at": nil})

	visitor.Get(pay).AssertOK().AssertInertia("Pay/Show").PropEquals("state", "live").PropEquals("checkout.number", float64(12))

	// The owner sees where the link came from; the public page still shows statuses only.
	f.show(t).PropEquals("buyers.0.origin", "public").PropEquals("numbers.11.status", "reserved")
	pub := f.app.Client(t).Get("/r/" + f.raffle.Slug)
	pub.AssertDontSee("paula@example.com").AssertDontSee("Paula Pública").AssertDontSee(strings.TrimPrefix(pay, "/pay/"))
}

func TestPublicCheckout_OnlyFreeNumbers(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	postLink(f.c, f.path, 3, buyerForm()).AssertCreated()
	markPaid(t, f.app, f.raffle.ID, 4)

	visitor := f.app.Client(t).Inertia("")
	for _, n := range []int{3, 4} {
		f.publicReserve(visitor, n, publicBuyer("x@example.com")).AssertRedirect("/r/" + f.raffle.Slug)
		visitor.Get("/r/" + f.raffle.Slug).AssertInertia("Raffles/Public").HasProp("errors.number")
	}

	f.publicReserve(visitor, 999, publicBuyer("x@example.com")).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 1)
}

func TestPublicCheckout_RefusesBotsAndMissingConsent(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	visitor := f.app.Client(t).Inertia("")

	bot := publicBuyer("bot@example.com")
	bot["website"] = "http://spam.example"
	f.publicReserve(visitor, 1, bot).AssertRedirect("/r/" + f.raffle.Slug)

	noConsent := publicBuyer("no@example.com")
	noConsent["consent"] = false
	f.publicReserve(visitor, 2, noConsent).AssertRedirect("/r/" + f.raffle.Slug)
	visitor.Get("/r/" + f.raffle.Slug).AssertInertia("Raffles/Public").HasProp("errors.consent")

	bad := publicBuyer("not-an-email")
	f.publicReserve(visitor, 3, bad).AssertRedirect("/r/" + f.raffle.Slug)

	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
}

func TestPublicCheckout_ClosedRafflesRefuse(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	f.c.PostJSON(f.path+"/archive", nil)

	visitor := f.app.Client(t).Inertia("")
	visitor.Get("/r/"+f.raffle.Slug).AssertInertia("Raffles/Public").PropEquals("can_buy", false)
	f.publicReserve(visitor, 1, publicBuyer("late@example.com")).AssertRedirect("/r/" + f.raffle.Slug)
	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
}

func TestPublicCheckout_CapsLiveLinksPerEmail(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	for n := 1; n <= 5; n++ {
		payPathOf(t, f.publicReserve(f.app.Client(t).Inertia(""), n, publicBuyer("greedy@example.com")))
	}

	f.publicReserve(f.app.Client(t).Inertia(""), 6, publicBuyer("GREEDY@example.com")).AssertRedirect("/r/" + f.raffle.Slug)
	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 5)

	// After the links expire the email can buy again.
	f.clock.Advance(11 * time.Minute)
	if _, err := f.svc.ExpireDue(t.Context()); err != nil {
		t.Fatal(err)
	}

	payPathOf(t, f.publicReserve(f.app.Client(t).Inertia(""), 6, publicBuyer("greedy@example.com")))
}

func TestPublicCheckout_RateLimitedPerIP(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	visitor := f.app.Client(t).Inertia("")

	limited := false

	for n := 1; n <= 12; n++ {
		res := f.publicReserve(visitor, n, publicBuyer(fmt.Sprintf("ip%d@example.com", n)))
		if res.Status() == 429 {
			limited = true
			break
		}
	}

	if !limited {
		t.Fatal("twelve public checkouts from one IP were never rate limited")
	}
}

// History linking: a signed-in buyer, or a verified account with the buyer
// email, gets the reservation; an unverified account only after verifying.
func TestPublicCheckout_LinksTheBuyerAccount(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	userID := func(email string) int64 {
		var id int64
		if err := db.QueryRowContext(t.Context(), "SELECT id FROM users WHERE email = ?", email).Scan(&id); err != nil {
			t.Fatal(err)
		}

		return id
	}

	buyerOf := func(number int) any {
		var id any
		if err := db.QueryRowContext(t.Context(), "SELECT buyer_user_id FROM reservations WHERE raffle_id = ? AND number = ?",
			f.raffle.ID, number).Scan(&id); err != nil {
			t.Fatal(err)
		}

		return id
	}

	// Signed in (even with another buyer email): linked to the session user.
	signedIn := owner(t, f.app, "comprador@example.com")
	payPathOf(t, f.publicReserve(signedIn, 1, publicBuyer("outro@example.com")))
	if got := buyerOf(1); fmt.Sprint(got) != fmt.Sprint(userID("comprador@example.com")) {
		t.Fatalf("signed-in buyer = %v", got)
	}

	// Anonymous with a verified account's email: linked.
	payPathOf(t, f.publicReserve(f.app.Client(t).Inertia(""), 2, publicBuyer("comprador@example.com")))
	if got := buyerOf(2); fmt.Sprint(got) != fmt.Sprint(userID("comprador@example.com")) {
		t.Fatalf("verified email buyer = %v", got)
	}

	// Anonymous with an unverified account's email: not linked until it verifies.
	unverified := f.app.Client(t).Inertia("")
	register(t, unverified, "novo@example.com")
	payPathOf(t, f.publicReserve(f.app.Client(t).Inertia(""), 3, publicBuyer("novo@example.com")))

	if got := buyerOf(3); got != nil {
		t.Fatalf("unverified email buyer = %v, want NULL", got)
	}

	unverified.Get(link(t, f.app, "novo@example.com")).AssertRedirect("/dashboard")

	if got := buyerOf(3); fmt.Sprint(got) != fmt.Sprint(userID("novo@example.com")) {
		t.Fatalf("after verification buyer = %v", got)
	}
}

func TestPublicCheckout_PurchaseHistoryShowsOnlyMyPurchases(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	me := owner(t, f.app, "eu@example.com")
	payPathOf(t, f.publicReserve(me, 7, publicBuyer("eu@example.com")))
	payPathOf(t, f.publicReserve(me, 8, publicBuyer("eu@example.com")))
	markPaid(t, f.app, f.raffle.ID, 7)

	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), "UPDATE reservations SET status = 'paid', paid_at = ? WHERE number = 7", f.clock.Now()); err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(11 * time.Minute)
	if _, err := f.svc.ExpireDue(t.Context()); err != nil {
		t.Fatal(err)
	}

	other := owner(t, f.app, "outra@example.com")
	payPathOf(t, f.publicReserve(other, 9, publicBuyer("outra@example.com")))

	page := me.Get("/account/purchases").AssertOK().AssertInertia("Account/Purchases").
		HasProp("paid", 1).HasProp("unpaid", 1)
	page.PropEquals("paid.0.number", float64(7)).PropEquals("paid.0.status", models.ReservationPaid).
		PropEquals("paid.0.raffle_title", "Rifa da moto").PropEquals("paid.0.amount_cents", float64(1000)).
		PropEquals("paid.0.raffle_slug", f.raffle.Slug).
		PropEquals("unpaid.0.number", float64(8)).PropEquals("unpaid.0.status", models.ReservationExpired)
	me.Get("/account/purchases").AssertDontSee("outra@example.com")

	f.app.Client(t).Get("/account/purchases").AssertRedirect("/login")
}

// Security review fix: a raffle holds at most max(5, 20%) live public links,
// whoever asks, so nobody can lock up the whole grid; the refusal is the
// same generic message for every cap and limit.
func TestPublicCheckout_GlobalCapOfLivePublicLinks(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t) // 100 numbers: at most 20 live public links

	// Twenty buyers (from many IPs, so straight through the service).
	for n := 1; n <= 20; n++ {
		b := buyer
		b.Email = fmt.Sprintf("crowd%d@example.com", n)
		if _, err := f.svc.ReserveWith(t.Context(), f.raffle, n, b, reservations.ReserveOptions{
			Origin: models.OriginPublic, MaxLivePublic: controllers.MaxLivePublicLinks(f.raffle.NumberCount),
		}); err != nil {
			t.Fatal(err)
		}
	}

	visitor := f.app.Client(t).Inertia("")
	f.publicReserve(visitor, 21, publicBuyer("late@example.com")).AssertRedirect("/r/" + f.raffle.Slug)
	visitor.Get("/r/"+f.raffle.Slug).AssertInertia("Raffles/Public").
		PropEquals("errors.number", "We could not reserve this number right now. Please try again later.")

	// Owner links are not counted.
	postLink(f.c, f.path, 22, buyerForm()).AssertCreated()
}

func TestPublicCheckout_EmailCapIgnoresPlusTagsAndGmailDots(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	for n, email := range []string{"ana.maria@gmail.com", "anamaria+1@gmail.com", "a.n.a.maria@googlemail.com", "anamaria+x@gmail.com", "AnaMaria@gmail.com"} {
		payPathOf(t, f.publicReserve(f.app.Client(t).Inertia(""), n+1, publicBuyer(email)))
	}

	visitor := f.app.Client(t).Inertia("")
	f.publicReserve(visitor, 9, publicBuyer("ana.maria+sixth@gmail.com")).AssertRedirect("/r/" + f.raffle.Slug)
	visitor.Get("/r/"+f.raffle.Slug).AssertInertia("Raffles/Public").
		PropEquals("errors.number", "We could not reserve this number right now. Please try again later.")
}
