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
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// LinksPerMinute caps payment-link generation per owner.
const LinksPerMinute = 30

// Numbers registers the owner's payment-link actions (verified owners,
// CSRF from the default middleware, link generation rate limited per
// owner).
func Numbers(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	nc := controllers.NewNumberController(
		raffles.New(acc.DB, app.Now, app.Queue),
		reservations.New(acc.DB, app.Now, app.Queue),
		app.Config.URL,
	)
	owner := []router.Middleware{auth.Required(acc.Web), auth.Verified(acc.Web)}
	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	links := ratelimit.Middleware(limiters("payment-links", LinksPerMinute, time.Minute), byOwner)

	r.Post("/raffles/{raffle}/numbers/{number}/link", nc.Link).Name("numbers.link").Use(append(owner, links)...)
	r.Post("/raffles/{raffle}/numbers/{number}/cancel", nc.Cancel).Name("numbers.cancel").Use(owner...)

	return nil
}

// byOwner keys the limit by the signed-in user (it runs after
// auth.Required), so owners behind one NAT do not share a bucket.
func byOwner(r *http.Request) string {
	c := web.NewCtx(nil, r, web.ServicesFrom(r.Context()))

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return ratelimit.ByIP(r)
	}

	return "user:" + strconv.FormatInt(u.ID, 10)
}
