package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/routes"
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

func TestLimitKeys_IPv6GroupsBySlash64(t *testing.T) {
	t.Parallel()

	a := routes.IPKey("2001:db8:abcd:12:1111:2222:3333:4444")
	b := routes.IPKey("2001:db8:abcd:12:ffff:ffff:ffff:ffff")
	c := routes.IPKey("2001:db8:abcd:13::1")

	if a != b || a == c {
		t.Fatalf("keys %q %q %q: want one /64 per key", a, b, c)
	}

	if routes.IPKey("203.0.113.7") != "ip:203.0.113.7" {
		t.Fatalf("IPv4 key = %q", routes.IPKey("203.0.113.7"))
	}
}
