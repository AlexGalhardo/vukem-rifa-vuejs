package controllers

import (
	"io"
	"net/http"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

// WebhookController receives AbacatePay events. The route middleware has
// already checked the webhookSecret, the X-Webhook-Signature and the
// event-id dedupe before this runs.
type WebhookController struct {
	pay *payments.Service
}

// NewWebhookController returns a WebhookController.
func NewWebhookController(s *payments.Service) *WebhookController {
	return &WebhookController{pay: s}
}

// AbacatePay applies one event. A 2xx is answered only after the
// transaction committed; errors answer 5xx so AbacatePay retries.
func (wc *WebhookController) AbacatePay(c *web.Ctx) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	ev, err := abacatepay.ParseEvent(body)
	if err != nil {
		web.SecurityEvent(c.Request(), "webhook.payload_invalid", "provider", payments.Provider)
		return errs.New(http.StatusBadRequest, "Invalid webhook payload.")
	}

	outcome, err := wc.pay.Handle(c.Context(), ev, body)
	if err != nil {
		return err
	}

	c.Log().InfoContext(c.Context(), "payments: webhook", "event_id", ev.ID, "event", ev.Type, "outcome", string(outcome))

	return c.JSON(http.StatusOK, map[string]string{"outcome": string(outcome)})
}
