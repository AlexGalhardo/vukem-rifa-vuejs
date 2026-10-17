package apiv1

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"
	"github.com/AlexGalhardo/vukem/z"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/export"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Guard is the name of the API key guard (accounts.API).
const Guard = "api"

// Handlers serve the API; every handler scopes data to the key's owner.
type Handlers struct {
	DB           *database.DB
	Raffles      *raffles.Service
	Reservations *reservations.Service
	Export       *export.Service
	// URL is APP_URL without a trailing slash (public and payment links).
	URL string
}

// Middleware the routes need, built by the caller (routes/api.go).
type Middleware struct {
	Read, Write  router.Middleware // ability checks
	Idempotency  router.Middleware // payment links
	LinkThrottle router.Middleware // payment links
}

var security = []string{"bearer"}

// Register adds the v1 endpoints to g (mounted at /api/v1).
func (h *Handlers) Register(g *router.Group, mw Middleware) {
	op := func(name, summary string) api.Op {
		return api.Op{Name: name, Summary: summary, Tags: []string{"raffles"}, Security: security, ValidateResponse: true}
	}

	api.Register(g, "GET /raffles", op("raffles.index", "List your raffles (paginated)"), h.list).Use(mw.Read)
	api.Register(g, "POST /raffles", op("raffles.store", "Create a raffle"), h.create).Use(mw.Write)
	api.Register(g, "GET /raffles/{id}", op("raffles.show", "Show a raffle"), h.show).Use(mw.Read)
	api.Register(g, "PATCH /raffles/{id}", op("raffles.update", "Update a raffle"), h.update).Use(mw.Write)
	api.Register(g, "DELETE /raffles/{id}", op("raffles.destroy", "Delete a raffle without paid numbers"), h.destroy).
		Use(mw.Write)

	archive := op("raffles.archive", "Archive a raffle")
	archive.Status = http.StatusOK
	api.Register(g, "POST /raffles/{id}/archive", archive, h.archive).Use(mw.Write)

	api.Register(g, "GET /raffles/{id}/numbers", op("raffles.numbers", "List every number of a raffle"), h.numbers).
		Use(mw.Read)

	link := op("raffles.payment_links.store", "Reserve a number for 10 minutes and get its payment link")
	link.Description = "Send an Idempotency-Key header to retry safely: the same key replays the first response."
	api.Register(g, "POST /raffles/{id}/numbers/{number}/payment-links", link, h.paymentLink).
		Use(mw.Write, mw.LinkThrottle, mw.Idempotency)

	exp := op("raffles.export", "Download the audit export (JSON or CSV)")
	// Not api.Register: the CSV variant is not JSON. The document describes the
	// JSON variant.
	rt := g.Get("/raffles/{id}/export", h.export).Name("raffles.export").Use(mw.Read)
	rt.Doc(&api.Operation{
		Op: exp, Method: http.MethodGet, Path: rt.Path(), Params: []string{"id"},
		In: reflect.TypeFor[ExportQuery](), Out: reflect.TypeFor[export.Export](),
	})

	me := api.Op{
		Name: "me.balance", Summary: "Your balance (owner shares minus payouts and refunds)",
		Tags: []string{"account"}, Security: security, ValidateResponse: true,
	}
	api.Register(g, "GET /me/balance", me, h.balance).Use(mw.Read)
}

func (h *Handlers) owner(c *web.Ctx) (models.User, error) { return auth.User[models.User](c, Guard) }

func (h *Handlers) raffle(c *web.Ctx, id int64) (models.Raffle, error) {
	u, err := h.owner(c)
	if err != nil {
		return models.Raffle{}, err
	}

	return h.Raffles.ForOwner(c.Context(), u.ID, id)
}

func (h *Handlers) present(c *web.Ctx, r models.Raffle) (Raffle, error) {
	counts, err := h.Raffles.Counts(c.Context(), r.ID)
	if err != nil {
		return Raffle{}, err
	}

	return toRaffle(r, counts, h.URL), nil
}

func (h *Handlers) list(c *web.Ctx, in ListRaffles) (RafflePage, error) {
	u, err := h.owner(c)
	if err != nil {
		return RafflePage{}, err
	}

	page, perPage := max(in.Page, 1), in.PerPage
	if perPage == 0 {
		perPage = 20
	}

	p, err := orm.QueryOf[models.Raffle](h.DB).Where("user_id", u.ID).OrderBy("id", orm.Desc).Paginate(c.Context(), page, perPage)
	if err != nil {
		return RafflePage{}, err
	}

	out := RafflePage{
		Data:  make([]Raffle, 0, len(p.Items)),
		Meta:  PageMeta{Page: p.Page, PerPage: p.PerPage, Total: p.Total, LastPage: p.LastPage},
		Links: PageLinks{Self: pageURL(c, p.Page, p.PerPage)},
	}

	for _, r := range p.Items {
		item, err := h.present(c, r)
		if err != nil {
			return RafflePage{}, err
		}

		out.Data = append(out.Data, item)
	}

	if p.Page < p.LastPage {
		out.Links.Next = new(pageURL(c, p.Page+1, p.PerPage))
	}

	if p.Page > 1 {
		out.Links.Prev = new(pageURL(c, min(p.Page-1, max(p.LastPage, 1)), p.PerPage))
	}

	return out, nil
}

func pageURL(c *web.Ctx, page, perPage int) string {
	q := url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
	return c.Request().URL.Path + "?" + q.Encode()
}

// canonical runs the app's own request schema (trim, normalization, the
// rules the browser forms use) on top of the tag validation.
func canonical[T any](schema z.Schema[T], input map[string]any) (T, error) {
	return web.ParseInput(schema, input)
}

func (h *Handlers) create(c *web.Ctx, in CreateRaffle) (Raffle, error) {
	u, err := h.owner(c)
	if err != nil {
		return Raffle{}, err
	}

	req, err := canonical(requests.RaffleSchema, map[string]any{
		"title": in.Title, "description": in.Description, "number_count": in.NumberCount,
		"price_reais": in.PriceReais, "duration_days": in.DurationDays,
	})
	if err != nil {
		return Raffle{}, err
	}

	r, err := h.Raffles.Create(c.Context(), u.ID, req)
	if err != nil {
		return Raffle{}, err
	}

	return h.present(c, r)
}

func (h *Handlers) show(c *web.Ctx, in RaffleID) (Raffle, error) {
	r, err := h.raffle(c, in.ID)
	if err != nil {
		return Raffle{}, err
	}

	return h.present(c, r)
}

func (h *Handlers) update(c *web.Ctx, in UpdateRaffle) (Raffle, error) {
	r, err := h.raffle(c, in.ID)
	if err != nil {
		return Raffle{}, err
	}

	fields := map[string]any{
		"title": r.Title, "description": r.Description, "number_count": r.NumberCount,
		"price_reais": r.PriceReais, "duration_days": r.DurationDays,
	}
	set := func(key string, v any, ok bool) {
		if ok {
			fields[key] = v
		}
	}

	set("title", deref(in.Title), in.Title != nil)
	set("description", deref(in.Description), in.Description != nil)
	set("number_count", deref(in.NumberCount), in.NumberCount != nil)
	set("price_reais", deref(in.PriceReais), in.PriceReais != nil)
	set("duration_days", deref(in.DurationDays), in.DurationDays != nil)

	req, err := canonical(requests.RaffleSchema, fields)
	if err != nil {
		return Raffle{}, err
	}

	cancelled, err := h.Raffles.Update(c.Context(), &r, req)
	if err != nil {
		return Raffle{}, err
	}

	if cancelled > 0 {
		web.AuditEvent(c.Request(), "raffle.links_cancelled", "raffle_id", r.ID, "count", cancelled, "reason", "edit")
	}

	return h.present(c, r)
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}

	return *p
}

func (h *Handlers) destroy(c *web.Ctx, in RaffleID) (struct{}, error) {
	r, err := h.raffle(c, in.ID)
	if err != nil {
		return struct{}{}, err
	}

	return struct{}{}, h.Raffles.Delete(c.Context(), &r)
}

func (h *Handlers) archive(c *web.Ctx, in RaffleID) (Raffle, error) {
	r, err := h.raffle(c, in.ID)
	if err != nil {
		return Raffle{}, err
	}

	if err := h.Raffles.Archive(c.Context(), &r); err != nil {
		return Raffle{}, err
	}

	return h.present(c, r)
}

func (h *Handlers) numbers(c *web.Ctx, in RaffleID) (NumberList, error) {
	r, err := h.raffle(c, in.ID)
	if err != nil {
		return NumberList{}, err
	}

	list, err := h.Raffles.Numbers(c.Context(), r.ID)
	if err != nil {
		return NumberList{}, err
	}

	out := NumberList{Data: make([]RaffleNumber, 0, len(list))}
	for _, n := range list {
		out.Data = append(out.Data, RaffleNumber{Number: n.Number, Status: n.Status, PaidAt: ptrTime(n.PaidAt)})
	}

	return out, nil
}

func (h *Handlers) paymentLink(c *web.Ctx, in CreatePaymentLink) (PaymentLink, error) {
	r, err := h.raffle(c, in.ID)
	if err != nil {
		return PaymentLink{}, err
	}

	buyer, err := canonical(requests.BuyerSchema, map[string]any{
		"buyer_name": in.BuyerName, "buyer_email": in.BuyerEmail, "buyer_phone": in.BuyerPhone,
	})
	if err != nil {
		return PaymentLink{}, err
	}

	link, err := h.Reservations.ReserveWith(c.Context(), r, in.Number, buyer, reservations.ReserveOptions{Origin: models.OriginAPI})
	if err != nil {
		return PaymentLink{}, err
	}

	return PaymentLink{
		URL:       h.URL + "/pay/" + link.Token,
		ExpiresAt: link.Reservation.ExpiresAt.UTC(),
		Number:    link.Reservation.Number,
	}, nil
}

func (h *Handlers) export(c *web.Ctx) error {
	input := c.QueryInput()
	input["id"] = c.Param("id")

	if _, ok := input["format"]; !ok {
		input["format"] = "json"
	}

	in, err := web.ParseInput(z.For[ExportQuery](), input)
	if err != nil {
		return err
	}

	r, err := h.raffle(c, in.ID)
	if err != nil {
		return err
	}

	e, err := h.Export.Build(c.Context(), r)
	if err != nil {
		return err
	}

	return controllers.Download(c, e, in.Format)
}

func (h *Handlers) balance(c *web.Ctx, _ struct{}) (Balance, error) {
	u, err := h.owner(c)
	if err != nil {
		return Balance{}, err
	}

	b, err := ledger.Balance(c.Context(), h.DB, u.ID)
	if err != nil {
		return Balance{}, err
	}

	return Balance{BalanceCents: b, Currency: "BRL"}, nil
}
