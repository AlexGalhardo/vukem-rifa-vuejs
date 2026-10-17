package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// paidSale pays number n of the fixture raffle through the webhook.
func (f reservationFixture) paidSale(t *testing.T, number int) {
	t.Helper()

	_, res := f.confirmed(t, number)
	deliverSigned(t, f.app, paidEvent("log_sale_"+res.TokenHash[:12], *res.ChargeID, res.ID, 1000)).AssertOK()
}

func (f reservationFixture) balance(t *testing.T) int64 {
	t.Helper()

	b, err := ledger.Balance(t.Context(), f.svc.DB, f.raffle.UserID)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

// login returns an Inertia client signed in as the user with email.
func login(t *testing.T, app *vukemtest.App, email string) *vukemtest.Client {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u, err := orm.QueryOf[models.User](db).Where("email", email).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return app.Client(t).Inertia("").ActingAs("web", u)
}

// loginJSON is login for an API-style client (validation errors are 422).
func loginJSON(t *testing.T, app *vukemtest.App, email string) *vukemtest.Client {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u, err := orm.QueryOf[models.User](db).Where("email", email).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return app.Client(t).AcceptJSON().ActingAs("web", u)
}

func makeAdmin(t *testing.T, app *vukemtest.App, email string) *vukemtest.Client {
	t.Helper()

	c := owner(t, app, email)
	if err := app.Call(t.Context(), "make:admin", []string{email}); err != nil {
		t.Fatal(err)
	}

	return c
}

func TestDashboard_ShowsTheOwnerBalance(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	f.paidSale(t, 1)
	f.paidSale(t, 2)

	login(t, f.app, "dono@example.com").Get("/dashboard").AssertOK().AssertInertia("Dashboard").
		PropEquals("balance_cents", 1800)
}

func TestMakeAdmin_UnknownEmailFails(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	if err := app.Call(t.Context(), "make:admin", []string{"nobody@example.com"}); err == nil {
		t.Fatal("want error for unknown email")
	}

	if err := app.Call(t.Context(), "make:admin", nil); err == nil {
		t.Fatal("want usage error")
	}
}

func TestAdminPayouts_HiddenFromNonAdmins(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	ownerClient := login(t, f.app, "dono@example.com")

	for _, c := range []*vukemtest.Client{f.app.Client(t), ownerClient} {
		c.Get("/admin/payouts").AssertNotFound()
		c.PostJSON("/admin/payouts", map[string]any{"user_id": f.raffle.UserID, "amount_cents": 100, "reference": "x"}).
			AssertNotFound()
	}

	vukemtest.AssertDatabaseCount(t, f.app, "payouts", 0)
}

func TestAdminPayouts_RecordsAPayoutWithinTheBalance(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	f.paidSale(t, 3)
	f.paidSale(t, 4)

	admin := makeAdmin(t, f.app, "admin@example.com")

	admin.Get("/admin/payouts").AssertOK().AssertInertia("Admin/Payouts").
		HasProp("owners", 1).
		PropEquals("owners.0.user_id", f.raffle.UserID).
		PropEquals("owners.0.balance_cents", 1800)

	api := loginJSON(t, f.app, "admin@example.com")
	for _, tc := range []map[string]any{
		{"user_id": f.raffle.UserID, "amount_cents": 1801, "reference": "E2E123"},
		{"user_id": f.raffle.UserID, "amount_cents": 0, "reference": "E2E123"},
		{"user_id": f.raffle.UserID, "amount_cents": -5, "reference": "E2E123"},
		{"user_id": f.raffle.UserID, "amount_cents": 100, "reference": ""},
	} {
		api.PostJSON("/admin/payouts", tc).AssertUnprocessable()
	}

	api.PostJSON("/admin/payouts", map[string]any{"user_id": 999999, "amount_cents": 100, "reference": "x"}).AssertNotFound()

	admin.Inertia("").PostJSON("/admin/payouts", map[string]any{
		"user_id": f.raffle.UserID, "amount_cents": 1500, "reference": "E2E-PIX-1",
	}).AssertRedirect("/admin/payouts")

	if got := f.balance(t); got != 300 {
		t.Fatalf("balance = %d, want 300", got)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "payouts", map[string]any{"user_id": f.raffle.UserID, "amount_cents": 1500, "reference": "E2E-PIX-1"})
	vukemtest.AssertDatabaseHas(t, f.app, "ledger_entries", map[string]any{"user_id": f.raffle.UserID, "kind": "payout", "amount_cents": -1500})

	// The rest of the balance, and nothing more.
	api.PostJSON("/admin/payouts", map[string]any{"user_id": f.raffle.UserID, "amount_cents": 301, "reference": "x"}).AssertUnprocessable()
	admin.Inertia("").PostJSON("/admin/payouts", map[string]any{"user_id": f.raffle.UserID, "amount_cents": 300, "reference": "E2E-PIX-2"}).
		AssertRedirect("/admin/payouts")

	if got := f.balance(t); got != 0 {
		t.Fatalf("balance = %d, want 0", got)
	}

	admin.Get("/admin/payouts").AssertOK().AssertInertia("Admin/Payouts").HasProp("owners", 0)
}
