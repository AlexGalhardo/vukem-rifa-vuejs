package tests

import (
	"fmt"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Security review fix: late payments (refund_due) are listed for admins,
// who mark them refunded after returning the PIX in the AbacatePay
// dashboard (the API has no refund endpoint); owners' balances never move.
func TestAdminRefunds_ListAndMarkRefunded(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 11)
	f.clock.Advance(models.ReservationTTL + reservations.PaymentGrace)
	deliverSigned(t, f.app, paidEvent("log_late_refund", *res.ChargeID, res.ID, 1000)).AssertOK()

	balanceBefore := f.balance(t)

	// Not for owners or guests.
	login(t, f.app, "dono@example.com").Get("/admin/refunds").AssertNotFound()
	f.app.Client(t).Get("/admin/refunds").AssertNotFound()

	admin := makeAdmin(t, f.app, "admin-refunds@example.com")
	page := admin.Get("/admin/refunds").AssertOK().AssertInertia("Admin/Refunds").HasProp("pending", 1).
		PropEquals("pending.0.amount_cents", float64(1000)).PropEquals("pending.0.number", float64(11)).
		PropEquals("pending.0.buyer_email", "maria@example.com")

	if n, _, err := ledger.PendingRefunds(t.Context(), f.svc.DB, f.clock.Now()); err != nil || n != 1 {
		t.Fatalf("pending refunds = %d, %v", n, err)
	}

	raw, _ := page.Prop("pending.0.id")
	path := fmt.Sprintf("/admin/refunds/%v", raw)

	api := loginJSON(t, f.app, "admin-refunds@example.com")
	api.PostJSON(path, map[string]any{"reference": ""}).AssertUnprocessable()
	admin.WithHeader("Referer", "/admin/refunds").PostJSON(path, map[string]any{"reference": "E2E-REFUND-1"}).
		AssertRedirect("/admin/refunds")
	admin.Get("/admin/refunds").AssertInertia("Admin/Refunds").HasProp("pending", 0).HasProp("done", 1)

	// Twice is refused; the owner balance never changed.
	admin.WithHeader("Referer", "/admin/refunds").PostJSON(path, map[string]any{"reference": "again"}).AssertStatus(409)
	vukemtest.AssertDatabaseCount(t, f.app, "refunds", 1)

	if n, _, _ := ledger.PendingRefunds(t.Context(), f.svc.DB, f.clock.Now()); n != 0 {
		t.Fatalf("pending refunds after marking = %d", n)
	}

	if got := f.balance(t); got != balanceBefore {
		t.Fatalf("owner balance moved from %d to %d", balanceBefore, got)
	}
}
