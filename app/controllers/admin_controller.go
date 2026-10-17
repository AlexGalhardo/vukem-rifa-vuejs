package controllers

import (
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// AdminController is the payouts page (admins only; the route middleware
// answers 404 to everyone else).
type AdminController struct {
	db   *database.DB
	user func(c *web.Ctx) (models.User, error)
}

// NewAdminController returns an AdminController; user resolves the
// signed-in admin.
func NewAdminController(db *database.DB, user func(c *web.Ctx) (models.User, error)) *AdminController {
	return &AdminController{db: db, user: user}
}

// Payouts lists owners with a balance.
func (ac *AdminController) Payouts(c *web.Ctx) error {
	owners, err := ledger.Balances(c.Context(), ac.db)
	if err != nil {
		return err
	}

	return inertia.Render(c, "Admin/Payouts", inertia.Props{"owners": owners, "max_cents": requests.MaxPayoutCents})
}

// StorePayout records a payout made outside the app.
func (ac *AdminController) StorePayout(c *web.Ctx) error {
	admin, err := ac.user(c)
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.PayoutSchema)
	if err != nil {
		return err
	}

	payout, err := ledger.RecordPayout(c.Context(), ac.db, in, admin.ID)
	if err != nil {
		return err
	}

	web.AuditEvent(c.Request(), "payout.recorded", "payout_id", payout.ID, "owner_id", payout.UserID,
		"amount_cents", payout.AmountCents, "admin_id", admin.ID)
	inertia.Flash(c, "success", i18n.T(c.Context(), "admin.payouts.recorded"))

	return c.Redirect("/admin/payouts")
}
