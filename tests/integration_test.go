//go:build integration

package tests

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

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

	attacker := app.Client(t).Inertia("")
	for range 5 {
		attacker.PostJSON("/login", map[string]any{"email": "pgbob@example.com", "password": "guess"})
	}

	attacker.PostJSON("/login", map[string]any{"email": "pgbob@example.com", "password": "guess"}).AssertStatus(429)

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

	app := newPostgresApp(t)
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
