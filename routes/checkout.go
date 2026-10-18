package routes

import (
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Checkout registers the public purchase of a free number (anyone with the
// raffle link; CSRF from the default middleware; rate limited per IP and,
// in the handler, per buyer email) and the buyer's purchase history.
func Checkout(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	pc := controllers.NewPublicCheckoutController(
		raffles.New(acc.DB, app.Now, app.Queue),
		reservations.New(acc.DB, app.Now, app.Queue),
		acc,
		limiters("public-checkout-email", 10, 10*time.Minute),
	)

	perIP := ratelimit.Middleware(limiters("public-checkout-ip", 10, 10*time.Minute), ratelimit.ByIP)

	r.Post("/r/{slug}/numbers/{number}/reserve", pc.Reserve).Name("checkout.reserve").Use(perIP)
	r.Get("/account/purchases", pc.Purchases).Name("account.purchases").Use(auth.Required(acc.Web))

	return nil
}
