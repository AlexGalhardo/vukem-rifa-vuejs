package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// A08: deliveries without the right secret or signature are refused before
// the body is parsed, and nothing is recorded.
func TestSecurity_WebhookRejectsForgedDeliveries(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 20)
	body := paidEvent("log_forged", *res.ChargeID, res.ID, 1000)
	tampered := []byte(strings.Replace(string(body), `"amount":1000`, `"amount":100`, 1))

	for name, tc := range map[string]struct {
		secret, signature string
		body              []byte
	}{
		"forged signature":       {webhookSecret, "c2lnbmF0dXJl", body},
		"signed with the secret": {webhookSecret, abacatepay.Sign([]byte(webhookSecret), body), body},
		"missing signature":      {webhookSecret, "", body},
		"tampered body":          {webhookSecret, sign(body), tampered},
		"wrong secret":           {"guess", sign(body), body},
		"missing secret":         {"", sign(body), body},
	} {
		t.Run(name, func(t *testing.T) {
			deliver(t, f.app, tc.secret, tc.signature, tc.body).AssertUnauthorized()
		})
	}

	vukemtest.AssertDatabaseCount(t, f.app, "payment_events", 0)
	vukemtest.AssertDatabaseCount(t, f.app, "ledger_entries", 0)

	if got := f.numberStatus(t, 20); got != models.NumberReserved {
		t.Fatalf("number = %s", got)
	}
}

// An app without ABACATEPAY_WEBHOOK_SECRET fails closed.
func TestSecurity_WebhookWithoutConfiguredSecretFailsClosed(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"ABACATEPAY_FAKE": "true"}})
	body := paidEvent("log_x", "pix_char_x", 1, 1000)

	deliver(t, app, "", sign(body), body).AssertUnauthorized()
	vukemtest.AssertDatabaseCount(t, app, "payment_events", 0)
}

// A08: a replayed delivery (AbacatePay retries with the same id) is
// answered 200 without running again.
func TestSecurity_WebhookReplayIsANoOp(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 21)
	body := paidEvent("log_replay", *res.ChargeID, res.ID, 1000)

	deliverSigned(t, f.app, body).AssertOK().AssertJSON(map[string]string{"outcome": "paid"})
	deliverSigned(t, f.app, body).AssertOK().AssertHeader("Webhook-Duplicate", "true")

	vukemtest.AssertDatabaseCount(t, f.app, "payment_events", 1)

	if n, _ := ledgerSum(t, f, models.LedgerSaleOwnerShare); n != 1 {
		t.Fatalf("owner share rows = %d", n)
	}

	if n, _ := ledgerSum(t, f, models.LedgerPlatformFee); n != 1 {
		t.Fatalf("fee rows = %d", n)
	}
}

// Even if the dedupe cache forgot the event, the unique event_id keeps
// the second delivery from paying twice.
func TestSecurity_WebhookReplayAfterCacheLossIsANoOp(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 22)
	body := paidEvent("log_replay_2", *res.ChargeID, res.ID, 1000)

	deliverSigned(t, f.app, body).AssertOK()

	c, err := f.app.Cache()
	if err != nil {
		t.Fatal(err)
	}

	// workaround(vukem v0.28.0): webhook has no exported dedupe key or Forget, so
	// the test spells the key format (length-prefixed namespace) by hand.
	if err := c.Store().Delete(t.Context(), "webhook:event:10:abacatepay:log_replay_2"); err != nil {
		t.Fatal(err)
	}

	deliverSigned(t, f.app, body).AssertOK().AssertJSON(map[string]string{"outcome": "duplicate"})

	if n, _ := ledgerSum(t, f, models.LedgerSaleOwnerShare); n != 1 {
		t.Fatalf("owner share rows = %d", n)
	}
}

// Malformed but correctly signed bodies are a 400, never a 5xx retry loop.
func TestSecurity_WebhookInvalidPayload(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())

	for _, body := range []string{`{"id":"log_bad","event":"transparent.completed","data":{"transparent":{"id":"x"}}}`, `{"event":"x"}`, `[]`} {
		deliverSigned(t, f.app, []byte(body)).AssertStatus(http.StatusBadRequest)
	}

	vukemtest.AssertDatabaseCount(t, f.app, "payment_events", 0)
}

// The confirm endpoint is rate limited per link and client.
func TestSecurity_ConfirmIsRateLimited(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	link := f.reserve(t, 23)
	c := f.app.Client(t).Inertia("")

	for range 10 {
		c.PostJSON("/pay/"+link.Token+"/confirm", map[string]any{}).AssertRedirect("/pay/" + link.Token)
	}

	c.PostJSON("/pay/"+link.Token+"/confirm", map[string]any{}).AssertStatus(http.StatusTooManyRequests)
}

// A01: a cross-site page cannot confirm a buyer's link (CSRF).
func TestSecurity_CrossSiteConfirmBlocked(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	link := f.reserve(t, 24)

	f.app.Client(t).WithHeader("Origin", "https://evil.example").WithHeader("Sec-Fetch-Site", "cross-site").
		PostJSON("/pay/"+link.Token+"/confirm", map[string]any{}).AssertForbidden()

	if res := f.reservation(t, link.Reservation.ID); res.Status != models.ReservationPending {
		t.Fatalf("status = %s", res.Status)
	}
}

// The fake gateway helpers exist only when ABACATEPAY_FAKE=true.
func TestSecurity_DevPaymentHelpersOffByDefault(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t)
	link := f.reserve(t, 25)

	f.app.Client(t).PostJSON("/dev/pay/"+link.Token+"/simulate", map[string]any{}).AssertNotFound()
}
