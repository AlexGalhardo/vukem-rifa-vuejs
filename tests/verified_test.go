package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// The owner dashboard needs a verified email; browsers are sent to the
// verification notice instead of a bare 403.
func TestDashboard_RequiresVerifiedEmail(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := app.Client(t).Inertia("")

	register(t, c, "dora@example.com")

	c.Get("/dashboard").AssertRedirect("/email/verify")

	c.Get(link(t, app, "dora@example.com")).AssertRedirect("/dashboard")
	c.Get("/dashboard").AssertOK().AssertInertia("Dashboard")
}
