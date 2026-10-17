package abacatepay_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

// From docs.abacatepay.com/pages/webhooks/events/transparent (PIX tab).
const completed = `{"id":"log_Qw8eR2tY4uI6oP1aS3dF5gH7","event":"transparent.completed","apiVersion":2,"devMode":false,
"data":{"transparent":{"id":"pix_char_mXq2Lk7vB9nR4sT8yW1zC6dF","externalId":null,"amount":5000,"paidAmount":null,
"platformFee":80,"status":"PAID","items":[],"methods":[],"frequency":"ONE_TIME","coupons":[],"devMode":false,
"customerId":"cust_aB3dE5fG7hJ9kL1mN3pQ5rS7","createdAt":"2026-10-12T19:00:00.000Z","updatedAt":"2026-10-12T19:00:05.000Z",
"receiptUrl":null,"metadata":{"pedidoId":"pedido-456"}},"customer":{"id":"cust_aB3dE5fG7hJ9kL1mN3pQ5rS7","name":"Maria Santos",
"email":"maria@exemplo.com","taxId":"529.***.247-**"},"payerInformation":{"method":"PIX"}}}`

func TestParseEvent_TransparentCompleted(t *testing.T) {
	t.Parallel()

	ev, err := abacatepay.ParseEvent([]byte(completed))
	if err != nil {
		t.Fatal(err)
	}

	if ev.ID != "log_Qw8eR2tY4uI6oP1aS3dF5gH7" || ev.Type != abacatepay.EventTransparentCompleted ||
		ev.ChargeID != "pix_char_mXq2Lk7vB9nR4sT8yW1zC6dF" || ev.AmountCents != 5000 ||
		ev.ChargeStatus != abacatepay.StatusPaid || ev.ExternalID != "" {
		t.Fatalf("event = %+v", ev)
	}
}

func TestParseEvent_PaidAmountWins(t *testing.T) {
	t.Parallel()

	body := strings.Replace(completed, `"paidAmount":null`, `"paidAmount":4900`, 1)
	body = strings.Replace(body, `"externalId":null`, `"externalId":"42"`, 1)

	ev, err := abacatepay.ParseEvent([]byte(body))
	if err != nil || ev.AmountCents != 4900 || ev.ExternalID != "42" {
		t.Fatalf("event = %+v, %v", ev, err)
	}
}

func TestParseEvent_UnknownEventsKeepTheEnvelope(t *testing.T) {
	t.Parallel()

	ev, err := abacatepay.ParseEvent([]byte(`{"id":"log_1","event":"payout.completed","apiVersion":2,"devMode":true,"data":{"payout":{}}}`))
	if err != nil || ev.ID != "log_1" || ev.Type != "payout.completed" || ev.ChargeID != "" {
		t.Fatalf("event = %+v, %v", ev, err)
	}
}

func TestParseEvent_Invalid(t *testing.T) {
	t.Parallel()

	for _, body := range []string{
		``, `nope`, `{}`, `{"event":"transparent.completed"}`,
		`{"id":"log_1","event":"transparent.completed","data":{}}`,
		strings.Replace(completed, `"amount":5000`, `"amount":0`, 1),
		strings.Replace(completed, `"log_Qw8eR2tY4uI6oP1aS3dF5gH7"`, `"`+strings.Repeat("x", 300)+`"`, 1),
	} {
		if _, err := abacatepay.ParseEvent([]byte(body)); err == nil {
			t.Errorf("ParseEvent(%.80q…%d) = nil error", body, len(body))
		}
	}
}

func TestSignature(t *testing.T) {
	t.Parallel()

	body := []byte(completed)
	mac := hmac.New(sha256.New, []byte(abacatepay.WebhookPublicKey))
	mac.Write(body)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if abacatepay.Sign([]byte(abacatepay.WebhookPublicKey), body) != sig {
		t.Fatal("Sign must be the base64 HMAC-SHA256 of the raw body")
	}
}

func TestFake_IsDeterministicAndPayable(t *testing.T) {
	t.Parallel()

	f := abacatepay.NewFake()
	in := abacatepay.PixRequest{AmountCents: 1000, ExpiresIn: 600, ExternalID: "7"}

	a, err := f.CreatePix(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}

	b, _ := f.CreatePix(t.Context(), in)
	if a != b || a.Status != abacatepay.StatusPending || a.AmountCents != 1000 ||
		!strings.HasPrefix(a.BrCodeBase64, "data:image/png;base64,") || a.BrCode == "" {
		t.Fatalf("charges = %+v / %+v", a, b)
	}

	if _, err := f.SimulatePayment(t.Context(), "pix_char_unknown"); err == nil {
		t.Fatal("unknown charge must fail")
	}

	paid, err := f.SimulatePayment(t.Context(), a.ID)
	if err != nil || paid.Status != abacatepay.StatusPaid {
		t.Fatalf("simulate = %+v, %v", paid, err)
	}

	st, err := f.Check(t.Context(), a.ID)
	if err != nil || st.Status != abacatepay.StatusPaid {
		t.Fatalf("check = %+v, %v", st, err)
	}
}

func TestParseEvent_NullNestedFields(t *testing.T) {
	t.Parallel()

	body := strings.Replace(completed, `"customer":{"id":"cust_aB3dE5fG7hJ9kL1mN3pQ5rS7","name":"Maria Santos",
"email":"maria@exemplo.com","taxId":"529.***.247-**"}`, `"customer":null`, 1)
	if !strings.Contains(body, `"customer":null`) {
		t.Fatal("fixture not replaced")
	}

	if _, err := abacatepay.ParseEvent([]byte(body)); err != nil {
		t.Fatal(err)
	}
}
