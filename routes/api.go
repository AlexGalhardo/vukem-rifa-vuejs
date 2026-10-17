package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/apidocs"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/openapi"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers/apiv1"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/export"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// APIPrefix is where the REST API v1 lives; the OpenAPI document covers
// only it.
const APIPrefix = "/api/v1"

// API registers the REST API v1 (API keys only, no cookies, no CORS), its
// OpenAPI document and the Scalar reference.
func API(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	store, err := app.Cache()
	if err != nil {
		return err
	}

	// workaround(vukem v0.22.0): OPENAPI_PREFIX can only come from the environment; the app pins it in code.
	if app.Config.OpenAPIPrefix == "" {
		app.Config.OpenAPIPrefix = APIPrefix
	}

	r.UseServer(problemJSON)

	h := &apiv1.Handlers{
		DB:           acc.DB,
		Raffles:      raffles.New(acc.DB, app.Now),
		Reservations: reservations.New(acc.DB, app.Now, app.Queue),
		Export:       export.New(acc.DB, app.Now),
		URL:          acc.URL,
	}

	mw := apiv1.Middleware{
		Read:         auth.Ability(acc.API, requests.AbilityRafflesRead),
		Write:        auth.Ability(acc.API, requests.AbilityRafflesWrite),
		LinkThrottle: ratelimit.Middleware(ratelimit.NewMemory(30, time.Minute), byAPIKey),
		Idempotency: api.Idempotency(store.Store(), api.IdempotencyOptions{
			// The response holds a payment link: keep it no longer than the
			// link could matter.
			TTL:   time.Hour,
			Scope: byAPIKey,
		}),
	}

	r.Group(APIPrefix, func(g *router.Group) {
		h.Register(g, mw)
	}).Name("api.v1.").Use(
		// Bounds made-up tokens before they are checked, then each key.
		ratelimit.Middleware(ratelimit.NewMemory(300, time.Minute), ratelimit.ByIP),
		tokenOnly(acc.API),
		ratelimit.Middleware(ratelimit.NewMemory(120, time.Minute), byAPIKey),
	)

	// Public by design (PLAN.md §5): the API surface is documented for owners.
	r.Get("/api/openapi.json", func(c *web.Ctx) error {
		doc, err := app.OpenAPI()
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, doc)
	}).Name("api.openapi").Doc(openapi.Skip{})

	apidocs.Register(r, "/api/docs", apidocs.Config{
		SpecURL:        "/api/openapi.json",
		Title:          app.Config.Name + " API",
		Authentication: map[string]any{"preferredSecurityScheme": "bearer"},
	})

	return nil
}

// tokenOnly authenticates with an API key and nothing else: a signed-in
// browser session is never ambient authority for the API. Unlike
// auth.Required it never redirects to the login page.
func tokenOnly(guard *auth.TokenGuard[models.User]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := web.NewCtx(w, r, web.ServicesFrom(r.Context()))
			if !guard.Check(c) {
				c.HandleError(auth.ErrUnauthenticated.WithHeader("WWW-Authenticate", `Bearer realm="api"`))
				return
			}

			next.ServeHTTP(w, c.Request())
		})
	}
}

// byAPIKey keys rate limits and idempotency records by the API key id
// (never by the secret). Empty before authentication.
func byAPIKey(r *http.Request) string {
	tok, ok := auth.CurrentToken(web.NewCtx(nil, r, nil), apiv1.Guard)
	if !ok {
		return ""
	}

	return "apikey:" + tok.ID
}

// problemJSON makes every /api/ response, including 404 and 405 from the
// router, an application/problem+json document: the error handler picks
// the format from Accept, and API clients often send none.
func problemJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1" {
			r.Header.Set("Accept", "application/json")
		}

		next.ServeHTTP(w, r)
	})
}
