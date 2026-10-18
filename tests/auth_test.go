package tests

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/queue"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/accounts"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

var linkRe = regexp.MustCompile(`https?://[^\s)"]+`)

// link extracts the first URL from the last mail sent to email.
func link(t *testing.T, app *vukemtest.App, email string) string {
	t.Helper()

	sent := app.Mails().SentTo(email)
	if len(sent) == 0 {
		deliverQueuedMail(t, app, email)
		sent = app.Mails().SentTo(email)
	}

	if len(sent) == 0 {
		t.Fatalf("no mail sent to %s", email)
	}

	u := linkRe.FindString(sent[len(sent)-1].Text)
	if u == "" {
		t.Fatalf("no link in mail: %q", sent[len(sent)-1].Text)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}

	return parsed.RequestURI()
}

// register signs up email and then logs in with it (sign-up itself never
// logs in, so new and existing emails look the same).
func register(t *testing.T, c *vukemtest.Client, email string) {
	t.Helper()

	c.PostJSON("/register", map[string]any{
		"name": "Ana Silva", "email": email,
		"password": "correct horse battery", "password_confirmation": "correct horse battery",
	}).AssertRedirect("/register/check-email")

	c.PostJSON("/login", map[string]any{"email": email, "password": "correct horse battery"}).AssertRedirect("/dashboard")
}

func TestRegisterAndVerifyEmail(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := app.Client(t).Inertia("")

	c.Get("/dashboard").AssertRedirect("/login")
	c.Get("/login").AssertInertia("Auth/Login").PropEquals("auth.user", nil)

	signup := app.Client(t).Inertia("")
	signup.PostJSON("/register", map[string]any{
		"name": "Ana Silva", "email": "ana@example.com",
		"password": "correct horse battery", "password_confirmation": "correct horse battery",
	}).AssertRedirect("/register/check-email")
	signup.Get("/register/check-email").AssertOK().AssertInertia("Auth/CheckEmail").PropEquals("auth.user", nil)
	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "ana@example.com", "email_verified_at": nil})

	verify := link(t, app, "ana@example.com")
	c.Get(verify + "x").AssertRedirect("/login") // signed out: the login comes first

	stranger := app.Client(t).Inertia("")
	register(t, stranger, "someone-else@example.com")
	stranger.Get(verify).AssertForbidden() // signed in as another account

	// Signed out, the link verifies nothing (vukem v0.27.0, A20): whoever
	// signed up with someone else's address cannot get it verified by the
	// owner's click. The login comes back to the link and verifies it.
	c.Get(verify).AssertRedirect("/login")
	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "ana@example.com", "email_verified_at": nil})

	back := c.PostJSON("/login", map[string]any{"email": "ana@example.com", "password": "correct horse battery"})
	if loc := back.Header("Location"); !strings.Contains(loc, "/email/verify/") {
		t.Fatalf("login went to %q, want back to the verification link", loc)
	}

	c.Get(verify + "x").AssertForbidden() // tampered signature, now signed in
	c.Get(back.Header("Location")).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseMissing(t, app, "users", map[string]any{"email": "ana@example.com", "email_verified_at": nil})

	// Opening it again is harmless.
	c.Get(verify).AssertRedirect("/dashboard")
	c.Get("/dashboard").AssertOK().
		AssertInertia("Dashboard").
		PropEquals("auth.user.name", "Ana Silva").
		HasProp("auth.user.email_verified_at")
}

// A07: signing up with an existing email answers exactly like a new one,
// creates nothing, and mails the owner a notice (once per hour).
func TestRegister_ExistingEmailIsNotRevealed(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	register(t, app.Client(t).Inertia(""), "ana@example.com")
	before := len(app.Mails().SentTo("ana@example.com"))

	for range 2 {
		other := app.Client(t).Inertia("").WithHeader("Referer", "/register")
		res := other.PostJSON("/register", map[string]any{
			"name": "Other", "email": " ANA@example.com",
			"password": "another long password", "password_confirmation": "another long password",
		}).AssertRedirect("/register/check-email")

		if res.Header("Set-Cookie") != "" && strings.Contains(res.Header("Set-Cookie"), "remember") {
			t.Fatal("sign-up of an existing email logged in")
		}

		other.Get("/dashboard").AssertRedirect("/login")
	}

	vukemtest.AssertDatabaseCount(t, app, "users", 1)
	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "ana@example.com", "name": "Ana Silva"})

	sent := app.Mails().SentTo("ana@example.com")
	if len(sent) != before+1 {
		t.Fatalf("mails to the existing owner = %d, want %d (one notice, throttled)", len(sent), before+1)
	}

	if last := sent[len(sent)-1]; !strings.Contains(last.Text, "/forgot-password") || !strings.Contains(last.Text, "/login") {
		t.Fatalf("notice has no sign-in/reset links: %q", last.Text)
	}

	// The old password still works: the attempt changed nothing.
	app.Client(t).Inertia("").PostJSON("/login", map[string]any{"email": "ana@example.com", "password": "correct horse battery"}).
		AssertRedirect("/dashboard")
}

func TestLoginLogoutAndThrottle(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	register(t, app.Client(t).Inertia(""), "bob@example.com")

	c := app.Client(t).Inertia("").WithHeader("Referer", "/login")
	c.PostJSON("/login", map[string]any{"email": "bob@example.com", "password": "wrong"}).AssertRedirect("/login")
	c.Get("/login").AssertInertia("Auth/Login").HasProp("errors.email")

	c.PostJSON("/login", map[string]any{"email": "bob@example.com", "password": "correct horse battery"}).
		AssertRedirect("/dashboard")
	c.Get("/login").AssertRedirect("/")
	c.PostJSON("/logout", nil).AssertRedirect("/")
	c.Get("/").AssertInertia("Home").PropEquals("auth.user", nil)
	c.Get("/dashboard").AssertRedirect("/login")

	attacker := app.Client(t).Inertia("")
	// Spaces and case do not open new buckets: they are the same account.
	for _, email := range []string{"bob@example.com", " bob@example.com", "BOB@example.com ", "	bob@example.com"} {
		attacker.PostJSON("/login", map[string]any{"email": email, "password": "guess"})
	}

	attacker.PostJSON("/login", map[string]any{"email": "bob@example.com", "password": "guess"}).AssertStatus(429)
}

func TestPasswordReset(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	register(t, app.Client(t).Inertia(""), "cid@example.com")

	c := app.Client(t).Inertia("")

	// Unknown emails get the same answer and no mail.
	c.PostJSON("/forgot-password", map[string]any{"email": "nobody@example.com"}).AssertRedirect("/forgot-password")

	if len(app.Mails().SentTo("nobody@example.com")) != 0 {
		t.Fatal("mail sent to an unknown address")
	}

	c.PostJSON("/forgot-password", map[string]any{"email": "cid@example.com"}).AssertRedirect("/forgot-password")
	c.Get("/forgot-password").AssertInertia("Auth/ForgotPassword")

	reset := link(t, app, "cid@example.com")
	if !strings.HasPrefix(reset, "/reset-password/") {
		t.Fatalf("reset link = %s", reset)
	}

	u, _ := url.Parse(reset)
	token := strings.TrimPrefix(u.Path, "/reset-password/")

	c.Get(reset).AssertOK().AssertInertia("Auth/ResetPassword").PropEquals("token", token)

	form := map[string]any{
		"token": token, "email": "cid@example.com",
		"password": "a brand new password", "password_confirmation": "a brand new password",
	}

	userID := apiTokenFor(t, app, "cid@example.com")
	vukemtest.AssertDatabaseHas(t, app, "access_tokens", map[string]any{"user_id": userID})

	c.PostJSON("/reset-password", form).AssertRedirect("/login")

	// The new password ends API tokens issued with the old one.
	vukemtest.AssertDatabaseMissing(t, app, "access_tokens", map[string]any{"user_id": userID})

	// Single use.
	c.WithHeader("Referer", reset).PostJSON("/reset-password", form).AssertRedirect(reset)

	c.PostJSON("/login", map[string]any{"email": "cid@example.com", "password": "a brand new password"}).
		AssertRedirect("/dashboard")
}

// apiTokenFor issues an API token for email directly, to check that a
// password reset revokes it.
func apiTokenFor(t *testing.T, app *vukemtest.App, email string) string {
	t.Helper()

	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	u, err := acc.Users.ByCredentials(t.Context(), map[string]string{"email": email})
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := acc.Tokens.Create(t.Context(), u.AuthID(), "cli", []string{"*"}, time.Hour); err != nil {
		t.Fatal(err)
	}

	return u.AuthID()
}

// deliverQueuedMail sends the auth mails waiting in the queue (they are
// queued, not sent inline): a queue.Fake is drained by hand, the database
// queue by a short-lived worker until a mail to email arrives.
func deliverQueuedMail(t *testing.T, app *vukemtest.App, email string) {
	t.Helper()

	q, err := app.Queue()
	if err != nil {
		t.Fatal(err)
	}

	if fake, ok := q.Driver().(*queue.Fake); ok {
		for _, m := range fake.Messages() {
			if m.Kind != vukem.MailJobKind {
				continue
			}

			if err := q.Process(t.Context(), &m); err != nil {
				t.Fatal(err)
			}
		}

		return
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = q.Work(ctx, queue.WorkOptions{Queues: []string{"default"}, Concurrency: 1})
	}()

	for ctx.Err() == nil && len(app.Mails().SentTo(email)) == 0 {
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-done
}

// An expired verification link (signed in) explains itself and offers a
// new one instead of a bare 403.
func TestVerifyEmail_ExpiredLinkOffersANewOne(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := app.Client(t).Inertia("")
	register(t, c, "eva@example.com")

	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	var id int64
	if err := acc.DB.QueryRowContext(t.Context(), "SELECT id FROM users WHERE email = ?", "eva@example.com").Scan(&id); err != nil {
		t.Fatal(err)
	}

	// signed.Signer reads the wall clock (not the app clock): a 1 s link.
	verify, err := acc.Signer.Sign(fmt.Sprintf("/email/verify/%d", id), time.Second)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2100 * time.Millisecond)

	c.Get(verify).AssertRedirect("/email/verify")
	page := c.Get("/email/verify").AssertInertia("Auth/VerifyEmail").Page()
	if flash, _ := page["flash"].(map[string]any); flash["status"] != "This verification link expired. We can send you a new one." {
		t.Fatalf("flash = %v", page["flash"])
	}
	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "eva@example.com", "email_verified_at": nil})
}
