//go:build integration

package tests

import (
	"fmt"
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
