package tests

import (
	"os"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/database/seeders"
)

// The e2e admin exists only when CI sets both variables (t.Setenv: not parallel).
func TestE2EAdminSeeder(t *testing.T) {
	app := vukemtest.New(t, bootstrap.New)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if err := seeders.E2EAdmin().Seed(t.Context(), db); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseMissing(t, app, "users", map[string]any{"email": "admin@e2e.test"})

	t.Setenv("E2E_ADMIN_EMAIL", "admin@e2e.test")
	t.Setenv("E2E_ADMIN_PASSWORD", "short")

	if err := seeders.E2EAdmin().Seed(t.Context(), db); err == nil {
		t.Fatal("short password accepted")
	}

	t.Setenv("E2E_ADMIN_PASSWORD", "e2e-admin-password-ci-only")

	for range 2 { // idempotent
		if err := seeders.E2EAdmin().Seed(t.Context(), db); err != nil {
			t.Fatal(err)
		}
	}

	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "admin@e2e.test", "is_admin": true})

	c := app.Client(t).Inertia("")
	c.PostJSON("/login", map[string]any{"email": "admin@e2e.test", "password": os.Getenv("E2E_ADMIN_PASSWORD")}).AssertRedirect("/dashboard")
	c.Get("/admin/payouts").AssertOK()
}
