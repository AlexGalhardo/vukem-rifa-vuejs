package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/routes"
)

// OWASP A01/A04/A06 abuse cases for payment-link generation.

func TestSecurity_NumbersOtherOwnersGet404(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	postLink(f.c, f.path, 2, buyerForm()).AssertCreated()

	bob := owner(t, f.app, "bob@example.com")
	postLink(bob, f.path, 1, buyerForm()).AssertNotFound()
	bob.PostJSON(f.path+"/numbers/2/cancel", nil).AssertNotFound()

	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 1)
	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{"number": 2, "status": "pending"})
}

func TestSecurity_NumbersCrossSitePostBlocked(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	for _, h := range [][2]string{{"Origin", "https://evil.example"}, {"Sec-Fetch-Site", "cross-site"}} {
		f.c.WithHeader(h[0], h[1])
		postLink(f.c, f.path, 1, buyerForm()).AssertForbidden()
		f.c.PostJSON(f.path+"/numbers/1/cancel", nil).AssertForbidden()
		f.c.WithHeader(h[0], "")
	}

	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
}

func TestSecurity_NumbersGuestsAndUnverifiedBlocked(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	guest := f.app.Client(t).Inertia("")
	guest.PostJSON(f.path+"/numbers/1/link", buyerForm()).AssertRedirect("/login")
	guest.PostJSON(f.path+"/numbers/1/cancel", nil).AssertRedirect("/login")

	unverified := f.app.Client(t).Inertia("")
	register(t, unverified, "new@example.com")
	unverified.PostJSON(f.path+"/numbers/1/link", buyerForm()).AssertRedirect("/email/verify")

	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
}

func TestSecurity_NumbersOnlyWhileOnSale(t *testing.T) {
	t.Parallel()

	t.Run("ended", func(t *testing.T) {
		t.Parallel()

		f := newNumbersFixture(t)
		// Moving the deadline instead of the clock keeps the session alive.
		if _, err := f.svc.DB.ExecContext(t.Context(), "UPDATE raffles SET ends_at = ? WHERE id = ?",
			f.clock.Now().Add(-time.Second), f.raffle.ID); err != nil {
			t.Fatal(err)
		}

		postLink(f.c, f.path, 1, buyerForm()).AssertStatus(409)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()

		f := newNumbersFixture(t)
		f.c.PostJSON(f.path+"/archive", nil).AssertRedirect(f.path)
		postLink(f.c, f.path, 1, buyerForm()).AssertStatus(409)
		vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
	})

	t.Run("out of range", func(t *testing.T) {
		t.Parallel()

		f := newNumbersFixture(t)
		for _, n := range []string{"0", "101", "-1", "99999999999999999999", "abc"} {
			postLink(f.c, f.path, n, buyerForm()).AssertNotFound()
			f.c.PostJSON(f.path+"/numbers/"+n+"/cancel", nil).AssertNotFound()
		}

		vukemtest.AssertDatabaseCount(t, f.app, "reservations", 0)
	})
}

func TestSecurity_NumbersLinkRateLimitedPerOwner(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	for n := 1; n <= routes.LinksPerMinute; n++ {
		postLink(f.c, f.path, n, buyerForm()).AssertCreated()
	}

	postLink(f.c, f.path, routes.LinksPerMinute+1, buyerForm()).AssertStatus(429).AssertHeader("Retry-After", "2")

	// Another owner has their own bucket.
	bob := owner(t, f.app, "bob@example.com")
	id := createRaffle(t, f.app, bob, validRaffle())
	postLink(bob, fmt.Sprintf("/raffles/%d", id), 1, buyerForm()).AssertCreated()
}
