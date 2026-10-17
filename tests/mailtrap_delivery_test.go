//go:build mailtrap

package tests

import (
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/tests/mailtrap"
)

// Real SMTP delivery to the Mailtrap Email Sandbox. Run with
// go test -tags=mailtrap ./tests/ and SMTP_* + MAILTRAP_* set (see .env.example).
func TestMailtrap_VerificationEmailDelivered(t *testing.T) {
	mt := mailtrap.FromEnv(t)

	env := map[string]string{"MAIL_MAILER": "smtp", "APP_URL": "http://localhost:3000"}
	for _, k := range []string{"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "MAIL_FROM"} {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: env})
	c := app.Client(t).Inertia(app.Inertia.Version())

	email := "owner-" + strconv.FormatInt(time.Now().UnixNano(), 36) + "@example.com"
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
