package controllers

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// RaffleController is the owner's raffle management (dashboard) and the
// public raffle page.
type RaffleController struct {
	raffles *raffles.Service
}

// NewRaffleController returns a RaffleController.
func NewRaffleController(s *raffles.Service) *RaffleController { return &RaffleController{raffles: s} }

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

	return inertia.Render(c, "Dashboard", inertia.Props{"raffles": list})
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

	numbers, err := rc.raffles.Numbers(c.Context(), r.ID)
	if err != nil {
		return err
	}

	return inertia.Render(c, "Raffles/Show", inertia.Props{"raffle": owned(r), "numbers": numbers})
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

	if err := rc.raffles.Update(c.Context(), &r, in); err != nil {
		return err
	}

	inertia.Flash(c, "success", i18n.T(c.Context(), "raffle.updated"))

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

	return inertia.Render(c, "Raffles/Public", inertia.Props{"raffle": public(r), "statuses": statuses})
}

func (rc *RaffleController) find(c *web.Ctx) (models.Raffle, error) {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return models.Raffle{}, err
	}

	id, err := c.ParamInt("raffle")
	if err != nil {
		return models.Raffle{}, err
	}

	return rc.raffles.ForOwner(c.Context(), u.ID, id)
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
