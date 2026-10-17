package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

// PayController is the buyer checkout of a payment link: no account, the
// unguessable token is the only credential.
type PayController struct {
	pay *payments.Service
}

// NewPayController returns a PayController.
func NewPayController(s *payments.Service) *PayController { return &PayController{pay: s} }

// Show is the checkout page. Expired or cancelled links answer 410 and
// unknown tokens 404, both with the "link expired" page.
func (pc *PayController) Show(c *web.Ctx) error {
	co, err := pc.pay.Load(c.Context(), c.Param("token"))
	if err != nil {
		return expiredOr(c, err)
	}

	if co.State == payments.StateExpired {
		return renderStatus(c, http.StatusGone, "Pay/Show", expiredProps())
	}

	return inertia.Render(c, "Pay/Show", checkoutProps(co, ""))
}

// Confirm creates (or reuses) the PIX charge and goes back to the page,
// which then shows the QR code.
func (pc *PayController) Confirm(c *web.Ctx) error {
	token := c.Param("token")

	co, err := pc.pay.Confirm(c.Context(), token)
	if err != nil {
		if errs.Status(err) == http.StatusServiceUnavailable && !c.WantsJSON() {
			var he *errs.HTTPError
			if errors.As(err, &he) {
				return renderStatus(c, http.StatusServiceUnavailable, "Pay/Show", checkoutProps(co, he.Message))
			}
		}

		return expiredOr(c, err)
	}

	return c.Redirect("/pay/" + token)
}

// Status is polled by the page every few seconds.
func (pc *PayController) Status(c *web.Ctx) error {
	st, err := pc.pay.Status(c.Context(), c.Param("token"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{"status": st})
}

// expiredOr renders the expired page for 404 and 410 errors and returns
// any other error.
func expiredOr(c *web.Ctx, err error) error {
	status := errs.Status(err)
	if (status == http.StatusNotFound || status == http.StatusGone) && !c.WantsJSON() {
		return renderStatus(c, status, "Pay/Show", expiredProps())
	}

	return err
}

func expiredProps() inertia.Props {
	return inertia.Props{"state": payments.StateExpired, "checkout": nil, "pix": nil, "error": nil}
}

// checkoutProps is what the buyer may see: their own data, masked, and
// nothing about other buyers.
func checkoutProps(co payments.Checkout, errMsg string) inertia.Props {
	res := co.Reservation

	var pix any
	if co.State == payments.StateLive && res.BrCode != nil && res.BrCodeBase64 != nil {
		pix = map[string]string{"br_code": *res.BrCode, "br_code_base64": *res.BrCodeBase64}
	}

	var errProp any
	if errMsg != "" {
		errProp = errMsg
	}

	return inertia.Props{
		"state": co.State,
		"checkout": map[string]any{
			"raffle_title": co.Raffle.Title,
			"raffle_slug":  co.Raffle.Slug,
			"number":       res.Number,
			"amount_cents": res.AmountCents,
			"buyer_name":   res.BuyerName,
			"buyer_email":  payments.MaskEmail(res.BuyerEmail),
			"buyer_phone":  payments.MaskPhone(res.BuyerPhone),
			"status":       res.Status,
			"seconds_left": co.SecondsLeft,
		},
		"pix":   pix,
		"error": errProp,
	}
}

// renderStatus renders an Inertia page with a status other than 200.
//
// workaround(vukem v0.22.0): inertia.Render always answers 200 and c.Status writes the header before Render sets its own.
func renderStatus(c *web.Ctx, status int, component string, props inertia.Props) error {
	w := &statusWriter{ResponseWriter: c.Writer(), status: status}
	return inertia.Render(web.NewCtx(w, c.Request(), c.Services()), component, props)
}

type statusWriter struct {
	http.ResponseWriter

	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wrote && code == http.StatusOK {
		code = w.status
	}

	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// DevPay holds the development-only helpers of the fake gateway (never
// registered in production: the fake refuses to boot there).
type DevPay struct {
	pay  *payments.Service
	fake *abacatepay.Fake
}

// NewDevPay returns the fake-gateway helpers.
func NewDevPay(s *payments.Service, f *abacatepay.Fake) *DevPay { return &DevPay{pay: s, fake: f} }

// Simulate pays the link's fake PIX charge and delivers the resulting
// transparent.completed event to the same handler the webhook uses.
func (d *DevPay) Simulate(c *web.Ctx) error {
	ctx := c.Context()

	co, err := d.pay.Load(ctx, c.Param("token"))
	if err != nil {
		return err
	}

	res := co.Reservation
	if res.ChargeID == nil {
		return errs.New(http.StatusConflict, "No PIX charge to pay yet.")
	}

	ch, err := d.fake.SimulatePayment(ctx, *res.ChargeID)
	if err != nil {
		return errs.New(http.StatusConflict, "Unknown fake charge (the app restarted?).")
	}

	ev := abacatepay.Event{
		ID: "log_fake_" + randomHex(12), Type: abacatepay.EventTransparentCompleted, DevMode: true,
		ChargeID: ch.ID, ExternalID: strconv.FormatInt(res.ID, 10), ChargeStatus: ch.Status, AmountCents: ch.AmountCents,
	}

	payload, err := json.Marshal(map[string]any{
		"id": ev.ID, "event": ev.Type, "apiVersion": 2, "devMode": true,
		"data": map[string]any{"transparent": map[string]any{
			"id": ch.ID, "externalId": ev.ExternalID, "amount": ch.AmountCents, "status": ch.Status,
		}},
	})
	if err != nil {
		return err
	}

	outcome, err := d.pay.Handle(ctx, ev, payload)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{"outcome": string(outcome)})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // crypto/rand.Read never fails (Go 1.24+)

	return strings.ToLower(hex.EncodeToString(b))
}
