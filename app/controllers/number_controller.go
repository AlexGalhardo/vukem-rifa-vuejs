package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// NumberController lets an owner generate and cancel payment links for
// the numbers of their raffles.
type NumberController struct {
	raffles      *raffles.Service
	reservations *reservations.Service
	appURL       string
}

// NewNumberController returns a NumberController; appURL (APP_URL) is the
// base of the payment links.
func NewNumberController(r *raffles.Service, res *reservations.Service, appURL string) *NumberController {
	return &NumberController{raffles: r, reservations: res, appURL: strings.TrimRight(appURL, "/")}
}

// LinkResponse is the payment link, returned once: only the token's hash is
// stored, so it can never be shown again.
type LinkResponse struct {
	Number    int       `json:"number"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	BuyerName string    `json:"buyer_name"`
	Phone     string    `json:"buyer_phone"`
}

// Link reserves {number} for the buyer in the body and returns its payment
// link as JSON (the page calls it with fetch so the token never goes
// through the session or an Inertia page).
func (nc *NumberController) Link(c *web.Ctx) error {
	r, err := ownedRaffle(c, nc.raffles)
	if err != nil {
		return err
	}

	number, err := numberParam(c, r)
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.BuyerSchema)
	if err != nil {
		return err
	}

	link, err := nc.reservations.Reserve(c.Context(), r, number, in)
	if err != nil {
		return err
	}

	res := link.Reservation

	return c.JSON(http.StatusCreated, LinkResponse{
		Number: res.Number, URL: nc.appURL + "/pay/" + link.Token, ExpiresAt: res.ExpiresAt,
		BuyerName: res.BuyerName, Phone: res.BuyerPhone,
	})
}

// Cancel releases the live payment link of {number}.
func (nc *NumberController) Cancel(c *web.Ctx) error {
	r, err := ownedRaffle(c, nc.raffles)
	if err != nil {
		return err
	}

	number, err := numberParam(c, r)
	if err != nil {
		return err
	}

	if err := nc.reservations.Cancel(c.Context(), r.ID, number); err != nil {
		return err
	}

	inertia.Flash(c, "success", i18n.T(c.Context(), "link.cancelled"))

	return c.RedirectRoute("raffles.show", "raffle", r.ID)
}

// numberParam reads {number}; numbers outside 1..N are 404.
func numberParam(c *web.Ctx, r models.Raffle) (int, error) {
	n, err := c.ParamInt("number")
	if err != nil || n < 1 || n > int64(r.NumberCount) {
		return 0, errs.New(http.StatusNotFound, i18n.T(c.Context(), "link.number_not_found"))
	}

	return int(n), nil
}
