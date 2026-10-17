package tests

import (
	"encoding/json"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/resources"
)

// Every key must exist in every catalog: a missing key silently shows the
// pt-BR text (Vue) or the key itself (Go).
func TestLangCatalogs_SameKeys(t *testing.T) {
	t.Parallel()

	keys := map[string][]string{}

	for _, name := range []string{"pt-BR.json", "en.json"} {
		raw, err := fs.ReadFile(resources.Lang(), name)
		if err != nil {
			t.Fatal(err)
		}

		var tree map[string]any
		if err := json.Unmarshal(raw, &tree); err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		flat := map[string]struct{}{}
		flatten("", tree, flat)
		keys[name] = slices.Sorted(maps.Keys(flat))
	}

	if !slices.Equal(keys["pt-BR.json"], keys["en.json"]) {
		t.Fatalf("catalog keys differ:\npt-BR: %v\nen:    %v", keys["pt-BR.json"], keys["en.json"])
	}
}

func flatten(prefix string, node map[string]any, out map[string]struct{}) {
	for k, v := range node {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if child, ok := v.(map[string]any); ok {
			flatten(key, child, out)
			continue
		}

		out[key] = struct{}{}
	}
}

func TestForgotPassword_FlashIsLocalized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang string
		want string
	}{
		{"pt-BR", "Se esse e-mail tiver uma conta, enviamos um link para redefinir a senha."},
		{"en", "If that email has an account, we sent a password reset link."},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()

			app := vukemtest.New(t, bootstrap.New)
			c := app.Client(t).Inertia("").WithCookie("lang", tt.lang)

			c.PostJSON("/forgot-password", map[string]any{"email": "nobody@example.com"}).AssertRedirect("/forgot-password")
			page := c.Get("/forgot-password").AssertInertia("Auth/ForgotPassword").Page()

			// Inertia v3 sends flash data next to props, not inside them.
			flash, _ := page["flash"].(map[string]any)
			if got := flash["status"]; got != tt.want {
				t.Fatalf("flash.status = %v, want %q", got, tt.want)
			}
		})
	}
}

func TestRegister_ValidationIsLocalized(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := app.Client(t).Inertia("").WithCookie("lang", "pt-BR").WithHeader("Referer", "/register")

	c.PostJSON("/register", map[string]any{
		"name": "", "email": "not-an-email", "password": "short", "password_confirmation": "short",
	}).AssertRedirect("/register")

	c.Get("/register").AssertInertia("Auth/Register").
		PropEquals("errors.email", "O campo e-mail deve ser um e-mail válido.").
		PropEquals("errors.password", "O campo senha deve ter pelo menos 12 caracteres.")
}

func TestVerificationEmail_IsLocalized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang, subject, body string
	}{
		{"pt-BR", "Confirme seu e-mail", "Confirme seu endereço de e-mail"},
		{"en", "Verify your email address", "Please confirm your email address"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()

			app := vukemtest.New(t, bootstrap.New)
			register(t, app.Client(t).Inertia("").WithCookie("lang", tt.lang), "mail@example.com")

			sent := app.Mails().SentTo("mail@example.com")
			if len(sent) != 1 {
				t.Fatalf("sent %d mails, want 1", len(sent))
			}

			if sent[0].Subject != tt.subject {
				t.Errorf("subject = %q, want %q", sent[0].Subject, tt.subject)
			}

			if !strings.Contains(sent[0].Text, tt.body) {
				t.Errorf("text = %q, want it to contain %q", sent[0].Text, tt.body)
			}
		})
	}
}
