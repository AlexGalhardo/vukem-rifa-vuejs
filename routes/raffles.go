package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
)

// Raffles registers the owner dashboard (verified owners only) and the
// public raffle page.
func Raffles(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	rc := controllers.NewRaffleController(raffles.New(acc.DB, app.Now, app.Queue), draws.New(acc.DB, app.Now))
	owner := []router.Middleware{auth.Required(acc.Web), auth.Verified(acc.Web)}

	r.Get("/dashboard", rc.Index).Name("dashboard").Use(owner...)
	r.Get("/raffles/create", rc.Create).Name("raffles.create").Use(owner...)
	r.Post("/raffles", rc.Store).Name("raffles.store").Use(owner...)
	r.Get("/raffles/{raffle}", rc.Show).Name("raffles.show").Use(owner...)
	r.Get("/raffles/{raffle}/edit", rc.Edit).Name("raffles.edit").Use(owner...)
	r.Put("/raffles/{raffle}", rc.Update).Name("raffles.update").Use(owner...)
	r.Delete("/raffles/{raffle}", rc.Destroy).Name("raffles.destroy").Use(owner...)
	r.Post("/raffles/{raffle}/archive", rc.Archive).Name("raffles.archive").Use(owner...)

	r.Get("/r/{slug}", rc.Public).Name("raffles.public")

	return nil
}
