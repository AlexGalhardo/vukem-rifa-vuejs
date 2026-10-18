package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// vukem v0.27.0: errors on Inertia visits render the Error page (inside the
// app layout) instead of the raw HTML error page in Inertia's modal.
func TestErrorPage_InertiaVisitsGetTheErrorComponent(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{})

	app.Client(t).Inertia("").Get("/r/does-not-exist").AssertStatus(http.StatusNotFound).
		AssertInertia("Error").PropEquals("status", float64(http.StatusNotFound))

	// A plain browser visit still gets the HTML error page.
	app.Client(t).Get("/r/does-not-exist").AssertStatus(http.StatusNotFound).AssertHeaderContains("Content-Type", "text/html")
}
