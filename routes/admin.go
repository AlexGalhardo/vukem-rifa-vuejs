package routes

import (
	"net/http"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Admin registers the payouts page. Guests and non-admins get 404, so the
// page's existence is not revealed.
func Admin(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	user := func(c *web.Ctx) (models.User, error) {
		u, err := acc.Web.User(c)
		if err != nil || !u.IsAdmin || !u.EmailVerified() {
			return models.User{}, errs.ErrNotFound
		}

		return u, nil
	}

	onlyAdmins := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := user(web.NewCtx(w, r, web.ServicesFrom(r.Context()))); err != nil {
				web.Fail(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	ac := controllers.NewAdminController(acc.DB, user)

	r.Get("/admin/payouts", ac.Payouts).Name("admin.payouts").Use(onlyAdmins)
	r.Post("/admin/payouts", ac.StorePayout).Name("admin.payouts.store").Use(onlyAdmins)

	return nil
}
