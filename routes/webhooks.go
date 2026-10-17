package routes

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"
	"github.com/AlexGalhardo/vukem/webhook"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

// Webhooks registers POST /webhooks/abacatepay. The checks run before the
// body is parsed and fail closed: the webhookSecret query parameter
// (ABACATEPAY_WEBHOOK_SECRET, proves the origin), then X-Webhook-Signature
// (HMAC-SHA256 of the raw body with AbacatePay's published key, proves
// integrity), then the event-id dedupe in CACHE_STORE. No CSRF bypass is
// needed: vukem's csrf only rejects cross-site browser requests (Origin /
// Sec-Fetch-Site), and AbacatePay's server-to-server POST sends neither.
func Webhooks(r *router.Router, app *vukem.App) error {
	db, err := app.DB()
	if err != nil {
		return err
	}

	c, err := app.Cache()
	if err != nil {
		return err
	}

	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	key := app.Source.Get("ABACATEPAY_WEBHOOK_PUBLIC_KEY")
	if key == "" {
		key = abacatepay.WebhookPublicKey
	}

	pay := payments.New(db, app.Now, nil, app.Queue, app.Log)
	wc := controllers.NewWebhookController(pay)

	r.Post("/webhooks/abacatepay", wc.AbacatePay).Name("webhooks.abacatepay").Use(
		ratelimit.Middleware(limiters("webhook-abacatepay", 120, time.Minute), ratelimit.ByIP),
		webhookSecret(app.Source.Get("ABACATEPAY_WEBHOOK_SECRET")),
		webhook.Middleware(abacatepay.SignatureHeader, []byte(key), 1<<20),
		webhook.Dedupe(c.Store(), eventID, webhook.DedupeOptions{}),
	)

	return nil
}

var errWebhookSecret = errs.New(http.StatusUnauthorized, "Invalid webhook secret.")

// webhookSecret compares the webhookSecret query parameter in constant
// time; an unset secret rejects every delivery.
func webhookSecret(secret string) router.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.URL.Query().Get("webhookSecret")
			if secret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
				web.SecurityEvent(r, "webhook.secret_invalid", "provider", payments.Provider)
				web.Fail(w, r, errWebhookSecret)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// eventID reads the delivery id ("id", the same on every retry) from the
// already verified body and restores it for the handler.
func eventID(r *http.Request) string {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ""
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	var ev struct {
		ID string `json:"id"`
	}

	if json.Unmarshal(body, &ev) != nil || len(ev.ID) > 191 {
		return ""
	}

	return ev.ID
}
