//go:build integration

package tests

import (
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
