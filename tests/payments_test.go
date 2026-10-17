package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

const webhookSecret = "whsec_test_secret"

// payEnv runs the app on the fake gateway with a webhook secret.
func payEnv(extra ...map[string]string) vukemtest.Options {
	env := map[string]string{"ABACATEPAY_FAKE": "true", "ABACATEPAY_WEBHOOK_SECRET": webhookSecret, "APP_LOCALE": "pt-BR"}
	for _, e := range extra {
		maps.Copy(env, e)
	}

	return vukemtest.Options{Env: env}
}

// mergeEnv adds payEnv's variables to opts (e.g. the Postgres options).
func mergeEnv(opts vukemtest.Options, env map[string]string) vukemtest.Options {
	o := opts
	o.Env = maps.Clone(opts.Env)

	if o.Env == nil {
		o.Env = map[string]string{}
	}

	maps.Copy(o.Env, env)

	return o
}

func (f reservationFixture) reserve(t *testing.T, number int) reservations.Link {
	t.Helper()

	link, err := f.svc.Reserve(t.Context(), f.raffle, number, buyer)
	if err != nil {
		t.Fatal(err)
	}

	return link
}

func (f reservationFixture) reservation(t *testing.T, id int64) models.Reservation {
	t.Helper()

	res, err := orm.QueryOf[models.Reservation](f.svc.DB).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	return res
}

// confirm confirms the link as the buyer's browser would (Inertia POST).
func (f reservationFixture) confirm(t *testing.T, token string) *vukemtest.Response {
	t.Helper()

	return f.app.Client(t).Inertia("").PostJSON("/pay/"+token+"/confirm", map[string]any{})
}

// confirmed confirms the link and returns the reservation with its charge.
func (f reservationFixture) confirmed(t *testing.T, number int) (reservations.Link, models.Reservation) {
	t.Helper()

	link := f.reserve(t, number)
	f.confirm(t, link.Token).AssertRedirect("/pay/" + link.Token)

	res := f.reservation(t, link.Reservation.ID)
	if res.ChargeID == nil {
		t.Fatalf("reservation %d has no charge", res.ID)
	}

	return link, res
}

func paidEvent(eventID, chargeID string, externalID int64, amount int) []byte {
	return fmt.Appendf(nil, `{"id":%q,"event":"transparent.completed","apiVersion":2,"devMode":true,"data":{"transparent":`+
		`{"id":%q,"externalId":"%d","amount":%d,"paidAmount":null,"platformFee":80,"status":"PAID","metadata":{}},`+
		`"customer":null,"payerInformation":{"method":"PIX"}}}`, eventID, chargeID, externalID, amount)
}

func sign(body []byte) string { return abacatepay.Sign([]byte(abacatepay.WebhookPublicKey), body) }

// deliver posts a webhook like AbacatePay does (server to server, no
// Origin header).
func deliver(t *testing.T, app *vukemtest.App, secret, signature string, body []byte) *vukemtest.Response {
	t.Helper()

	c := app.Client(t)
	if signature != "" {
		c = c.WithHeader(abacatepay.SignatureHeader, signature)
	}

	return c.Do(http.MethodPost, "/webhooks/abacatepay?webhookSecret="+secret, strings.NewReader(string(body)), "application/json")
}

func deliverSigned(t *testing.T, app *vukemtest.App, body []byte) *vukemtest.Response {
	t.Helper()

	return deliver(t, app, webhookSecret, sign(body), body)
}

func ledgerSum(t *testing.T, f reservationFixture, kind string) (count int, sum int64) {
	t.Helper()

	p := f.svc.DB.Dialect().Placeholder
	if err := f.svc.DB.QueryRowContext(t.Context(), "SELECT COUNT(*), COALESCE(SUM(amount_cents), 0) FROM ledger_entries WHERE kind = "+p(1)+
		" AND raffle_id = "+p(2), kind, f.raffle.ID).Scan(&count, &sum); err != nil {
		t.Fatal(err)
	}

	return count, sum
}

func TestPay_ShowsOnlyTheBuyersOwnMaskedData(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	link := f.reserve(t, 7)

	if _, err := f.svc.Reserve(t.Context(), f.raffle, 8, buyerTwo); err != nil {
		t.Fatal(err)
	}

	res := f.app.Client(t).Inertia("").Get("/pay/"+link.Token).
		AssertOK().
		AssertHeader("Cache-Control", "no-store").
		AssertHeader("Referrer-Policy", "no-referrer").
		AssertDontSee("maria@example.com").
		AssertDontSee("987654321").
		AssertDontSee("joao@example.com").
		AssertDontSee("Joao")

	res.AssertInertia("Pay/Show").
		PropEquals("state", "live").
		PropEquals("checkout.number", 7).
		PropEquals("checkout.amount_cents", 1000).
		PropEquals("checkout.raffle_title", "Rifa da moto").
		PropEquals("checkout.buyer_name", "Maria Silva").
		PropEquals("checkout.buyer_email", "m•••@example.com").
		PropEquals("checkout.buyer_phone", "+55 11 •••••-4321").
		PropEquals("checkout.seconds_left", 600).
		PropEquals("pix", nil)
}

var buyerTwo = requests.Buyer{Name: "Joao Souza", Email: "joao@example.com", Phone: "+5521912345678"}

func TestPay_ExpiredCancelledAndUnknownLinks(t *testing.T) {
	t.Parallel()

	const expired = "Link expirado. Peça ao dono da rifa para gerar um novo link e te enviar novamente para pagar"

	f := newReservationFixture(t, payEnv())
	old := f.reserve(t, 1)
	cancelled := f.reserve(t, 2)

	if err := f.svc.Cancel(t.Context(), f.raffle.ID, 2); err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(10 * time.Minute)

	for _, tc := range []struct {
		token  string
		status int
	}{
		{old.Token, http.StatusGone},
		{cancelled.Token, http.StatusGone},
		{"unknown-token", http.StatusNotFound},
	} {
		c := f.app.Client(t)
		c.Get("/pay/"+tc.token).AssertStatus(tc.status).AssertSee(expired).AssertDontSee("maria@example.com").
			AssertHeader("Cache-Control", "no-store")
		c.Inertia("").Get("/pay/"+tc.token).AssertStatus(tc.status).AssertInertia("Pay/Show").
			PropEquals("state", "expired").PropEquals("checkout", nil)
		f.confirm(t, tc.token).AssertStatus(tc.status)
	}

	f.app.Client(t).GetJSON("/pay/" + old.Token + "/status").AssertOK().AssertJSON(map[string]string{"status": "expired"})
	f.app.Client(t).GetJSON("/pay/unknown-token/status").AssertNotFound()

	// English buyers get the English text.
	f.app.Client(t).WithHeader("Accept-Language", "en").Get("/pay/" + old.Token).AssertStatus(http.StatusGone).
		AssertSee("Link expired. Ask the raffle owner to generate a new link and send it to you again to pay")
}

func TestPay_ConfirmCreatesOnePixAndShowsIt(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	link, res := f.confirmed(t, 5)

	if res.Status != models.ReservationAwaitingPayment || res.BrCode == nil || res.BrCodeBase64 == nil ||
		!strings.HasPrefix(*res.BrCodeBase64, "data:image/png;base64,") || res.Locale != "pt-BR" {
		t.Fatalf("reservation = %+v", res)
	}

	// Confirming again (double click, reload) reuses the charge.
	f.confirm(t, link.Token).AssertRedirect("/pay/" + link.Token)

	if again := f.reservation(t, res.ID); *again.ChargeID != *res.ChargeID {
		t.Fatalf("charge changed: %s → %s", *res.ChargeID, *again.ChargeID)
	}

	f.app.Client(t).Inertia("").Get("/pay/"+link.Token).AssertOK().AssertInertia("Pay/Show").
		PropEquals("state", "live").
		PropEquals("checkout.status", "awaiting_payment").
		PropEquals("pix.br_code", *res.BrCode).
		PropEquals("pix.br_code_base64", *res.BrCodeBase64)

	f.app.Client(t).GetJSON("/pay/"+link.Token+"/status").AssertOK().
		AssertHeader("Cache-Control", "no-store").AssertJSON(map[string]string{"status": "awaiting_payment"})
}

// abacatePayServer is a fake AbacatePay API recording the PIX requests.
func abacatePayServer(t *testing.T, status int) (*httptest.Server, func() []map[string]any) {
	t.Helper()

	var (
		mu   sync.Mutex
		reqs []map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		mu.Lock()
		reqs = append(reqs, body)
		mu.Unlock()

		w.WriteHeader(status)

		if status != http.StatusOK {
			_, _ = io.WriteString(w, `{"data":null,"success":false,"error":"boom"}`)
			return
		}

		data, _ := body["data"].(map[string]any)
		_, _ = fmt.Fprintf(w, `{"data":{"id":"pix_char_%v","amount":%v,"status":"PENDING","devMode":true,"brCode":"000201pix",`+
			`"brCodeBase64":"data:image/png;base64,iVBORw0KGgo="},"success":true,"error":null}`, data["externalId"], data["amount"])
	}))
	t.Cleanup(srv.Close)

	return srv, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()

		return append([]map[string]any(nil), reqs...)
	}
}

func TestPay_ConfirmChargesTheRemainingTime(t *testing.T) {
	t.Parallel()

	srv, requests := abacatePayServer(t, http.StatusOK)
	f := newReservationFixture(t, vukemtest.Options{Env: map[string]string{
		"ABACATEPAY_API_KEY": "abc_dev_test", "ABACATEPAY_BASE_URL": srv.URL,
	}})

	link := f.reserve(t, 3)
	f.clock.Advance(4 * time.Minute)
	f.confirm(t, link.Token).AssertRedirect("/pay/" + link.Token)

	got := requests()
	if len(got) != 1 {
		t.Fatalf("requests = %d", len(got))
	}

	data, _ := got[0]["data"].(map[string]any)
	if got[0]["method"] != "PIX" || data["amount"] != float64(1000) || data["expiresIn"] != float64(360) ||
		data["externalId"] != fmt.Sprint(link.Reservation.ID) {
		t.Fatalf("pix request = %v", got[0])
	}

	if res := f.reservation(t, link.Reservation.ID); res.ChargeID == nil || *res.ChargeID != fmt.Sprintf("pix_char_%d", res.ID) {
		t.Fatalf("charge = %v", res.ChargeID)
	}
}

func TestPay_ConfirmWithoutGatewayIsAFriendly503(t *testing.T) {
	t.Parallel()

	failing, _ := abacatePayServer(t, http.StatusInternalServerError)

	for name, env := range map[string]map[string]string{
		"not configured": {},
		"gateway down":   {"ABACATEPAY_API_KEY": "abc_dev_test", "ABACATEPAY_BASE_URL": failing.URL},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newReservationFixture(t, vukemtest.Options{Env: env})
			link := f.reserve(t, 4)

			f.confirm(t, link.Token).AssertStatus(http.StatusServiceUnavailable).AssertInertia("Pay/Show").
				PropEquals("state", "live").HasProp("error")
			f.app.Client(t).AcceptJSON().PostJSON("/pay/"+link.Token+"/confirm", map[string]any{}).
				AssertProblem(http.StatusServiceUnavailable)

			if res := f.reservation(t, link.Reservation.ID); res.Status != models.ReservationPending || res.ChargeID != nil {
				t.Fatalf("reservation = %+v", res)
			}

			// The page itself keeps working.
			f.app.Client(t).Get("/pay/" + link.Token).AssertOK()
		})
	}
}

func TestWebhook_PaymentMarksTheNumberPaid(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	link, res := f.confirmed(t, 9)

	body := paidEvent("log_paid_1", *res.ChargeID, res.ID, 1000)
	deliverSigned(t, f.app, body).AssertOK().AssertJSON(map[string]string{"outcome": "paid"})

	paid := f.reservation(t, res.ID)
	if paid.Status != models.ReservationPaid || !paid.PaidAt.Valid || f.numberStatus(t, 9) != models.NumberPaid {
		t.Fatalf("reservation = %+v, number = %s", paid, f.numberStatus(t, 9))
	}

	raffle, err := orm.QueryOf[models.Raffle](f.svc.DB).Find(t.Context(), f.raffle.ID)
	if err != nil || !raffle.Locked() {
		t.Fatalf("raffle locked_at = %v, %v", raffle.LockedAt, err)
	}

	if n, sum := ledgerSum(t, f, models.LedgerSaleOwnerShare); n != 1 || sum != 900 {
		t.Fatalf("owner share = %d rows, %d", n, sum)
	}

	if n, sum := ledgerSum(t, f, models.LedgerPlatformFee); n != 1 || sum != 100 {
		t.Fatalf("platform fee = %d rows, %d", n, sum)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "payment_events", map[string]any{
		"event_id": "log_paid_1", "event_type": "transparent.completed", "reservation_id": res.ID, "provider": "abacatepay",
	})
	vukemtest.AssertDatabaseMissing(t, f.app, "payment_events", map[string]any{"event_id": "log_paid_1", "processed_at": nil})

	f.app.Client(t).GetJSON("/pay/" + link.Token + "/status").AssertJSON(map[string]string{"status": "paid"})
	f.app.Client(t).Inertia("").Get("/pay/"+link.Token).AssertOK().AssertInertia("Pay/Show").
		PropEquals("state", "paid").PropEquals("pix", nil)
}

func TestWebhook_PaymentEmailsBuyerAndOwner(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		lang, buyerSubject, buyerText string
	}{
		{"pt-BR", "Pagamento confirmado: número 9", "Recebemos o seu PIX de R$ 10,00"},
		{"en", "Payment confirmed: number 9", "We received your PIX of R$10.00"},
	} {
		t.Run(tc.lang, func(t *testing.T) {
			t.Parallel()

			f := newReservationFixture(t, payEnv())
			link := f.reserve(t, 9)
			f.app.Client(t).Inertia("").WithHeader("Accept-Language", tc.lang).
				PostJSON("/pay/"+link.Token+"/confirm", map[string]any{}).AssertRedirect("/pay/" + link.Token)

			res := f.reservation(t, link.Reservation.ID)
			deliverSigned(t, f.app, paidEvent("log_mail_"+tc.lang, *res.ChargeID, res.ID, 1000)).AssertOK()

			sent := f.app.Mails().SentTo("maria@example.com")
			if len(sent) != 1 || sent[0].Subject != tc.buyerSubject || !strings.Contains(sent[0].HTML, tc.buyerText) ||
				!strings.Contains(sent[0].HTML, "/r/"+f.raffle.Slug) {
				t.Fatalf("buyer mails = %+v", sent)
			}

			// Owners have no stored locale: APP_LOCALE (pt-BR).
			owned := f.app.Mails().SentTo("dono@example.com")

			var sold []string

			for _, m := range owned {
				if strings.Contains(m.Subject, "vendido") {
					sold = append(sold, m.HTML)
				}
			}

			if len(sold) != 1 || !strings.Contains(sold[0], "Maria Silva pagou R$ 10,00 pelo número 9") ||
				!strings.Contains(sold[0], fmt.Sprintf("/raffles/%d", f.raffle.ID)) {
				t.Fatalf("owner mails = %v", sold)
			}
		})
	}
}

func TestWebhook_LatePaymentIsFlaggedForRefund(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 11)

	// Past expires_at + PaymentGrace; the sweeper has not run yet.
	f.clock.Advance(models.ReservationTTL + reservations.PaymentGrace)
	deliverSigned(t, f.app, paidEvent("log_late_1", *res.ChargeID, res.ID, 1000)).AssertOK().
		AssertJSON(map[string]string{"outcome": "refund_due"})

	if got := f.numberStatus(t, 11); got == models.NumberPaid {
		t.Fatal("a late payment must never assign the number")
	}

	if n, sum := ledgerSum(t, f, models.LedgerRefundDue); n != 1 || sum != 1000 {
		t.Fatalf("refund_due = %d rows, %d", n, sum)
	}

	if n, _ := ledgerSum(t, f, models.LedgerSaleOwnerShare); n != 0 {
		t.Fatal("late payment must not credit the owner")
	}
}

func TestWebhook_PaymentAfterTheNumberWasReReserved(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 12)

	f.clock.Advance(models.ReservationTTL + reservations.PaymentGrace)

	if _, err := f.svc.ExpireDue(t.Context()); err != nil {
		t.Fatal(err)
	}

	f.reserve(t, 12) // another buyer holds the number now

	deliverSigned(t, f.app, paidEvent("log_late_2", *res.ChargeID, res.ID, 1000)).AssertOK().
		AssertJSON(map[string]string{"outcome": "refund_due"})

	if got := f.numberStatus(t, 12); got != models.NumberReserved {
		t.Fatalf("number = %s, want still reserved by the new buyer", got)
	}

	if n, _ := ledgerSum(t, f, models.LedgerRefundDue); n != 1 {
		t.Fatalf("refund_due rows = %d", n)
	}
}

func TestWebhook_AmountMismatchIsNotAPayment(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 13)

	deliverSigned(t, f.app, paidEvent("log_amount_1", *res.ChargeID, res.ID, 100)).AssertOK().
		AssertJSON(map[string]string{"outcome": "refund_due"})

	if f.numberStatus(t, 13) == models.NumberPaid || f.reservation(t, res.ID).Status == models.ReservationPaid {
		t.Fatal("a wrong amount must not pay the number")
	}

	if n, sum := ledgerSum(t, f, models.LedgerRefundDue); n != 1 || sum != 100 {
		t.Fatalf("refund_due = %d rows, %d", n, sum)
	}
}

func TestWebhook_UnknownEventsAndChargesAreRecorded(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())

	deliverSigned(t, f.app, []byte(`{"id":"log_payout_1","event":"payout.completed","apiVersion":2,"devMode":true,"data":{}}`)).
		AssertOK().AssertJSON(map[string]string{"outcome": "ignored"})
	deliverSigned(t, f.app, paidEvent("log_unknown_1", "pix_char_nobody", 999999, 1000)).
		AssertOK().AssertJSON(map[string]string{"outcome": "unknown_charge"})

	vukemtest.AssertDatabaseCount(t, f.app, "payment_events", 2)
	vukemtest.AssertDatabaseCount(t, f.app, "ledger_entries", 0)
}

func TestWebhook_SecondEventForAPaidChargeIsANoOp(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t, payEnv())
	_, res := f.confirmed(t, 14)

	deliverSigned(t, f.app, paidEvent("log_a", *res.ChargeID, res.ID, 1000)).AssertOK()
	// A manual resend from the dashboard carries a new event id.
	deliverSigned(t, f.app, paidEvent("log_b", *res.ChargeID, res.ID, 1000)).AssertOK().
		AssertJSON(map[string]string{"outcome": "already_paid"})

	if n, _ := ledgerSum(t, f, models.LedgerSaleOwnerShare); n != 1 {
		t.Fatalf("owner share rows = %d, want 1", n)
	}
}
