// Package routes declares the HTTP routes.
package routes

import (
	"net/http"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/middleware"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/resources"
)

// Web registers the browser routes. vukem route:list shows them.
func Web(r *router.Router, app *vukem.App) error {
	if err := vukem.DefaultMiddleware(app, r); err != nil {
		return err
	}

	r.UseServer(middleware.StaticFS(resources.Public(), middleware.StaticConfig{Prefix: "/"}))
	// Catalog prefixes the pages use; mail.* and validation.* stay server-side.
	r.Use(app.Inertia.Middleware, inertia.I18n(clientMessagePrefixes...), shareProps(app))

	r.Get("/", controllers.Home).Name("home")

	return nil
}

// clientMessagePrefixes are the translation keys shared with the pages.
var clientMessagePrefixes = []string{"app.", "home.", "theme.", "locale.", "nav.", "auth.", "dashboard.", "raffle.", "link.", "pay.", "draw.", "ledger.", "admin.", "apikeys.", "export."}

// shareProps adds the props every page receives (usePage().props).
func shareProps(app *vukem.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inertia.Share(r.Context(), "appName", app.Config.Name)
			next.ServeHTTP(w, r)
		})
	}
}
