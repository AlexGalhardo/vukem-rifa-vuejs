package abacatepay_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

const pixResponse = `{"data":{"id":"pix_char_abc123","amount":1000,"status":"PENDING","devMode":true,
"brCode":"00020101021226950014br.gov.bcb.pix","brCodeBase64":"data:image/png;base64,iVBORw0KGgo=",
"platformFee":80,"receiptUrl":null,"createdAt":"2026-10-17T18:00:00.000Z","updatedAt":"2026-10-17T18:00:00.000Z",
"expiresAt":"2026-10-17T18:10:00.000Z","metadata":{}},"success":true,"error":null}`

func server(t *testing.T, h http.HandlerFunc) *abacatepay.Client {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return abacatepay.New(srv.URL, "abc_dev_key", nil)
}

func TestClient_CreatePix(t *testing.T) {
	t.Parallel()

	var got map[string]any

	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/transparents/create" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer abc_dev_key" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("headers = %v", r.Header)
		}

		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}

		_, _ = io.WriteString(w, pixResponse)
	})

	ch, err := c.CreatePix(t.Context(), abacatepay.PixRequest{
		AmountCents: 1000, ExpiresIn: 540, Description: "Rifa da moto #7", ExternalID: "42",
	})
	if err != nil {
		t.Fatal(err)
	}

	if ch.ID != "pix_char_abc123" || ch.AmountCents != 1000 || ch.Status != abacatepay.StatusPending ||
		!strings.HasPrefix(ch.BrCodeBase64, "data:image/png;base64,") || ch.BrCode == "" || !ch.DevMode {
		t.Fatalf("charge = %+v", ch)
	}

	data, _ := got["data"].(map[string]any)
	if got["method"] != "PIX" || data["amount"] != float64(1000) || data["expiresIn"] != float64(540) ||
		data["externalId"] != "42" || data["description"] != "Rifa da moto #7" {
		t.Fatalf("body = %v", got)
	}
}

func TestClient_RejectsInvalidResponses(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status int
		body   string
	}{
		"api error":         {http.StatusUnauthorized, `{"error":"Token de autenticação inválido ou ausente."}`},
		"server error":      {http.StatusBadGateway, `<html>bad gateway</html>`},
		"success false":     {http.StatusOK, `{"data":null,"success":false,"error":"Insufficient permissions"}`},
		"not json":          {http.StatusOK, `nope`},
		"missing id":        {http.StatusOK, `{"data":{"amount":1000,"status":"PENDING","brCode":"x","brCodeBase64":"data:image/png;base64,x"},"success":true,"error":null}`},
		"unknown status":    {http.StatusOK, strings.Replace(pixResponse, `"PENDING"`, `"WHATEVER"`, 1)},
		"image not a data:": {http.StatusOK, strings.Replace(pixResponse, `data:image/png;base64,`, `javascript:`, 1)},
		"negative amount":   {http.StatusOK, strings.Replace(pixResponse, `"amount":1000`, `"amount":-5`, 1)},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := server(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})

			if _, err := c.CreatePix(t.Context(), abacatepay.PixRequest{AmountCents: 1000, ExpiresIn: 60}); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestClient_APIErrorKeepsStatus(t *testing.T) {
	t.Parallel()

	c := server(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"data":null,"success":false,"error":"Insufficient permissions"}`)
	})

	_, err := c.Check(t.Context(), "pix_char_abc123")

	var apiErr *abacatepay.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden || apiErr.Message != "Insufficient permissions" {
		t.Fatalf("err = %v", err)
	}
}

func TestClient_CheckAndSimulate(t *testing.T) {
	t.Parallel()

	c := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "pix_char_abc123" {
			t.Errorf("id = %q", r.URL.Query().Get("id"))
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/transparents/check":
			_, _ = io.WriteString(w, `{"data":{"id":"pix_char_abc123","status":"PAID","expiresAt":"2026-10-17T18:10:00.000Z"},"success":true,"error":null}`)
		case r.Method == http.MethodPost && r.URL.Path == "/transparents/simulate-payment":
			_, _ = io.WriteString(w, strings.Replace(pixResponse, `"PENDING"`, `"PAID"`, 1))
		default:
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	ch, err := c.Check(t.Context(), "pix_char_abc123")
	if err != nil || ch.Status != abacatepay.StatusPaid || ch.ID != "pix_char_abc123" {
		t.Fatalf("check = %+v, %v", ch, err)
	}

	ch, err = c.SimulatePayment(t.Context(), "pix_char_abc123")
	if err != nil || ch.Status != abacatepay.StatusPaid {
		t.Fatalf("simulate = %+v, %v", ch, err)
	}
}

func TestClient_Timeout(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	c := abacatepay.New(srv.URL, "k", &http.Client{Timeout: 50 * time.Millisecond})

	start := time.Now()
	if _, err := c.Check(t.Context(), "pix_char_abc123"); err == nil {
		t.Fatal("want timeout error")
	}

	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout not applied")
	}
}

func TestNew_DefaultsToOfficialBaseURLWithTimeout(t *testing.T) {
	t.Parallel()

	c := abacatepay.New("", "k", nil)
	if c.BaseURL() != abacatepay.DefaultBaseURL {
		t.Fatalf("base url = %q", c.BaseURL())
	}
}
