package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// OWASP A01/A07 abuse cases for the auth surface.

func TestSecurity_CrossSiteLoginBlocked(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	register(t, app.Client(t).Inertia(app.Inertia.Version()), "csrf@example.com")

	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"foreign origin", "Origin", "https://evil.example"},
		{"cross-site fetch metadata", "Sec-Fetch-Site", "cross-site"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app.Client(t).WithHeader(tt.header, tt.value).
				PostJSON("/login", map[string]any{"email": "csrf@example.com", "password": "correct horse battery"}).
				AssertForbidden()
		})
	}
}

func TestSecurity_ForgotPasswordRateLimited(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := app.Client(t).Inertia(app.Inertia.Version())

	// 3 per minute per IP (routes/auth.go).
	for range 3 {
		c.PostJSON("/forgot-password", map[string]any{"email": "x@example.com"}).AssertRedirect("/forgot-password")
	}

	c.PostJSON("/forgot-password", map[string]any{"email": "x@example.com"}).AssertStatus(429)
}

func TestSecurity_SessionRotatesOnLogin(t *testing.T) {
	t.Parallel()

	// Server-side store: the cookie is the session id (encrypted cookies change on every write).
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"SESSION_DRIVER": "memory"}})
	register(t, app.Client(t).Inertia(app.Inertia.Version()), "fix@example.com")

	c := app.Client(t).Inertia(app.Inertia.Version())
	c.Get("/login").AssertOK()

	before := c.CookieValue("vukem_session")
	if before == "" {
		t.Fatal("no session cookie before login")
	}

	c.PostJSON("/login", map[string]any{"email": "fix@example.com", "password": "correct horse battery"}).
		AssertRedirect("/dashboard")

	if after := c.CookieValue("vukem_session"); after == before {
		t.Fatal("session cookie unchanged after login (session fixation)")
	}
}
