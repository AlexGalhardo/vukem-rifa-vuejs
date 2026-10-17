package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// MaxLivePublicLinksPerEmail caps the live public payment links one buyer
// email may hold in a raffle.
const MaxLivePublicLinksPerEmail = 5

// PublicCheckoutController lets anyone with the public raffle link buy a
// free number (no account needed) and signed-in buyers see their purchases.
type PublicCheckoutController struct {
	raffles      *raffles.Service
	reservations *reservations.Service
	acc          *accounts.Accounts
	perEmail     ratelimit.Limiter
}

// NewPublicCheckoutController returns a PublicCheckoutController; perEmail
// limits checkouts per buyer email (keyed by its hash).
func NewPublicCheckoutController(r *raffles.Service, res *reservations.Service, acc *accounts.Accounts, perEmail ratelimit.Limiter) *PublicCheckoutController {
	return &PublicCheckoutController{raffles: r, reservations: res, acc: acc, perEmail: perEmail}
}

// Reserve reserves a free {number} of the raffle {slug} for the buyer in
// the form and sends them to the payment page. Every refusal (taken number,
// raffle not on sale, caps, honeypot) is a form error on "number", so the
// page shows a friendly message and refreshes the grid.
func (pc *PublicCheckoutController) Reserve(c *web.Ctx) error {
	ctx := c.Context()

	r, err := pc.raffles.BySlug(ctx, c.Param("slug"))
	if err != nil {
		return err
	}

	number, err := numberParam(c, r)
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.PublicBuyerSchema)
	if err != nil {
		return err
	}

	refuse := func(key string) error {
		return errs.NewValidation().Add("number", i18n.T(ctx, key))
	}

	if in.Website != "" {
		web.SecurityEvent(c.Request(), "checkout.honeypot", "raffle_id", r.ID)
		return refuse("link.public.refused")
	}

	sum := sha256.Sum256([]byte(in.Email))
	if res, err := pc.perEmail.Allow(ctx, "email:"+hex.EncodeToString(sum[:])); err != nil || !res.Allowed {
		web.SecurityEvent(c.Request(), "checkout.email_rate_limited", "raffle_id", r.ID)
		return refuse("link.too_many_for_email")
	}

	link, err := pc.reservations.ReserveWith(ctx, r, number, in.Buyer(), reservations.ReserveOptions{
		Origin: models.OriginPublic, BuyerUserID: pc.buyerAccount(c, in.Email), MaxLivePerEmail: MaxLivePublicLinksPerEmail,
	})
	if err != nil {
		switch errs.Status(err) {
		case http.StatusConflict:
			return refuse("link.public.taken")
		case http.StatusTooManyRequests:
			return refuse("link.too_many_for_email")
		default:
			return err
		}
	}

	return c.Redirect("/pay/" + link.Token)
}

// buyerAccount is the account a public purchase belongs to: the signed-in
// user, else a verified account with the buyer email, else none (it is
// linked when that email gets verified).
func (pc *PublicCheckoutController) buyerAccount(c *web.Ctx, email string) *int64 {
	if pc.acc.Web.Check(c) {
		if u, err := pc.acc.Web.User(c); err == nil {
			return &u.ID
		}
	}

	u, err := orm.QueryOf[models.User](pc.acc.DB).Where("email", email).WhereNotNull("email_verified_at").First(c.Context())
	if err != nil {
		return nil
	}

	return &u.ID
}

// Purchase is one row of the buyer's history (their own data only).
type Purchase struct {
	RaffleTitle string     `json:"raffle_title"`
	RaffleSlug  string     `json:"raffle_slug"`
	Number      int        `json:"number"`
	AmountCents int        `json:"amount_cents"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	PaidAt      *time.Time `json:"paid_at"`
}

// Purchases is the signed-in user's history: paid numbers and the attempts
// that were not paid (pending, expired, cancelled).
func (pc *PublicCheckoutController) Purchases(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	list, err := orm.QueryOf[models.Reservation](pc.acc.DB).Where("buyer_user_id", u.ID).OrderBy("created_at", orm.Desc).Get(c.Context())
	if err != nil {
		return err
	}

	titles := map[int64]models.Raffle{}
	paid, unpaid := []Purchase{}, []Purchase{}

	for _, res := range list {
		r, ok := titles[res.RaffleID]
		if !ok {
			if r, err = orm.QueryOf[models.Raffle](pc.acc.DB).Find(c.Context(), res.RaffleID); err != nil && !errors.Is(err, orm.ErrNotFound) {
				return err
			}

			titles[res.RaffleID] = r
		}

		p := Purchase{
			RaffleTitle: r.Title, RaffleSlug: r.Slug, Number: res.Number, AmountCents: res.AmountCents,
			Status: res.Status, CreatedAt: res.CreatedAt, ExpiresAt: res.ExpiresAt,
		}
		if res.PaidAt.Valid {
			t := res.PaidAt.V
			p.PaidAt = &t
		}

		if res.Status == models.ReservationPaid {
			paid = append(paid, p)
		} else {
			unpaid = append(unpaid, p)
		}
	}

	return inertia.Render(c, "Account/Purchases", inertia.Props{"paid": paid, "unpaid": unpaid})
}
