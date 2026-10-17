package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// New accounts are never admins; payouts (Phase 5) are recorded by admins only.
func TestRegister_IsNotAdmin(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	register(t, app.Client(t).Inertia(app.Inertia.Version()), "eva@example.com")

	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "eva@example.com", "is_admin": false})
}
