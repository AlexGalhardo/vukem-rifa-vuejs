//go:build integration

package tests

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/config"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/tests/pgtest"
)

func newPostgresApp(t *testing.T) *vukemtest.App {
	t.Helper()

	return vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{
		"DATABASE_URL":     pgtest.NewDatabase(t),
		"SESSION_DRIVER":   "database",
		"QUEUE_CONNECTION": "database",
		"CACHE_STORE":      "database",
		"RATE_LIMIT_STORE": "cache",
	}})
}

func TestPostgres_BootMigrateAndServe(t *testing.T) {
	t.Parallel()

	app := newPostgresApp(t)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if got := db.Dialect().Name(); got != "postgres" {
		t.Fatalf("dialect = %q, want postgres", got)
	}

	app.Client(t).Get("/up?checks=1").AssertOK()
	app.Client(t).Get("/").AssertOK().AssertInertia("Home")
}

func TestPostgres_RaffleNumbersAndConstraints(t *testing.T) {
	t.Parallel()

	app := newPostgresApp(t)
	c := owner(t, app, "pg@example.com")

	fields := validRaffle()
	fields["number_count"] = 1000
	id := createRaffle(t, app, c, fields)

	c.Get(fmt.Sprintf("/raffles/%d", id)).AssertInertia("Raffles/Show").HasProp("numbers", 1000).
		PropEquals("numbers.999.number", float64(1000))

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	// The database refuses what the z schema would: defense in depth.
	bad := []string{
		"UPDATE raffles SET number_count = 1001 WHERE id = $1",
		"UPDATE raffles SET price_reais = 0 WHERE id = $1",
		"UPDATE raffles SET duration_days = 61 WHERE id = $1",
		"UPDATE raffles SET status = 'bogus' WHERE id = $1",
		"UPDATE raffle_numbers SET status = 'sold' WHERE raffle_id = $1 AND number = 1",
		"INSERT INTO raffle_numbers (raffle_id, number, status, created_at, updated_at) VALUES ($1, 5, 'available', now(), now())",
	}

	for _, stmt := range bad {
		if _, err := db.ExecContext(t.Context(), stmt, id); err == nil {
			t.Errorf("%s: accepted, want a constraint error", stmt)
		}
	}
}

// RATE_LIMIT_STORE=cache counts attempts in the database cache, so every
// Railway instance shares the login budget.
func TestPostgres_LoginThrottleSharedInDatabase(t *testing.T) {
	t.Parallel()

	app := newPostgresApp(t)
	register(t, app.Client(t).Inertia(""), "pgbob@example.com")

	// Fixed windows (RATE_LIMIT_STORE=cache): a slow run can straddle a
	// window boundary, so allow up to two windows' worth before the 429.
	attacker := app.Client(t).Inertia("")
	throttled := false

	for range 11 {
		res := attacker.PostJSON("/login", map[string]any{"email": "pgbob@example.com", "password": "guess"})
		if res.Status() == 429 {
			throttled = true
			break
		}
	}

	if !throttled {
		t.Fatal("eleven failed logins were never throttled (429)")
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM cache").Scan(&n); err != nil || n == 0 {
		t.Fatalf("cache rows = %d, %v; want rate-limit counters", n, err)
	}
}

// A password reset signs the account out of every other browser (database
// sessions hold the user id since vukem v0.24.0).
func TestPostgres_PasswordResetEndsOtherSessions(t *testing.T) {
	t.Parallel()

	// Sync queue: the queued reset mail is delivered during the request.
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{
		"DATABASE_URL":   pgtest.NewDatabase(t),
		"SESSION_DRIVER": "database",
		"CACHE_STORE":    "database",
	}})
	victim := owner(t, app, "reset-pg@example.com")
	victim.Get("/dashboard").AssertOK()

	c := app.Client(t).Inertia("")
	c.PostJSON("/forgot-password", map[string]any{"email": "reset-pg@example.com"}).AssertRedirect("/forgot-password")

	reset := link(t, app, "reset-pg@example.com")
	u, err := url.Parse(reset)
	if err != nil {
		t.Fatal(err)
	}

	c.PostJSON("/reset-password", map[string]any{
		"token": strings.TrimPrefix(u.Path, "/reset-password/"), "email": "reset-pg@example.com",
		"password": "a brand new password", "password_confirmation": "a brand new password",
	}).AssertRedirect("/login")

	victim.Get("/dashboard").AssertRedirect("/login")
}

// A07: a session cookie copied before logout is dead after it (database
// sessions are destroyed server-side, not just cleared in the browser).
func TestPostgres_OWASP_A07_LogoutKillsCopiedSessionCookie(t *testing.T) {
	t.Parallel()

	app := newPostgresApp(t)
	c := owner(t, app, "logout-pg@example.com")
	c.Get("/dashboard").AssertOK()

	stolen := c.CookieValue("vukem_session")
	if stolen == "" {
		t.Fatal("no session cookie")
	}

	c.PostJSON("/logout", nil).AssertRedirect("/")

	app.Client(t).Inertia("").WithCookie("vukem_session", stolen).Get("/dashboard").AssertRedirect("/login")
}

// A07: the reset mail is queued, not sent during the request, so a known
// email does not answer measurably slower than an unknown one (no account
// enumeration by timing); both get the same redirect.
func TestPostgres_OWASP_A07_ResetMailIsQueuedNotSentInline(t *testing.T) {
	t.Parallel()

	app := newPostgresApp(t)
	register(t, app.Client(t).Inertia(""), "enum-pg@example.com")

	// The database queue creates its table on first use.
	if _, err := app.Queue(); err != nil {
		t.Fatal(err)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	jobs := func() int {
		var n int
		if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM jobs").Scan(&n); err != nil {
			t.Fatal(err)
		}

		return n
	}

	before, sent := jobs(), len(app.Mails().SentTo("enum-pg@example.com"))

	c := app.Client(t).Inertia("")
	c.PostJSON("/forgot-password", map[string]any{"email": "nobody-pg@example.com"}).AssertRedirect("/forgot-password")
	c.PostJSON("/forgot-password", map[string]any{"email": "enum-pg@example.com"}).AssertRedirect("/forgot-password")

	if got := jobs(); got != before+1 {
		t.Fatalf("queued jobs = %d, want %d (one reset mail, none for the unknown email)", got, before+1)
	}

	if got := len(app.Mails().SentTo("enum-pg@example.com")); got != sent {
		t.Fatalf("reset mail sent inline (%d → %d)", sent, got)
	}
}

// A07: sign-up never reveals an existing account: same status, redirect and
// cookies for a new and an existing email, one queued mail each (the
// verification or the "account exists" notice), nothing sent inline.
func TestPostgres_OWASP_A07_SignupIsEnumerationSafe(t *testing.T) {
	t.Parallel()

	app := newPostgresApp(t)
	register(t, app.Client(t).Inertia(""), "taken-pg@example.com")

	if _, err := app.Queue(); err != nil {
		t.Fatal(err)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	jobs := func() int {
		var n int
		if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM jobs").Scan(&n); err != nil {
			t.Fatal(err)
		}

		return n
	}

	sentBefore := len(app.Mails().Sent())

	signup := func(email string) (int, string, []string) {
		res := app.Client(t).Inertia("").PostJSON("/register", map[string]any{
			"name": "Eve", "email": email, "password": "a long enough password", "password_confirmation": "a long enough password",
		})

		var cookies []string
		for _, c := range res.Recorder().Header().Values("Set-Cookie") {
			cookies = append(cookies, strings.SplitN(c, "=", 2)[0])
		}

		return res.Status(), res.Header("Location"), cookies
	}

	before := jobs()
	newStatus, newLoc, newCookies := signup("fresh-pg@example.com")
	takenStatus, takenLoc, takenCookies := signup("taken-pg@example.com")

	if newStatus != takenStatus || newLoc != takenLoc || fmt.Sprint(newCookies) != fmt.Sprint(takenCookies) {
		t.Fatalf("responses differ: new %d %s %v, existing %d %s %v", newStatus, newLoc, newCookies, takenStatus, takenLoc, takenCookies)
	}

	if got := jobs(); got != before+2 {
		t.Fatalf("queued jobs = %d, want %d (one mail per sign-up)", got, before+2)
	}

	if got := len(app.Mails().Sent()); got != sentBefore {
		t.Fatalf("mails sent inline during sign-up: %d → %d", sentBefore, got)
	}
}

// A04: APP_KEY rotation. Data sealed with the old key (payment-link tokens
// shown to the owner) still opens after a rotation that keeps the old key
// in APP_PREVIOUS_KEYS, and fails closed once the old key is dropped.
func TestPostgres_OWASP_A04_AppKeyRotationKeepsSealedData(t *testing.T) {
	t.Parallel()

	dbURL := pgtest.NewDatabase(t)
	oldKey, newKey := config.GenerateKey(), config.GenerateKey()
	env := func(key, previous string) vukemtest.Options {
		return vukemtest.Options{Env: map[string]string{
			"DATABASE_URL": dbURL, "APP_KEY": key, "APP_PREVIOUS_KEYS": previous,
			"SESSION_DRIVER": "database", "QUEUE_CONNECTION": "database", "CACHE_STORE": "database",
		}}
	}

	before := newReservationFixture(t, env(oldKey, ""))
	link, err := before.svc.Reserve(t.Context(), before.raffle, 5, buyer)
	if err != nil {
		t.Fatal(err)
	}

	links := func(o vukemtest.Options) ([]raffles.ActiveLink, error) {
		app := vukemtest.New(t, bootstrap.New, o)

		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		return raffles.New(db, app.Now, nil).ActiveLinks(t.Context(), before.raffle.ID)
	}

	rotated, err := links(env(newKey, oldKey))
	if err != nil || len(rotated) != 1 || rotated[0].Token != link.Token {
		t.Fatalf("after rotation: %v, %v; want the sealed token back", rotated, err)
	}

	if _, err := links(env(newKey, "")); err == nil {
		t.Fatal("old ciphertext opened without the previous key")
	}
}
