package mailtrap_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/tests/mailtrap"
)

func TestWaitLink(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Api-Token") != "tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/api/sandboxes/42/messages":
			if r.URL.Query().Get("search") != "ana@example.com" {
				t.Errorf("search = %q", r.URL.Query().Get("search"))
			}

			fmt.Fprint(w, `[{"id":7,"subject":"Verify","to_email":"other@example.com","txt_path":"/api/sandboxes/42/messages/6/body.txt"},
				{"id":8,"subject":"Verify","to_email":"Ana@Example.com","txt_path":"/api/sandboxes/42/messages/8/body.txt"}]`)
		case "/api/sandboxes/42/messages/8/body.txt":
			fmt.Fprint(w, "Hello,\nopen http://localhost:3000/email/verify/1?signature=abc to verify.\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &mailtrap.Client{BaseURL: srv.URL, Token: "tok", SandboxID: "42", HTTP: srv.Client()}

	link, err := c.WaitLink(t.Context(), "ana@example.com", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if want := "http://localhost:3000/email/verify/1?signature=abc"; link != want {
		t.Fatalf("link = %q, want %q", link, want)
	}
}

func TestMessages_ErrorStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	c := &mailtrap.Client{BaseURL: srv.URL, Token: "tok", SandboxID: "42", HTTP: srv.Client()}
	if _, err := c.Messages(t.Context(), ""); err == nil {
		t.Fatal("want an error on 429")
	}
}
