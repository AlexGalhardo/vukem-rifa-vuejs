package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// Owner request: the public raffle page shows who organizes it, and the
// organizer may opt in to show an email, WhatsApp and Telegram contact.

func contactForm(over map[string]any) map[string]any {
	f := map[string]any{
		"show_email": false, "whatsapp": "", "show_whatsapp": false, "telegram": "", "show_telegram": false,
	}
	for k, v := range over {
		f[k] = v
	}

	return f
}

func TestOrganizer_PublicPageShowsTheOrganizerButNoContactByDefault(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	page := f.app.Client(t).Inertia("").Get("/r/" + f.raffle.Slug).AssertOK()
	page.AssertDontSee("dono@example.com")

	page.AssertInertia("Raffles/Public").
		PropEquals("organizer.name", "Ana Silva").
		PropEquals("organizer.raffles_total", float64(1)).
		PropEquals("organizer.raffles_finished", float64(0)).
		HasProp("organizer.member_since").
		HasProp("raffle.created_at").
		PropEquals("organizer.has_email", false).
		PropEquals("organizer.whatsapp", nil).
		PropEquals("organizer.telegram", nil)
}

func TestOrganizer_ContactTogglesShowExactlyWhatWasChosen(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	f.c.Get("/settings/contact").AssertOK().AssertInertia("Settings/Contact").
		PropEquals("contact.show_email", false).PropEquals("contact.show_whatsapp", false)

	f.c.WithHeader("Referer", "/settings/contact").PutJSON("/settings/contact", contactForm(map[string]any{
		"whatsapp": "(11) 98765-4321", "show_whatsapp": true, "telegram": "@Rifa_da_Ana", "show_telegram": false,
	})).AssertRedirect("/settings/contact")

	pub := func() *vukemtest.InertiaPage {
		return f.app.Client(t).Inertia("").Get("/r/" + f.raffle.Slug).AssertOK().AssertInertia("Raffles/Public")
	}

	pub().PropEquals("organizer.whatsapp.url", "https://wa.me/5511987654321").
		PropEquals("organizer.whatsapp.display", "+55 11 98765-4321").
		PropEquals("organizer.telegram", nil).
		PropEquals("organizer.has_email", false)

	// Saved but hidden: the Telegram handle never reaches the public page.
	f.app.Client(t).Get("/r/" + f.raffle.Slug).AssertDontSee("rifa_da_ana")

	f.c.WithHeader("Referer", "/settings/contact").PutJSON("/settings/contact", contactForm(map[string]any{
		"show_email": true, "telegram": "@Rifa_da_Ana", "show_telegram": true,
	})).AssertRedirect("/settings/contact")

	page := pub().PropEquals("organizer.whatsapp", nil).
		PropEquals("organizer.telegram.url", "https://t.me/rifa_da_ana").
		PropEquals("organizer.telegram.display", "@rifa_da_ana").
		PropEquals("organizer.has_email", true)
	_ = page

	// Security review fix: the address is in no prop at all, not even split;
	// a visitor's click fetches it from a rate-limited endpoint.
	f.app.Client(t).Get("/r/" + f.raffle.Slug).AssertDontSee("dono@example.com").AssertDontSee(`"domain"`)

	visitor := f.app.Client(t)
	visitor.PostJSON("/r/"+f.raffle.Slug+"/organizer-email", nil).AssertOK().
		AssertJSON(map[string]string{"email": "dono@example.com"}).AssertHeader("Cache-Control", "no-store")

	// Opted out (or no such raffle): 404, the address never leaves.
	f.c.WithHeader("Referer", "/settings/contact").PutJSON("/settings/contact", contactForm(map[string]any{})).
		AssertRedirect("/settings/contact")
	f.app.Client(t).PostJSON("/r/"+f.raffle.Slug+"/organizer-email", nil).AssertNotFound()
	f.app.Client(t).PostJSON("/r/nope-nope/organizer-email", nil).AssertNotFound()
	pub().PropEquals("organizer.has_email", false)

	// Per client IP (every test client shares one), counted before the lookup.
	limited := false
	for range 12 {
		if visitor.PostJSON("/r/"+f.raffle.Slug+"/organizer-email", nil).Status() == http.StatusTooManyRequests {
			limited = true
			break
		}
	}

	if !limited {
		t.Fatal("the reveal endpoint must be rate limited")
	}
}

func TestOrganizer_InvalidContactsAreRejected(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)

	for _, bad := range []map[string]any{
		{"whatsapp": "123", "show_whatsapp": true},
		{"whatsapp": "", "show_whatsapp": true},
		{"telegram": "a b", "show_telegram": true},
		{"telegram": "@ab", "show_telegram": true},
		{"telegram": "https://evil.example", "show_telegram": true},
	} {
		f.c.WithHeader("Referer", "/settings/contact").PutJSON("/settings/contact", contactForm(bad)).
			AssertRedirect("/settings/contact")
		f.c.Get("/settings/contact").AssertInertia("Settings/Contact").HasProp("errors")
	}

	f.app.Client(t).Inertia("").Get("/r/"+f.raffle.Slug).AssertInertia("Raffles/Public").
		PropEquals("organizer.whatsapp", nil).PropEquals("organizer.telegram", nil)
}

func TestOrganizer_OtherOwnersNeverLeak(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t)
	f.c.WithHeader("Referer", "/settings/contact").PutJSON("/settings/contact", contactForm(map[string]any{
		"whatsapp": "(11) 98765-4321", "show_whatsapp": true,
	})).AssertRedirect("/settings/contact")

	other := owner(t, f.app, "outra-dona@example.com")
	createRaffle(t, f.app, other, validRaffle())

	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var otherSlug string
	if err := db.QueryRowContext(t.Context(), "SELECT r.slug FROM raffles r JOIN users u ON u.id = r.user_id WHERE u.email = ?",
		"outra-dona@example.com").Scan(&otherSlug); err != nil {
		t.Fatal(err)
	}

	page := f.app.Client(t).Get("/r/" + otherSlug).AssertOK()
	page.AssertDontSee("5511987654321").AssertDontSee("dono@example.com")
	page.AssertInertia("Raffles/Public").PropEquals("organizer.whatsapp", nil).PropEquals("organizer.raffles_total", float64(1))

	// Guests cannot read or change anyone's contact settings.
	f.app.Client(t).Get("/settings/contact").AssertRedirect("/login")
}
