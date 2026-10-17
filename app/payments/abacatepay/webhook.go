package abacatepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AlexGalhardo/vukem/z"
)

// Webhook events the app reacts to; every other event is recorded and
// ignored.
const (
	EventTransparentCompleted = "transparent.completed"
)

// WebhookPublicKey is the key AbacatePay signs X-Webhook-Signature with
// (HMAC-SHA256, base64, raw body). It is published in the docs
// (docs.abacatepay.com/pages/webhooks/security) and is not a secret: the
// signature only proves body integrity, the webhookSecret query parameter
// proves the origin. ABACATEPAY_WEBHOOK_PUBLIC_KEY overrides it if
// AbacatePay rotates it.
const WebhookPublicKey = "t9dXRhHHo3yDEj5pVDYz0frf7q6bMKyMRmxxCPIPp3RCplBfXRxqlC6ZpiWmOqj4L63qEaeUOtrCI8P0VMUgo6iIga2ri9ogaHFs0WIIywSMg0q7RmBfybe1E5XJcfC4IW3alNqym0tXoAKkzvfEjZxV6bE0oG2zJrNNYmUCKZyV0KZ3JS8Votf9EAWWYdiDkMkpbMdPggfh1EqHlVkMiTady6jOR3hyzGEHrIz2Ret0xHKMbiqkr9HS1JhNHDX9"

// SignatureHeader carries the body signature.
const SignatureHeader = "X-Webhook-Signature"

// Sign returns the base64 HMAC-SHA256 of body, as AbacatePay sends it (for
// tests and the dev simulator).
func Sign(key, body []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write(body)

	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

// Event is a webhook delivery. ID is the same on every retry of an event.
// The charge fields are set for transparent.* events only.
type Event struct {
	ID      string
	Type    string
	DevMode bool

	ChargeID     string
	ExternalID   string
	ChargeStatus string
	// AmountCents is the paid amount when the provider reports one, else
	// the charge amount.
	AmountCents int64
}

// ErrInvalidEvent is a webhook body that is not a valid event.
var ErrInvalidEvent = errors.New("abacatepay: invalid webhook event")

type envelopeDTO struct {
	ID      string         `json:"id"`
	Event   string         `json:"event"`
	DevMode bool           `json:"devMode"`
	Data    map[string]any `json:"data"`
}

type transparentDTO struct {
	ID         string  `json:"id"`
	ExternalID *string `json:"externalId"`
	Amount     int64   `json:"amount"`
	PaidAmount *int64  `json:"paidAmount"`
	Status     string  `json:"status"`
}

// Only the fields the app uses are validated, so new provider fields do not
// break the endpoint (AbacatePay asks for this).
var envelopeSchema = z.Struct[envelopeDTO](z.Shape{
	"id":      z.String().Min(1).Max(191),
	"event":   z.String().Min(1).Max(64),
	"devMode": z.Bool().Optional(),
	"data": z.Custom(func(in any) (map[string]any, error) {
		m, ok := in.(map[string]any)
		if !ok && in != nil {
			return nil, errors.New("data must be an object")
		}

		return m, nil
	}).Optional(),
})

var transparentSchema = z.Struct[transparentDTO](z.Shape{
	"id":         z.String().Min(1).Max(191),
	"externalId": z.String().Max(191).Nullable().Optional(),
	"amount":     z.Int64().Positive(),
	"paidAmount": z.Int64().Positive().Nullable().Optional(),
	"status":     z.String().Min(1).Max(32),
})

// ParseEvent validates a webhook body. Call it only after the secret and
// the signature were verified.
func ParseEvent(body []byte) (Event, error) {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Event{}, fmt.Errorf("%w: %w", ErrInvalidEvent, err)
	}

	env, err := envelopeSchema.Parse(raw)
	if err != nil {
		return Event{}, fmt.Errorf("%w: %w", ErrInvalidEvent, err)
	}

	ev := Event{ID: env.ID, Type: env.Event, DevMode: env.DevMode}
	if ev.Type != EventTransparentCompleted {
		return ev, nil
	}

	tr, err := transparentSchema.Parse(env.Data["transparent"])
	if err != nil {
		return Event{}, fmt.Errorf("%w: %w", ErrInvalidEvent, err)
	}

	ev.ChargeID, ev.ChargeStatus, ev.AmountCents = tr.ID, tr.Status, tr.Amount
	if tr.ExternalID != nil {
		ev.ExternalID = *tr.ExternalID
	}

	if tr.PaidAmount != nil {
		ev.AmountCents = *tr.PaidAmount
	}

	return ev, nil
}
