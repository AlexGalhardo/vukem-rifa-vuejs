package tests

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// OWASP Top 10 (2021) abuse cases not covered elsewhere. docs/owasp-matrix.md
// maps every category to these and to the existing security tests.

// A01: fields the owner must not control are ignored on create and update
// (mass assignment), so nobody can create a raffle for another user, skip
// the lock, or pick the status or slug.
func TestOWASP_A01_MassAssignmentIsIgnored(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	victim := owner(t, app, "victim-ma@example.com")
	victimRaffle := createRaffle(t, app, victim, validRaffle())
	attacker := owner(t, app, "attacker-ma@example.com")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	victimRow, err := orm.QueryOf[models.Raffle](db).Find(t.Context(), victimRaffle)
	if err != nil {
		t.Fatal(err)
	}

	fields := validRaffle()
	for k, v := range map[string]any{
		"id": victimRaffle, "user_id": victimRow.UserID, "status": models.RaffleDrawn, "slug": "chosen-slug",
		"locked_at": "2026-01-01T00:00:00Z", "starts_at": "2020-01-01T00:00:00Z", "is_admin": true,
	} {
		fields[k] = v
	}

	id := createRaffle(t, app, attacker, fields)
	if id == victimRaffle {
		t.Fatal("create reused the victim's id")
	}

	r, err := orm.QueryOf[models.Raffle](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	if r.UserID == victimRow.UserID || r.Status != models.RaffleActive || r.Slug == "chosen-slug" || r.Locked() || r.StartsAt.Year() == 2020 {
		t.Fatalf("mass-assigned fields were stored: %+v", r)
	}

	attacker.PutJSON(fmt.Sprintf("/raffles/%d", id), fields).AssertRedirect(fmt.Sprintf("/raffles/%d", id))

	r, err = orm.QueryOf[models.Raffle](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	if r.UserID == victimRow.UserID || r.Status != models.RaffleActive || r.Slug == "chosen-slug" || r.Locked() {
		t.Fatalf("update mass-assigned fields: %+v", r)
	}

	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "attacker-ma@example.com", "is_admin": false})
}

// A02/A05: the session cookie is HttpOnly and SameSite, so scripts cannot
// read it and other sites cannot ride it.
func TestOWASP_A02_SessionCookieFlags(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	register(t, app.Client(t).Inertia(""), "cookie@example.com")

	// Sessions are lazy: the cookie is issued when the login writes one.
	res := app.Client(t).Inertia("").PostJSON("/login", map[string]any{"email": "cookie@example.com", "password": "correct horse battery"}).
		AssertRedirect("/dashboard")

	var session string

	for _, c := range res.Recorder().Header().Values("Set-Cookie") {
		if strings.HasPrefix(c, "vukem_session=") {
			session = c
		}
	}

	if session == "" {
		t.Fatalf("no session cookie after login: %v", res.Recorder().Header().Values("Set-Cookie"))
	}

	for _, want := range []string{"HttpOnly", "SameSite=Lax", "Path=/"} {
		if !strings.Contains(session, want) {
			t.Errorf("session cookie %q lacks %s", session, want)
		}
	}
}

// A03: injection payloads in path parameters and credentials are plain
// data (parameterized SQL, typed route params).
func TestOWASP_A03_InjectionPayloadsAreInert(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "inj@example.com")
	createRaffle(t, app, c, validRaffle())

	for _, path := range []string{
		"/r/x'%20OR%20'1'='1", "/r/x%22%3B%20DROP%20TABLE%20raffles%3B--", "/pay/'%20OR%201=1--",
	} {
		app.Client(t).Get(path).AssertNotFound()
	}

	for _, path := range []string{"/raffles/1%20OR%201=1", "/raffles/1;DROP%20TABLE%20raffles"} {
		if got := c.Get(path).Status(); got != http.StatusNotFound && got != http.StatusBadRequest {
			t.Errorf("%s = %d, want 404 or 400", path, got)
		}
	}

	payload := "x' OR '1'='1"
	app.Client(t).Inertia("").WithHeader("Referer", "/login").
		PostJSON("/login", map[string]any{"email": "inj@example.com' OR '1'='1", "password": payload}).
		AssertRedirect("/login")

	vukemtest.AssertDatabaseCount(t, app, "raffles", 1)
}

// A05: security headers on HTML pages, the buyer page and the API.
func TestOWASP_A05_SecurityHeaders(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)

	for _, path := range []string{"/", "/login", "/pay/unknown-token"} {
		res := app.Client(t).Get(path)
		res.AssertHeader("X-Frame-Options", "DENY").
			AssertHeader("X-Content-Type-Options", "nosniff").
			AssertHeaderContains("Content-Security-Policy", "frame-ancestors 'none'").
			AssertHeaderContains("Content-Security-Policy", "object-src 'none'").
			AssertHeaderContains("Content-Security-Policy", "'nonce-").
			AssertHeaderMissing("X-Powered-By")

		if res.Header("Referrer-Policy") == "" || res.Header("Permissions-Policy") == "" {
			t.Errorf("%s: missing Referrer-Policy or Permissions-Policy", path)
		}

		if strings.Contains(res.Header("Content-Security-Policy"), "unsafe-inline") {
			t.Errorf("%s: CSP allows unsafe-inline", path)
		}
	}

	app.Client(t).Get("/pay/unknown-token").AssertHeader("Cache-Control", "no-store").AssertHeader("Referrer-Policy", "no-referrer")
	app.Client(t).Get("/api/v1/raffles").AssertUnauthorized().AssertHeader("Content-Type", "application/problem+json").
		AssertHeaderMissing("Access-Control-Allow-Origin")
}

// lockedBuffer is a log sink shared by the app's goroutines.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// A09: a forged webhook is logged as a security event with the request id,
// and the log never contains the secret or the signature it carried.
func TestOWASP_A09_RejectedWebhookIsLoggedWithoutSecrets(t *testing.T) {
	t.Parallel()

	var logs lockedBuffer

	env := payEnv()
	env.AppOptions = append(env.AppOptions, vukem.WithLogger(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))))

	f := newReservationFixture(t, env)
	_, res := f.confirmed(t, 30)
	body := paidEvent("log_a09", *res.ChargeID, res.ID, 1000)

	deliver(t, f.app, "guess-the-secret", sign(body), body).AssertUnauthorized()

	out := logs.String()
	if !strings.Contains(out, "webhook") || !strings.Contains(out, "request_id") {
		t.Fatalf("no security event with a request id for the rejected webhook:\n%s", out)
	}

	for _, secret := range []string{webhookSecret, "guess-the-secret", sign(body)} {
		if strings.Contains(out, secret) {
			t.Fatalf("log leaks %q", secret)
		}
	}
}

// A10 (SSRF / key exfiltration): the only outbound HTTP is the payment
// gateway at an operator-set URL; production refuses a non-HTTPS URL so a
// typo cannot send the API key in clear text, and the fake gateway.
func TestOWASP_A10_GatewayURLIsHTTPSOnlyInProduction(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		cfg     payments.Config
		wantErr error
	}{
		"http in production":    {payments.Config{APIKey: "k", BaseURL: "http://api.abacatepay.com/v2", Production: true}, payments.ErrInsecureBaseURL},
		"internal host":         {payments.Config{APIKey: "k", BaseURL: "http://169.254.169.254/latest", Production: true}, payments.ErrInsecureBaseURL},
		"fake in production":    {payments.Config{Fake: true, Production: true}, payments.ErrFakeInProduction},
		"https in production":   {payments.Config{APIKey: "k", BaseURL: "https://api.abacatepay.com/v2", Production: true}, nil},
		"default in production": {payments.Config{APIKey: "k", Production: true}, nil},
		"http in development":   {payments.Config{APIKey: "k", BaseURL: "http://localhost:9999"}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := tc.cfg.Gateway(); err != tc.wantErr { //nolint:errorlint // sentinel identity is the contract
				t.Fatalf("Gateway() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// A02: production passwords are hashed with argon2id at vukem's parameters
// (64 MiB, t=1, p=4: above OWASP's 46 MiB/t=1/p=1 minimum), salted, never
// stored in clear; the fast fake is for the test kind only.
func TestOWASP_A02_PasswordsAreArgon2id(t *testing.T) {
	t.Parallel()

	h := accounts.HasherFor(vukem.KindWeb)

	first, err := h.Make("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}

	second, err := h.Make("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(first, "$argon2id$v=19$m=65536,t=1,p=4$") {
		t.Fatalf("hash = %q, want argon2id m=65536,t=1,p=4", first)
	}

	if first == second || strings.Contains(first, "correct horse") {
		t.Fatal("hashes must be salted and never contain the password")
	}

	if ok, err := h.Verify(first, "correct horse battery"); err != nil || !ok {
		t.Fatalf("Verify = %v, %v", ok, err)
	}

	if ok, _ := h.Verify(first, "wrong password!!"); ok {
		t.Fatal("wrong password accepted")
	}
}

// A02/A05: production refuses to boot on an http APP_URL.
func TestOWASP_A05_ProductionRefusesHTTPAppURL(t *testing.T) {
	t.Parallel()

	_, err := vukemtest.Build(t, bootstrap.New, vukemtest.Options{Env: map[string]string{
		"APP_ENV": "production", "APP_URL": "http://rifa.example.com", "APP_DEBUG": "false",
	}})
	if !errors.Is(err, bootstrap.ErrInsecureAppURL) {
		t.Fatalf("boot error = %v, want ErrInsecureAppURL", err)
	}
}

// A02/A05: over HTTPS, production answers with HSTS.
func TestOWASP_A05_ProductionSendsHSTS(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{
		"APP_ENV": "production", "APP_URL": "https://rifa.example.com", "APP_DEBUG": "false",
	}})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://rifa.example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if hsts := rec.Header().Get("Strict-Transport-Security"); !strings.Contains(hsts, "max-age=") {
		t.Fatalf("Strict-Transport-Security = %q", hsts)
	}
}

// A09: failed logins are security events with the request id; the log never
// holds the password or the email that was tried.
func TestOWASP_A09_FailedLoginIsLoggedWithoutCredentials(t *testing.T) {
	t.Parallel()

	var logs lockedBuffer

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{AppOptions: []vukem.Option{
		vukem.WithLogger(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))),
	}})
	register(t, app.Client(t).Inertia(""), "victim-log@example.com")

	guess := "hunter2-" + strings.Repeat("x", 8)
	app.Client(t).Inertia("").WithHeader("Referer", "/login").
		PostJSON("/login", map[string]any{"email": "victim-log@example.com", "password": guess}).AssertRedirect("/login")

	out := logs.String()
	if !strings.Contains(out, `"event":"auth.login_failed"`) || !strings.Contains(out, "request_id") {
		t.Fatalf("no auth.login_failed security event:\n%s", out)
	}

	for _, leak := range []string{guess, "victim-log@example.com"} {
		if strings.Contains(out, leak) {
			t.Fatalf("log leaks %q", leak)
		}
	}
}

// A09: every payout is a security event (who paid whom how much), without
// the free-text reference.
func TestOWASP_A09_PayoutIsLogged(t *testing.T) {
	t.Parallel()

	var logs lockedBuffer

	env := payEnv()
	env.AppOptions = append(env.AppOptions, vukem.WithLogger(slog.New(slog.NewJSONHandler(&logs, nil))))

	f := newReservationFixture(t, env)
	f.paidSale(t, 3)
	makeAdmin(t, f.app, "admin-log@example.com")

	loginJSON(t, f.app, "admin-log@example.com").
		PostJSON("/admin/payouts", map[string]any{"user_id": f.raffle.UserID, "amount_cents": 500, "reference": "PIX-E2E-SECRETREF"})

	out := logs.String()
	for _, want := range []string{`"event":"payout.recorded"`, `"amount_cents":500`, `"admin_id"`, "request_id"} {
		if !strings.Contains(out, want) {
			t.Fatalf("payout log lacks %s:\n%s", want, out)
		}
	}

	if strings.Contains(out, "PIX-E2E-SECRETREF") {
		t.Fatal("payout log leaks the reference")
	}
}
