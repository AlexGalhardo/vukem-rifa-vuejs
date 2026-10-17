package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// newAPIKey creates an API key on the settings page as c's owner and
// returns the plain token shown once.
func newAPIKey(t *testing.T, c *vukemtest.Client, name string, abilities ...string) string {
	t.Helper()

	page := c.PostJSON("/settings/api-keys", map[string]any{"name": name, "abilities": abilities}).
		AssertOK().AssertInertia("Settings/ApiKeys")

	v, ok := page.Prop("plain_token")
	token, _ := v.(string)

	if !ok || !strings.Contains(token, ".") {
		t.Fatalf("plain_token = %v", v)
	}

	return token
}

func keyID(token string) string {
	id, _, _ := strings.Cut(token, ".")
	return id
}

func TestAPIKeys_CreateShowsTokenOnce(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "chaves@example.com")

	c.Get("/settings/api-keys").AssertOK().AssertInertia("Settings/ApiKeys").
		HasProp("keys", 0).PropEquals("plain_token", nil).HasProp("abilities", 2)

	token := newAPIKey(t, c, "Integração ERP", "raffles:read", "raffles:write")

	page := c.Get("/settings/api-keys").AssertOK().AssertDontSee(token).AssertInertia("Settings/ApiKeys")
	page.HasProp("keys", 1).PropEquals("plain_token", nil).
		PropEquals("keys.0.id", keyID(token)).
		PropEquals("keys.0.name", "Integração ERP").
		PropEquals("keys.0.abilities", []any{"raffles:read", "raffles:write"})

	// Only the hash is stored.
	vukemtest.AssertDatabaseMissing(t, app, "access_tokens", map[string]any{"hash": strings.SplitN(token, ".", 2)[1]})
}

func TestAPIKeys_Validation(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	c := owner(t, app, "chaves.validacao@example.com")

	tests := []struct {
		name  string
		input map[string]any
		field string
	}{
		{"empty name", map[string]any{"name": " ", "abilities": []string{"raffles:read"}}, "name"},
		{"long name", map[string]any{"name": strings.Repeat("a", 61), "abilities": []string{"raffles:read"}}, "name"},
		{"no ability", map[string]any{"name": "x", "abilities": []string{}}, "abilities"},
		{"unknown ability", map[string]any{"name": "x", "abilities": []string{"*"}}, "abilities.0"},
		{"admin ability", map[string]any{"name": "x", "abilities": []string{"raffles:read", "admin"}}, "abilities.1"},
	}
	for _, tt := range tests {
		c.PostJSON("/settings/api-keys", tt.input).AssertStatus(http.StatusSeeOther)
		page := c.Get("/settings/api-keys").AssertInertia("Settings/ApiKeys").HasProp("keys", 0)

		// Field keys contain dots (abilities.0), so read the map directly.
		errors, _ := page.Prop("errors")
		if m, _ := errors.(map[string]any); m[tt.field] == nil {
			t.Errorf("%s: errors = %v, want %s", tt.name, errors, tt.field)
		}
	}

	vukemtest.AssertDatabaseCount(t, app, "access_tokens", 0)
}

func TestAPIKeys_RevokeIsOwnerChecked(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	ana := owner(t, app, "ana.chaves@example.com")
	bia := owner(t, app, "bia.chaves@example.com")

	token := newAPIKey(t, ana, "ERP", "raffles:read")
	app.Client(t).WithToken(token).Get("/api/v1/me/balance").AssertOK()

	// Another owner cannot revoke (or confirm) Ana's key.
	bia.Delete("/settings/api-keys/" + keyID(token)).AssertNotFound()
	app.Client(t).WithToken(token).Get("/api/v1/me/balance").AssertOK()

	ana.Delete("/settings/api-keys/" + keyID(token)).AssertRedirect("/settings/api-keys")
	ana.Get("/settings/api-keys").AssertInertia("Settings/ApiKeys").HasProp("keys", 0)

	// A revoked key is rejected.
	app.Client(t).WithToken(token).Get("/api/v1/me/balance").AssertUnauthorized()
	ana.Delete("/settings/api-keys/" + keyID(token)).AssertNotFound()
}

func TestAPIKeys_RequireVerifiedOwner(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)

	app.Client(t).Get("/settings/api-keys").AssertRedirect("/login")
	app.Client(t).PostJSON("/settings/api-keys", map[string]any{"name": "x", "abilities": []string{"raffles:read"}}).
		AssertUnauthorized()
}
