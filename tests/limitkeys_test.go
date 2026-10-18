package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

func TestLimitKeys_EmailNormalization(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"Ana.Maria+promo@Gmail.com": "anamaria@gmail.com",
		"a.n.a@googlemail.com":      "ana@gmail.com",
		"joao.silva+x@example.com":  "joao.silva@example.com",
		"maria@empresa.com.br":      "maria@empresa.com.br",
		"not-an-email":              "not-an-email",
	} {
		if got := requests.CanonicalEmail(in); got != want {
			t.Errorf("CanonicalEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

// vukem v0.27.0 ratelimit.ByIP keys IPv6 by /64: rotating addresses
// inside one /64 (behind the trusted proxy) shares one budget.
func TestLimitKeys_IPv6GroupsBySlash64(t *testing.T) {
	t.Parallel()

	f := newNumbersFixture(t, vukemtest.Options{Env: map[string]string{
		"TRUSTED_PROXIES": "192.0.2.0/24", "CLIENT_IP_HEADER": "X-Real-IP",
	}})

	limited := false
	for n := 1; n <= 12; n++ {
		c := f.app.Client(t).Inertia("").WithHeader("X-Real-IP", fmt.Sprintf("2001:db8:abcd:12::%x", n))
		if f.publicReserve(c, n, publicBuyer(fmt.Sprintf("v6-%d@example.com", n))).Status() == http.StatusTooManyRequests {
			limited = true
			break
		}
	}

	if !limited {
		t.Fatal("rotating IPv6 addresses inside one /64 escaped the per-IP limit")
	}

	// Another /64 has its own budget.
	c := f.app.Client(t).Inertia("").WithHeader("X-Real-IP", "2001:db8:abcd:13::1")
	if f.publicReserve(c, 20, publicBuyer("v6-other@example.com")).Status() == http.StatusTooManyRequests {
		t.Fatal("a different /64 shared the budget")
	}
}
