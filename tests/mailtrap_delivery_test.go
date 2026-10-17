//go:build mailtrap

package tests

import (
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/tests/mailtrap"
)

// Real SMTP delivery to the Mailtrap Email Sandbox. Run with
// go test -tags=mailtrap -p 1 ./tests/ and SMTP_* + MAILTRAP_* set (see
// .env.example). The free sandbox is shared with other projects and rate
// limited: tests use a unique recipient, filter by it, run serially and
// never clean the inbox.
func TestMailtrap_VerificationEmailDelivered(t *testing.T) {
	mt := mailtrap.FromEnv(t)

	env := map[string]string{"MAIL_MAILER": "smtp", "APP_URL": "http://localhost:3000"}
	for _, k := range []string{"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "MAIL_FROM"} {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: env})
	c := app.Client(t).Inertia("")

	email := "rifa-ci-" + strings.ToLower(rand.Text()[:12]) + "@example.com"
	register(t, c, email)

	link, err := mt.WaitLink(t.Context(), email, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}

	c.Get(u.RequestURI()).AssertRedirect("/dashboard")
}
