package controllers

import (
	"strings"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// RaffleController is the owner's raffle management (dashboard) and the
// public raffle page.
type RaffleController struct {
	raffles *raffles.Service
	draws   *draws.Service
	appURL  string
}

// NewRaffleController returns a RaffleController; appURL builds the payment
// links shown to the owner.
func NewRaffleController(s *raffles.Service, d *draws.Service, appURL string) *RaffleController {
	return &RaffleController{raffles: s, draws: d, appURL: strings.TrimSuffix(appURL, "/")}
}

// limits are sent to the forms so client validation mirrors the server.
var limits = map[string]int{
	"min_numbers": requests.MinNumbers, "max_numbers": requests.MaxNumbers,
	"min_price": requests.MinPrice, "max_price": requests.MaxPrice,
	"min_duration": requests.MinDuration, "max_duration": requests.MaxDuration,
}

// Index is the dashboard: the owner's raffles with number counts.
func (rc *RaffleController) Index(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	list, err := rc.raffles.List(c.Context(), u.ID)
	if err != nil {
		return err
	}

	balance, err := ledger.Balance(c.Context(), rc.raffles.DB, u.ID)
	if err != nil {
		return err
	}

	return inertia.Render(c, "Dashboard", inertia.Props{"raffles": list, "balance_cents": balance})
}

// Create shows the new raffle form.
func (*RaffleController) Create(c *web.Ctx) error {
	return inertia.Render(c, "Raffles/Create", inertia.Props{"limits": limits})
}

// Store validates and creates a raffle.
func (rc *RaffleController) Store(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.RaffleSchema)
	if err != nil {
		return err
	}

	r, err := rc.raffles.Create(c.Context(), u.ID, in)
	if err != nil {
		return err
	}

	inertia.Flash(c, "success", i18n.T(c.Context(), "raffle.created"))

	return c.RedirectRoute("raffles.show", "raffle", r.ID)
}

// Show is the owner's view of one raffle with every number.
func (rc *RaffleController) Show(c *web.Ctx) error {
	r, err := rc.find(c)
	if err != nil {
		return err
	}

	ctx := c.Context()

	numbers, err := rc.raffles.Numbers(ctx, r.ID)
	if err != nil {
		return err
	}

	buyers, err := rc.raffles.Buyers(ctx, r.ID)
	if err != nil {
		return err
	}

	d, drawn, err := rc.draws.ForRaffle(ctx, r.ID)
	if err != nil {
		return err
	}

	active, err := rc.raffles.ActiveLinks(ctx, r.ID)
	if err != nil {
		return err
	}

	links := make([]map[string]any, len(active))
	for i, l := range active {
		links[i] = map[string]any{
			"number": l.Number, "url": rc.appURL + "/pay/" + l.Token, "expires_at": l.ExpiresAt,
			"buyer_name": l.Name, "buyer_phone": l.Phone,
		}
	}

	props := inertia.Props{
		"raffle": owned(r), "numbers": numbers, "buyers": buyers, "links": links, "draw": nil, "winner": nil,
	}
	if drawn {
		props["draw"] = drawResult(d)
		props["winner"] = winner(d, buyers)
	}

	return inertia.Render(c, "Raffles/Show", props)
}

// Edit shows the edit form.
func (rc *RaffleController) Edit(c *web.Ctx) error {
	r, err := rc.find(c)
	if err != nil {
		return err
	}

	return inertia.Render(c, "Raffles/Edit", inertia.Props{"raffle": owned(r), "limits": limits})
}

// Update validates and saves changes (locked fields stay as they are).
func (rc *RaffleController) Update(c *web.Ctx) error {
	r, err := rc.find(c)
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.RaffleSchema)
	if err != nil {
		return err
	}

	cancelled, err := rc.raffles.Update(c.Context(), &r, in)
	if err != nil {
		return err
	}

	msg := i18n.T(c.Context(), "raffle.updated")
	if cancelled > 0 {
		web.SecurityEvent(c.Request(), "raffle.links_cancelled", "raffle_id", r.ID, "count", cancelled, "reason", "edit")
		msg = i18n.T(c.Context(), "raffle.updated_links_cancelled", "count", cancelled)
	}

	inertia.Flash(c, "success", msg)

	return c.RedirectRoute("raffles.show", "raffle", r.ID)
}

// Destroy deletes a raffle that has no paid number.
func (rc *RaffleController) Destroy(c *web.Ctx) error {
	r, err := rc.find(c)
	if err != nil {
		return err
	}

	if err := rc.raffles.Delete(c.Context(), &r); err != nil {
		return err
	}

	inertia.Flash(c, "success", i18n.T(c.Context(), "raffle.deleted"))

	return c.RedirectRoute("dashboard")
}

// Archive archives a raffle (the only way out once a number is paid).
func (rc *RaffleController) Archive(c *web.Ctx) error {
	r, err := rc.find(c)
	if err != nil {
		return err
	}

	if err := rc.raffles.Archive(c.Context(), &r); err != nil {
		return err
	}

	inertia.Flash(c, "success", i18n.T(c.Context(), "raffle.archived"))

	return c.RedirectRoute("raffles.show", "raffle", r.ID)
}

// Public is the public raffle page: no personal data, statuses only.
func (rc *RaffleController) Public(c *web.Ctx) error {
	r, err := rc.raffles.BySlug(c.Context(), c.Param("slug"))
	if err != nil {
		return err
	}

	numbers, err := rc.raffles.Numbers(c.Context(), r.ID)
	if err != nil {
		return err
	}

	// One status per number, in order: numbers[i] is number i+1.
	statuses := make([]string, len(numbers))
	for i, n := range numbers {
		statuses[i] = n.Status
	}

	d, drawn, err := rc.draws.ForRaffle(c.Context(), r.ID)
	if err != nil {
		return err
	}

	props := inertia.Props{"raffle": public(r), "statuses": statuses, "draw": nil}
	if drawn {
		props["draw"] = drawResult(d)
	}

	return inertia.Render(c, "Raffles/Public", props)
}

func (rc *RaffleController) find(c *web.Ctx) (models.Raffle, error) {
	return ownedRaffle(c, rc.raffles)
}

// ownedRaffle loads the {raffle} route parameter scoped to the signed-in
// owner (404 for anyone else's).
func ownedRaffle(c *web.Ctx, s *raffles.Service) (models.Raffle, error) {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return models.Raffle{}, err
	}

	id, err := c.ParamInt("raffle")
	if err != nil {
		return models.Raffle{}, err
	}

	return s.ForOwner(c.Context(), u.ID, id)
}

// owned is the owner's view of a raffle (id for links, lock state).
func owned(r models.Raffle) map[string]any {
	p := public(r)
	p["id"] = r.ID
	p["duration_days"] = r.DurationDays
	p["locked"] = r.Locked()
	p["created_at"] = r.CreatedAt

	return p
}

// public lists the raffle fields anyone may see; the owner id stays out.
func public(r models.Raffle) map[string]any {
	return map[string]any{
		"slug":         r.Slug,
		"title":        r.Title,
		"description":  r.Description,
		"number_count": r.NumberCount,
		"price_reais":  r.PriceReais,
		"starts_at":    r.StartsAt,
		"ends_at":      r.EndsAt,
		"status":       r.Status,
	}
}

// drawResult is the public, auditable draw: everything needed to recompute
// the winner (docs/draw-algorithm.md) and no personal data.
func drawResult(d models.Draw) map[string]any {
	return map[string]any{
		"drawn_at":            d.DrawnAt,
		"seed":                d.Seed,
		"participants_count":  d.ParticipantsCount,
		"participants_sha256": d.ParticipantsSHA256,
		"winning_number":      d.WinningNumber,
		"algorithm":           d.Algorithm,
		"trigger":             d.Trigger,
	}
}

// winner is the paid buyer of the winning number (owner page only).
func winner(d models.Draw, buyers []raffles.Buyer) *raffles.Buyer {
	if d.WinningNumber == nil {
		return nil
	}

	for _, b := range buyers {
		if b.Number == *d.WinningNumber && b.Status == models.ReservationPaid {
			return &b
		}
	}

	return nil
}
