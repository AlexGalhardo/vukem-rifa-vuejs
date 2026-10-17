package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

func TestHome(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)

	// First visit: HTML with the page object; Inertia visits: JSON.
	app.Client(t).Get("/").AssertOK().AssertInertia("Home").HasProp("appName").HasProp("locales")
	app.Client(t).Inertia(app.Inertia.Version()).Get("/").AssertOK().AssertInertia("Home")
}

func TestHealth(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)

	app.Client(t).Get("/up").AssertOK()
}

func TestHome_Locale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		accept string
		want   string
	}{
		{"default is pt-BR", "/", "", "pt-BR"},
		{"accept-language en", "/", "en-US,en;q=0.9", "en"},
		{"query wins", "/?lang=en", "pt-BR", "en"},
		{"unknown falls back", "/?lang=xx", "", "pt-BR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"APP_LOCALE": "pt-BR"}})

			c := app.Client(t)
			if tt.accept != "" {
				c = c.WithHeader("Accept-Language", tt.accept)
			}

			c.Get(tt.target).AssertOK().AssertInertia("Home").PropEquals("locale", tt.want)
		})
	}
}
