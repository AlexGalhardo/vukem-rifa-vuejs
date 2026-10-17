// Package routes declares the HTTP routes.
package routes

import (
	"net/http"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/i18n"
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
	r.Use(app.Inertia.Middleware, shareProps(app))

	r.Get("/", controllers.Home).Name("home")

	return nil
}

// shareProps adds the props every page receives (usePage().props).
func shareProps(app *vukem.App) func(http.Handler) http.Handler {
	var locales []string
	if app.I18n != nil {
		locales = app.I18n.Locales()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			inertia.Share(ctx, "appName", app.Config.Name)
			inertia.Share(ctx, "locale", i18n.Locale(ctx))
			inertia.Share(ctx, "locales", locales)
			next.ServeHTTP(w, r)
		})
	}
}
