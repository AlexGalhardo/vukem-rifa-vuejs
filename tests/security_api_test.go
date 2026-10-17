package tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// OWASP abuse suite for the REST API (A01 access control, A07 auth, A04
// insecure design: replays, A05 injection: CSV).

func TestSecurityAPI_RejectsMissingInvalidAndRevokedTokens(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.tokens@example.com")
	id, _, _ := strings.Cut(f.token, ".")

	tests := map[string]*vukemtest.Client{
		"missing":        f.app.Client(t),
		"garbage":        f.app.Client(t).WithToken("garbage"),
		"wrong secret":   f.app.Client(t).WithToken(id + ".AAAAAAAAAAAAAAAAAAAAAAAAAA"),
		"basic scheme":   f.app.Client(t).WithHeader("Authorization", "Basic "+f.token),
		"token in query": f.app.Client(t),
	}
	for name, c := range tests {
		res := c.Get("/api/v1/raffles?token=" + f.token).AssertUnauthorized()
		res.AssertHeaderContains("Content-Type", "application/problem+json").
			AssertHeaderContains("WWW-Authenticate", "Bearer")

		c.PostJSON("/api/v1/raffles", validRaffle()).AssertUnauthorized()

		if t.Failed() {
			t.Fatalf("case %q", name)
		}
	}

	f.web.Delete("/settings/api-keys/" + id).AssertRedirect("/settings/api-keys")
	f.api.Get("/api/v1/raffles").AssertUnauthorized().AssertMatchesOpenAPI()
	vukemtest.AssertDatabaseCount(t, f.app, "raffles", 0)
}

func TestSecurityAPI_ReadOnlyKeyCannotWrite(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.abilities@example.com")
	id := f.createAPIRaffle(t, validRaffle())
	base := fmt.Sprintf("/api/v1/raffles/%d", id)
	buyer := map[string]any{"buyer_name": "Maria Silva", "buyer_email": "maria@example.com", "buyer_phone": "11987654321"}

	f.readOnly.Get(base).AssertOK()

	for _, res := range []*vukemtest.Response{
		f.readOnly.PostJSON("/api/v1/raffles", validRaffle()),
		f.readOnly.PatchJSON(base, map[string]any{"title": "Hackeada"}),
		f.readOnly.Delete(base),
		f.readOnly.PostJSON(base+"/archive", nil),
		f.readOnly.PostJSON(base+"/numbers/1/payment-links", buyer),
	} {
		res.AssertForbidden().AssertHeaderContains("Content-Type", "application/problem+json").AssertMatchesOpenAPI()
	}

	f.api.Get(base).AssertJSONPath("title", "Rifa da moto").AssertJSONPath("status", "active").AssertJSONPath("reserved", 0)
	vukemtest.AssertDatabaseCount(t, f.app, "raffles", 1)
}

func TestSecurityAPI_OtherOwnersRafflesAreNotFound(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.idor@example.com")
	id := f.createAPIRaffle(t, validRaffle())
	base := fmt.Sprintf("/api/v1/raffles/%d", id)

	// The intruder signs up on the same app with a full-access key.
	intruderWeb := owner(t, f.app, "sec.api.intruder@example.com")
	intruder := f.app.Client(t).WithToken(newAPIKey(t, intruderWeb, "x", "raffles:read", "raffles:write"))
	buyer := map[string]any{"buyer_name": "Maria Silva", "buyer_email": "maria@example.com", "buyer_phone": "11987654321"}

	for name, res := range map[string]*vukemtest.Response{
		"show":         intruder.Get(base),
		"update":       intruder.PatchJSON(base, map[string]any{"title": "Hackeada"}),
		"archive":      intruder.PostJSON(base+"/archive", nil),
		"numbers":      intruder.Get(base + "/numbers"),
		"payment link": intruder.PostJSON(base+"/numbers/1/payment-links", buyer),
		"export json":  intruder.Get(base + "/export"),
		"export csv":   intruder.Get(base + "/export?format=csv"),
		"delete":       intruder.Delete(base),
	} {
		res.AssertNotFound().AssertDontSee("Rifa da moto")

		if t.Failed() {
			t.Fatalf("%s leaked", name)
		}
	}

	intruder.Get("/api/v1/raffles").AssertJSONCount("data", 0)
	f.api.Get(base).AssertOK().AssertJSONPath("title", "Rifa da moto").AssertJSONPath("status", "active").
		AssertJSONPath("reserved", 0)
}

func TestSecurityAPI_IdempotentPaymentLinks(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.idem@example.com")
	id := f.createAPIRaffle(t, validRaffle())
	base := fmt.Sprintf("/api/v1/raffles/%d/numbers", id)
	buyer := map[string]any{"buyer_name": "Maria Silva", "buyer_email": "maria@example.com", "buyer_phone": "11987654321"}

	f.api.WithHeader("Idempotency-Key", "order-42")

	first := f.api.PostJSON(base+"/7/payment-links", buyer).AssertCreated()
	replay := f.api.PostJSON(base+"/7/payment-links", buyer).AssertCreated().AssertHeader("Idempotent-Replayed", "true")

	if first.Body() != replay.Body() {
		t.Fatalf("replay differs:\n%s\n%s", first.Body(), replay.Body())
	}

	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 1)

	// The same key for a different request is refused, not replayed.
	f.api.PostJSON(base+"/8/payment-links", buyer).AssertUnprocessable().AssertHeaderMissing("Idempotent-Replayed")
	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 1)

	// Another API key (same owner) using the same Idempotency-Key never sees
	// the first key's response: its request runs on its own.
	other := f.app.Client(t).WithToken(newAPIKey(t, f.web, "other", "raffles:read", "raffles:write")).
		WithHeader("Idempotency-Key", "order-42")
	res := other.PostJSON(base+"/8/payment-links", buyer).AssertCreated().AssertHeaderMissing("Idempotent-Replayed")
	res.AssertJSONPath("number", 8)

	if res.Body() == first.Body() {
		t.Fatal("second key got the first key's payment link")
	}

	other.PostJSON(base+"/7/payment-links", buyer).AssertUnprocessable()
	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 2)
}

func TestSecurityAPI_CSVExportGuardsFormulas(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.csv@example.com")
	fields := validRaffle()
	fields["title"] = `=HYPERLINK("https://evil.example")`
	fields["description"] = "+cmd|' /C calc'!A0"
	id := f.createAPIRaffle(t, fields)

	f.api.PostJSON(fmt.Sprintf("/api/v1/raffles/%d/numbers/3/payment-links", id),
		map[string]any{"buyer_name": "@SUM(1+1)", "buyer_email": "maria@example.com", "buyer_phone": "11987654321"}).
		AssertCreated()

	body := f.api.Get(fmt.Sprintf("/api/v1/raffles/%d/export?format=csv", id)).AssertOK().Body()
	for _, want := range []string{`"'=HYPERLINK(""https://evil.example"")"`, `'+cmd|' /C calc'!A0`, `'@SUM(1+1)`} {
		if !strings.Contains(body, want) {
			t.Errorf("CSV lacks guarded cell %s:\n%s", want, body)
		}
	}
}

func TestSecurityAPI_CookieSessionIsNotAmbientAuthority(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.cookie@example.com")

	// The owner's signed-in browser session cannot call the API.
	f.web.Get("/api/v1/raffles").AssertUnauthorized().AssertHeaderContains("Content-Type", "application/problem+json")
	f.web.PostJSON("/api/v1/raffles", validRaffle()).AssertUnauthorized()
	vukemtest.AssertDatabaseCount(t, f.app, "raffles", 0)

	// API responses never start a session.
	f.api.Get("/api/v1/raffles").AssertOK().AssertHeaderMissing("Set-Cookie")
}

func TestSecurityAPI_NoCORSAndCrossSiteBrowsersBlocked(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.cors@example.com")

	preflight := f.app.Client(t).WithHeader("Origin", "https://evil.example").
		WithHeader("Access-Control-Request-Method", "POST").
		WithHeader("Access-Control-Request-Headers", "authorization")
	preflight.Do(http.MethodOptions, "/api/v1/raffles", nil, "").AssertHeaderMissing("Access-Control-Allow-Origin")

	f.api.WithHeader("Origin", "https://evil.example")
	f.api.Get("/api/v1/raffles").AssertHeaderMissing("Access-Control-Allow-Origin")

	// A cross-site browser write is refused even with a token; server-to-
	// server clients (no Origin, no Sec-Fetch-Site) are not affected.
	f.api.WithHeader("Sec-Fetch-Site", "cross-site").PostJSON("/api/v1/raffles", validRaffle()).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, f.app, "raffles", 0)
}

func TestSecurityAPI_RateLimitedPerKey(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "sec.api.ratelimit@example.com")

	// 120 per minute; the bucket refills while the loop runs (slowly under
	// -race), so allow some slack before the first 429.
	var limited *vukemtest.Response

	for i := range 200 {
		res := f.api.Get("/api/v1/me/balance")
		if res.Status() == http.StatusTooManyRequests {
			if i < 120 {
				t.Fatalf("limited after %d requests", i)
			}

			limited = res

			break
		}

		res.AssertOK()
	}

	if limited == nil {
		t.Fatal("200 requests in a row were never limited")
	}

	limited.AssertHeaderContains("Content-Type", "application/problem+json")

	if limited.Header("Retry-After") == "" {
		t.Fatal("429 without Retry-After")
	}

	// Another key of the same owner has its own bucket.
	f.readOnly.Get("/api/v1/me/balance").AssertOK()
}
