package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
	"github.com/AlexGalhardo/vukem/z"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/controllers/apiv1"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// apiFixture is an owner with a read+write key and a read-only key.
type apiFixture struct {
	app      *vukemtest.App
	web      *vukemtest.Client // the owner's browser session
	api      *vukemtest.Client // read + write key
	readOnly *vukemtest.Client
	token    string
}

func newAPIFixture(t *testing.T, email string, opts ...vukemtest.Options) apiFixture {
	t.Helper()

	app := vukemtest.New(t, bootstrap.New, opts...)
	web := owner(t, app, email)
	token := newAPIKey(t, web, "full", "raffles:read", "raffles:write")
	read := newAPIKey(t, web, "read", "raffles:read")

	return apiFixture{
		app: app, web: web, token: token,
		api:      app.Client(t).WithToken(token),
		readOnly: app.Client(t).WithToken(read),
	}
}

// createAPIRaffle creates a raffle through the API and returns its id.
func (f apiFixture) createAPIRaffle(t *testing.T, fields map[string]any) int64 {
	t.Helper()

	res := f.api.PostJSON("/api/v1/raffles", fields).AssertCreated().AssertMatchesOpenAPI()

	id, ok := res.JSONPath("id")
	if !ok {
		t.Fatalf("no id in %s", res.Body())
	}

	n, _ := id.(json.Number).Int64()

	return n
}

func TestAPI_RaffleCRUD(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "api.crud@example.com")

	fields := validRaffle()
	fields["title"] = "  Rifa via API  "
	id := f.createAPIRaffle(t, fields)
	path := fmt.Sprintf("/api/v1/raffles/%d", id)

	f.api.Get(path).AssertOK().AssertMatchesOpenAPI().
		AssertHeaderContains("Content-Type", "application/json").
		AssertJSONPath("title", "Rifa via API").
		AssertJSONPath("status", "active").
		AssertJSONPath("available", 100).
		AssertJSONPath("locked", false)

	slug, _ := f.api.Get(path).JSONPath("slug")
	f.api.Get(path).AssertJSONPath("public_url", "http://localhost:3000/r/"+fmt.Sprint(slug))

	f.api.PatchJSON(path, map[string]any{"title": "Novo título", "number_count": 50}).AssertOK().AssertMatchesOpenAPI().
		AssertJSONPath("title", "Novo título").
		AssertJSONPath("number_count", 50).
		AssertJSONPath("price_reais", 10).
		AssertJSONPath("available", 50)

	f.api.PatchJSON(path, map[string]any{"price_reais": 0}).AssertUnprocessable().AssertMatchesOpenAPI().
		AssertHeaderContains("Content-Type", "application/problem+json").
		AssertJSONValidationErrors("price_reais")

	// The browser sees the same raffle.
	f.web.Get(fmt.Sprintf("/raffles/%d", id)).AssertOK().AssertInertia("Raffles/Show").PropEquals("raffle.title", "Novo título")

	f.api.PostJSON(path+"/archive", nil).AssertOK().AssertMatchesOpenAPI().AssertJSONPath("status", "archived")

	f.api.Delete(path).AssertNoContent().AssertMatchesOpenAPI()
	f.api.Get(path).AssertNotFound().AssertMatchesOpenAPI().AssertProblem(http.StatusNotFound)
}

func TestAPI_CreateValidation(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "api.validation@example.com")

	tests := []struct {
		field string
		value any
	}{
		{"title", "ab"},
		{"title", "   ab   "}, // trimmed by requests.RaffleSchema
		{"number_count", 9},
		{"number_count", 1001},
		{"price_reais", 0},
		{"duration_days", 61},
		{"description", strings.Repeat("a", 5001)},
	}
	for _, tt := range tests {
		fields := validRaffle()
		fields[tt.field] = tt.value
		f.api.PostJSON("/api/v1/raffles", fields).AssertUnprocessable().AssertMatchesOpenAPI().
			AssertJSONValidationErrors(tt.field)
	}

	vukemtest.AssertDatabaseCount(t, f.app, "raffles", 0)
}

func TestAPI_ListIsPaginatedAndScoped(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "api.list@example.com")
	for range 3 {
		f.createAPIRaffle(t, validRaffle())
	}

	other := newAPIFixture(t, "api.list.other@example.com")
	other.createAPIRaffle(t, validRaffle())

	res := f.api.Get("/api/v1/raffles?per_page=2").AssertOK().AssertMatchesOpenAPI().
		AssertJSONCount("data", 2).
		AssertJSONPath("meta.total", 3).
		AssertJSONPath("meta.last_page", 2).
		AssertJSONPath("links.prev", nil)

	next, _ := res.JSONPath("links.next")
	f.api.Get(fmt.Sprint(next)).AssertOK().AssertMatchesOpenAPI().
		AssertJSONCount("data", 1).AssertJSONPath("links.next", nil).AssertJSONPath("meta.page", 2)

	f.api.Get("/api/v1/raffles?per_page=101").AssertUnprocessable().AssertMatchesOpenAPI()
}

func TestAPI_NumbersAndPaymentLink(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "api.links@example.com")
	id := f.createAPIRaffle(t, validRaffle())
	base := fmt.Sprintf("/api/v1/raffles/%d", id)

	f.api.Get(base+"/numbers").AssertOK().AssertMatchesOpenAPI().
		AssertJSONCount("data", 100).AssertJSONPath("data.6.number", 7).AssertJSONPath("data.6.status", "available")

	buyer := map[string]any{"buyer_name": "Maria Silva", "buyer_email": "Maria@Example.com", "buyer_phone": "(11) 98765-4321"}
	res := f.api.PostJSON(base+"/numbers/7/payment-links", buyer).AssertCreated().AssertMatchesOpenAPI().
		AssertJSONPath("number", 7)

	url, _ := res.JSONPath("url")
	if !strings.HasPrefix(fmt.Sprint(url), "http://localhost:3000/pay/") {
		t.Fatalf("url = %v", url)
	}

	vukemtest.AssertDatabaseHas(t, f.app, "reservations", map[string]any{
		"number": 7, "buyer_email": "maria@example.com", "buyer_phone": "+5511987654321", "status": "pending",
	})

	f.api.Get(base+"/numbers").AssertJSONPath("data.6.status", "reserved")

	// Taken number, unknown number, bad buyer.
	f.api.PostJSON(base+"/numbers/7/payment-links", buyer).AssertStatus(http.StatusConflict).AssertMatchesOpenAPI().
		AssertProblem(http.StatusConflict)
	f.api.PostJSON(base+"/numbers/101/payment-links", buyer).AssertNotFound().AssertMatchesOpenAPI()

	bad := map[string]any{"buyer_name": "Maria", "buyer_email": "maria@example.com", "buyer_phone": "123"}
	f.api.PostJSON(base+"/numbers/8/payment-links", bad).AssertUnprocessable().AssertMatchesOpenAPI().
		AssertJSONValidationErrors("buyer_phone")
}

func TestAPI_ExportAndBalance(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "api.export@example.com")
	id := f.createAPIRaffle(t, validRaffle())
	base := fmt.Sprintf("/api/v1/raffles/%d/export", id)

	f.api.Get(base).AssertOK().AssertMatchesOpenAPI().
		AssertHeaderContains("Content-Type", "application/json").
		AssertHeaderContains("Cache-Control", "no-store").
		AssertJSONPath("raffle.id", id).
		AssertJSONCount("numbers", 100)

	f.api.Get(base+"?format=csv").AssertOK().
		AssertHeaderContains("Content-Type", "text/csv").
		AssertHeaderContains("Content-Disposition", "attachment").
		AssertSee("record,number,status,paid_at,reservation_id")

	f.api.Get(base + "?format=xml").AssertUnprocessable().AssertMatchesOpenAPI().AssertJSONValidationErrors("format")

	f.api.Get("/api/v1/me/balance").AssertOK().AssertMatchesOpenAPI().
		AssertJSON(map[string]any{"balance_cents": 0, "currency": "BRL"})
}

func TestAPI_ErrorsAreProblemJSON(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "api.problems@example.com")

	// No Accept header at all, like curl.
	for _, target := range []string{"/api/v1/nope", "/api/v1/raffles/999"} {
		res := f.api.Get(target).AssertNotFound()
		res.AssertHeaderContains("Content-Type", "application/problem+json")
	}

	f.api.Do(http.MethodPut, "/api/v1/raffles", nil, "").AssertStatus(http.StatusMethodNotAllowed).
		AssertHeaderContains("Content-Type", "application/problem+json")
}

func TestAPI_OpenAPIDocument(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)
	res := app.Client(t).Get("/api/openapi.json").AssertOK().
		AssertJSONPath("openapi", "3.1.0").
		AssertJSONPath("components.securitySchemes.bearer.scheme", "bearer")

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal([]byte(res.Body()), &doc); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"/api/v1/raffles":                                     {"get", "post"},
		"/api/v1/raffles/{id}":                                {"get", "patch", "delete"},
		"/api/v1/raffles/{id}/archive":                        {"post"},
		"/api/v1/raffles/{id}/numbers":                        {"get"},
		"/api/v1/raffles/{id}/numbers/{number}/payment-links": {"post"},
		"/api/v1/raffles/{id}/export":                         {"get"},
		"/api/v1/me/balance":                                  {"get"},
	}
	if len(doc.Paths) != len(want) {
		t.Errorf("document has %d paths, want only the %d API v1 paths: %v", len(doc.Paths), len(want), doc.Paths)
	}

	for path, methods := range want {
		for _, m := range methods {
			if _, ok := doc.Paths[path][m]; !ok {
				t.Errorf("missing %s %s", m, path)
			}
		}
	}
}

func TestAPI_DocsPageKeepsCSP(t *testing.T) {
	t.Parallel()

	app := vukemtest.New(t, bootstrap.New)

	res := app.Client(t).Get("/api/docs").AssertOK().
		AssertHeaderContains("Content-Type", "text/html").
		AssertSee(`/api/openapi.json`).
		AssertSee(`/api/docs/scalar.js`)

	csp := res.Header("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' 'nonce-") || strings.Contains(csp, "cdn") ||
		strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Fatalf("CSP = %q", csp)
	}

	app.Client(t).Get("/api/docs/scalar.js").AssertOK().AssertHeaderContains("Content-Type", "javascript")
}

// The tag rules that document and pre-validate the API bodies must match
// the app's request schemas, which run on every request too.
func TestAPI_BodyRulesMatchRequestSchemas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tagged z.AnySchema
		app    z.AnySchema
	}{
		{"raffle", z.For[apiv1.CreateRaffle](), requests.RaffleSchema},
	}
	for _, tt := range tests {
		got, want := constraints(z.JSONSchema(tt.tagged)), constraints(z.JSONSchema(tt.app))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: tags %v, request schema %v", tt.name, got, want)
		}
	}
}

// constraints keeps the length and range keywords of each property.
func constraints(schema map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}

	props, _ := schema["properties"].(map[string]any)
	for name, p := range props {
		m, _ := p.(map[string]any)
		out[name] = map[string]any{}

		for _, k := range []string{"minLength", "maxLength", "minimum", "maximum"} {
			if v, ok := m[k]; ok {
				out[name][k] = fmt.Sprint(v)
			}
		}
	}

	return out
}
