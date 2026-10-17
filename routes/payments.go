package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Payments registers the buyer checkout (/pay/{token}). Without
// ABACATEPAY_API_KEY the pages work and confirming answers 503; with
// ABACATEPAY_FAKE=true (refused in production) the fake gateway and its
// /dev helpers are used instead.
func Payments(r *router.Router, app *vukem.App) error {
	db, err := app.DB()
	if err != nil {
		return err
	}

	gw, err := payments.ConfigFrom(app.Source.Get, app.Config.IsProduction()).Gateway()
	if err != nil {
		return err
	}

	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	pay := payments.New(db, app.Now, gw, app.Queue, app.Log)
	pc := controllers.NewPayController(pay)
	buyer := []router.Middleware{privatePage}

	r.Get("/pay/{token}", pc.Show).Name("pay.show").Use(append(buyer,
		ratelimit.Middleware(limiters("pay-show", 60, time.Minute), ratelimit.ByIP))...)
	r.Post("/pay/{token}/confirm", pc.Confirm).Name("pay.confirm").Use(append(buyer,
		ratelimit.Middleware(limiters("pay-confirm", 10, time.Minute), byTokenAndIP))...)
	r.Get("/pay/{token}/status", pc.Status).Name("pay.status").Use(append(buyer,
		ratelimit.Middleware(limiters("pay-status", 60, time.Minute), byTokenAndIP))...)

	if fake, ok := gw.(*abacatepay.Fake); ok {
		app.Log.Warn("payments: AbacatePay fake gateway enabled (ABACATEPAY_FAKE=true); /dev/* helpers are on")

		return devRoutes(r, app, pay, fake)
	}

	return nil
}

// privatePage keeps payment pages out of caches and Referer headers: the
// URL is the buyer's credential.
func privatePage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// byTokenAndIP limits one client on one link; the token is hashed so the
// limiter never holds it in clear.
func byTokenAndIP(r *http.Request) string {
	return reservations.HashToken(r.PathValue("token")) + "|" + ratelimit.ByIP(r)
}

// devRoutes are the fake-gateway helpers for development and e2e: pay a
// link's fake PIX, and reserve a number for a verified owner.
//
// The reserve helper stands in for POST /raffles/{id}/numbers/{n}/link
// (raffles stream) until that route lands; e2e uses it.
func devRoutes(r *router.Router, app *vukem.App, pay *payments.Service, fake *abacatepay.Fake) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	dev := controllers.NewDevPay(pay, fake)
	rs := raffles.New(pay.DB, app.Now, app.Queue)
	res := reservations.New(pay.DB, app.Now, app.Queue)

	r.Post("/dev/pay/{token}/simulate", dev.Simulate).Name("dev.pay.simulate")
	r.Post("/dev/raffles/{raffle}/numbers/{number}/reserve", func(c *web.Ctx) error {
		u, err := auth.User[models.User](c, "web")
		if err != nil {
			return err
		}

		id, err := c.ParamInt("raffle")
		if err != nil {
			return err
		}

		n, err := strconv.Atoi(c.Param("number"))
		if err != nil {
			return err
		}

		raffle, err := rs.ForOwner(c.Context(), u.ID, id)
		if err != nil {
			return err
		}

		in, err := web.Parse(c, requests.BuyerSchema)
		if err != nil {
			return err
		}

		link, err := res.Reserve(c.Context(), raffle, n, in)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusCreated, map[string]string{"token": link.Token, "url": "/pay/" + link.Token})
	}).Name("dev.reserve").Use(auth.Required(acc.Web), auth.Verified(acc.Web))

	return nil
}
