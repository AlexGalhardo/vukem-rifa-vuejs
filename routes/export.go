package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/export"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
)

// Export registers the owner's audit export downloads.
func Export(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	ec := controllers.NewExportController(raffles.New(acc.DB, app.Now), export.New(acc.DB, app.Now))
	owner := []router.Middleware{auth.Required(acc.Web), auth.Verified(acc.Web)}

	r.Get("/raffles/{raffle}/export.json", ec.JSON).Name("raffles.export.json").Use(owner...)
	r.Get("/raffles/{raffle}/export.csv", ec.CSV).Name("raffles.export.csv").Use(owner...)

	return nil
}
