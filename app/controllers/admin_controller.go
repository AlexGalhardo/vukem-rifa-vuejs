package controllers

import (
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
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
	now  func() time.Time
}

// NewAdminController returns an AdminController; user resolves the
// signed-in admin.
func NewAdminController(db *database.DB, user func(c *web.Ctx) (models.User, error), now func() time.Time) *AdminController {
	return &AdminController{db: db, user: user, now: now}
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

// Refunds lists late payments the platform owes back (refund_due).
func (ac *AdminController) Refunds(c *web.Ctx) error {
	pending, done, err := ledger.RefundsDue(c.Context(), ac.db)
	if err != nil {
		return err
	}

	return inertia.Render(c, "Admin/Refunds", inertia.Props{"pending": pending, "done": done})
}

// MarkRefunded records that a refund_due payment was returned outside the
// app. AbacatePay's API has no refund endpoint, so the admin refunds the
// PIX in its dashboard and records the reference here.
func (ac *AdminController) MarkRefunded(c *web.Ctx) error {
	admin, err := ac.user(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return errs.ErrNotFound
	}

	in, err := web.Parse(c, requests.RefundSchema)
	if err != nil {
		return err
	}

	if err := ledger.MarkRefunded(c.Context(), ac.db, id, in.Reference, admin.ID, ac.now()); err != nil {
		return err
	}

	web.AuditEvent(c.Request(), "refund.recorded", "ledger_entry_id", id, "admin_id", admin.ID)
	inertia.Flash(c, "success", i18n.T(c.Context(), "admin.refunds.recorded"))

	return c.Redirect("/admin/refunds")
}
