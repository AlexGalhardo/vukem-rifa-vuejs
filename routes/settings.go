package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
)

// Settings registers the owner's settings pages (API keys).
func Settings(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	kc := controllers.NewAPIKeyController(acc.Tokens)
	owner := []router.Middleware{auth.Required(acc.Web), auth.Verified(acc.Web)}

	r.Get("/settings/api-keys", kc.Index).Name("apikeys.index").Use(owner...)
	r.Post("/settings/api-keys", kc.Store).Name("apikeys.store").Use(owner...)
	r.Delete("/settings/api-keys/{id}", kc.Destroy).Name("apikeys.destroy").Use(owner...)

	return nil
}
