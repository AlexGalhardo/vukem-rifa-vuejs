package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// owner registers and verifies an account and returns its Inertia client.
func owner(t *testing.T, app *vukemtest.App, email string) *vukemtest.Client {
	t.Helper()

	c := app.Client(t).Inertia(app.Inertia.Version())
	register(t, c, email)
	c.Get(link(t, app, email)).AssertRedirect("/dashboard")

	return c
}

func validRaffle() map[string]any {
	return map[string]any{
		"title": "Rifa da moto", "description": "Uma moto 0 km.",
		"number_count": 100, "price_reais": 10, "duration_days": 30,
	}
}

// createRaffle creates a raffle and returns its id.
func createRaffle(t *testing.T, app *vukemtest.App, c *vukemtest.Client, fields map[string]any) int64 {
	t.Helper()

	res := c.PostJSON("/raffles", fields)
	loc := res.Header("Location")

	var id int64
	if _, err := fmt.Sscanf(loc, "/raffles/%d", &id); err != nil {
		t.Fatalf("create raffle: status %d, Location %q", res.Status(), loc)
	}

	return id
}

func TestRaffle_CreateWithNumbers(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "ana@example.com")

	id := createRaffle(t, app, c, validRaffle())

	page := c.Get(fmt.Sprintf("/raffles/%d", id)).AssertOK().AssertInertia("Raffles/Show").
		PropEquals("raffle.title", "Rifa da moto").
		PropEquals("raffle.price_reais", float64(10)).
		PropEquals("raffle.locked", false).
		HasProp("numbers", 100)

	page.PropEquals("numbers.0.number", float64(1)).PropEquals("numbers.99.number", float64(100)).
		PropEquals("numbers.0.status", "available")

	c.Get("/dashboard").AssertInertia("Dashboard").HasProp("raffles", 1).
		PropEquals("raffles.0.available", float64(100))
}

func TestRaffle_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		field string
		value any
	}{
		{"number_count", 9},
		{"number_count", 1001},
		{"price_reais", 0},
		{"price_reais", 10.5},
		{"duration_days", 6},
		{"duration_days", 61},
		{"title", "ab"},
		{"title", strings.Repeat("x", 121)},
		{"description", strings.Repeat("x", 5001)},
	}

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "val@example.com")

	for _, tt := range tests {
		name := fmt.Sprintf("%s=%v", tt.field, tt.value)
		t.Run(name[:min(40, len(name))], func(t *testing.T) {
			fields := validRaffle()
			fields[tt.field] = tt.value

			c.WithHeader("Referer", "/raffles/create").PostJSON("/raffles", fields).AssertRedirect("/raffles/create")
			c.Get("/raffles/create").AssertInertia("Raffles/Create").HasProp("errors." + tt.field)
		})
	}

	vukemtest.AssertDatabaseCount(t, app, "raffles", 0)
}

func TestRaffle_OtherOwnersGet404(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	ana := owner(t, app, "ana@example.com")
	bob := owner(t, app, "bob@example.com")

	id := createRaffle(t, app, ana, validRaffle())
	path := fmt.Sprintf("/raffles/%d", id)

	bob.Get(path).AssertNotFound()
	bob.Get(path + "/edit").AssertNotFound()
	bob.PutJSON(path, validRaffle()).AssertNotFound()
	bob.Delete(path).AssertNotFound()
	bob.PostJSON(path+"/archive", nil).AssertNotFound()

	bob.Get("/dashboard").AssertInertia("Dashboard").HasProp("raffles", 0)
	vukemtest.AssertDatabaseCount(t, app, "raffles", 1)
}

func TestRaffle_GuestsAndUnverifiedCannotManage(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)

	app.Client(t).Get("/raffles/create").AssertRedirect("/login")

	c := app.Client(t).Inertia(app.Inertia.Version())
	register(t, c, "new@example.com")
	c.Get("/raffles/create").AssertRedirect("/email/verify")
	c.PostJSON("/raffles", validRaffle()).AssertRedirect("/email/verify")
	vukemtest.AssertDatabaseCount(t, app, "raffles", 0)
}

func TestRaffle_UpdateAndResize(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "ana@example.com")
	id := createRaffle(t, app, c, validRaffle())
	path := fmt.Sprintf("/raffles/%d", id)

	fields := validRaffle()
	fields["title"], fields["number_count"], fields["price_reais"] = "Rifa nova", 150, 20
	c.PutJSON(path, fields).AssertRedirect(path)
	c.Get(path).AssertInertia("Raffles/Show").PropEquals("raffle.title", "Rifa nova").HasProp("numbers", 150).
		PropEquals("numbers.149.number", float64(150))

	fields["number_count"] = 10
	c.PutJSON(path, fields).AssertRedirect(path)
	c.Get(path).AssertInertia("Raffles/Show").HasProp("numbers", 10)
}

func TestRaffle_LockedAfterFirstPaidNumber(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "ana@example.com")
	id := createRaffle(t, app, c, validRaffle())
	path := fmt.Sprintf("/raffles/%d", id)

	markPaid(t, app, id, 7)

	locked := validRaffle()
	locked["price_reais"] = 99
	c.WithHeader("Referer", path+"/edit").PutJSON(path, locked).AssertRedirect(path + "/edit")
	c.Get(path + "/edit").AssertInertia("Raffles/Edit").HasProp("errors.price_reais")

	// Title and description stay editable.
	ok := validRaffle()
	ok["title"], ok["description"] = "Moto 2026", "Nova descrição"
	c.PutJSON(path, ok).AssertRedirect(path)
	c.Get(path).AssertInertia("Raffles/Show").PropEquals("raffle.title", "Moto 2026").PropEquals("raffle.locked", true)

	// Paid numbers forbid delete; archive instead.
	c.Delete(path).AssertStatus(409)
	c.PostJSON(path+"/archive", nil).AssertRedirect(path)
	c.Get(path).AssertInertia("Raffles/Show").PropEquals("raffle.status", "archived")
	vukemtest.AssertDatabaseCount(t, app, "raffles", 1)
}

func TestRaffle_DeleteWithoutPaidNumbers(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "ana@example.com")
	id := createRaffle(t, app, c, validRaffle())

	c.Delete(fmt.Sprintf("/raffles/%d", id)).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseCount(t, app, "raffles", 0)
	vukemtest.AssertDatabaseCount(t, app, "raffle_numbers", 0)
}

func TestRaffle_PublicPageHasNoPersonalData(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "ana@example.com")
	id := createRaffle(t, app, c, validRaffle())
	markPaid(t, app, id, 3)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var slug string
	if err := db.QueryRowContext(t.Context(), "SELECT slug FROM raffles WHERE id = ?", id).Scan(&slug); err != nil {
		t.Fatal(err)
	}

	res := app.Client(t).Get("/r/" + slug).AssertOK()
	res.AssertInertia("Raffles/Public").
		PropEquals("raffle.title", "Rifa da moto").
		HasProp("statuses", 100).
		PropEquals("statuses.2", "paid").
		PropEquals("statuses.3", "available").
		MissingProp("raffle.id").
		MissingProp("raffle.user_id").
		MissingProp("numbers")
	res.AssertDontSee("ana@example.com")

	app.Client(t).Get("/r/doesnotexist").AssertNotFound()
}

// markPaid simulates a confirmed payment until the payments phase lands.
func markPaid(t *testing.T, app *vukemtest.App, raffleID int64, number int) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	now := app.Now().UTC()
	if _, err := db.ExecContext(t.Context(), "UPDATE raffle_numbers SET status = 'paid', paid_at = ? WHERE raffle_id = ? AND number = ?",
		now, raffleID, number); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), "UPDATE raffles SET locked_at = ? WHERE id = ?", now, raffleID); err != nil {
		t.Fatal(err)
	}
}
