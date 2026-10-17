package payments_test

import (
	"errors"
	"testing"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

func TestFormatBRL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		cents  int64
		pt, en string
	}{
		{0, "R$ 0,00", "R$0.00"},
		{5, "R$ 0,05", "R$0.05"},
		{1000, "R$ 10,00", "R$10.00"},
		{123456, "R$ 1.234,56", "R$1,234.56"},
		{100000000, "R$ 1.000.000,00", "R$1,000,000.00"},
		{-90050, "-R$ 900,50", "-R$900.50"},
	} {
		if got := payments.FormatBRL(tc.cents, "pt-BR"); got != tc.pt {
			t.Errorf("FormatBRL(%d, pt-BR) = %q, want %q", tc.cents, got, tc.pt)
		}

		if got := payments.FormatBRL(tc.cents, "en"); got != tc.en {
			t.Errorf("FormatBRL(%d, en) = %q, want %q", tc.cents, got, tc.en)
		}
	}
}

func TestMask(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{"maria@example.com": "m•••@example.com", "x@y.z": "x•••@y.z", "broken": "•••", "@y.z": "•••"} {
		if got := payments.MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}

	for in, want := range map[string]string{"+5511987654321": "+55 11 •••••-4321", "+551133334444": "+55 11 •••••-4444", "+1": "•••"} {
		if got := payments.MaskPhone(in); got != want {
			t.Errorf("MaskPhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConfig_Gateway(t *testing.T) {
	t.Parallel()

	if _, err := (payments.Config{Fake: true, Production: true}).Gateway(); !errors.Is(err, payments.ErrFakeInProduction) {
		t.Fatalf("fake in production: err = %v", err)
	}

	if gw, err := (payments.Config{Fake: true}).Gateway(); err != nil {
		t.Fatal(err)
	} else if _, ok := gw.(*abacatepay.Fake); !ok {
		t.Fatalf("gateway = %T, want fake", gw)
	}

	if gw, err := (payments.Config{APIKey: "abc_dev_x", Production: true}).Gateway(); err != nil {
		t.Fatal(err)
	} else if c, ok := gw.(*abacatepay.Client); !ok || c.BaseURL() != abacatepay.DefaultBaseURL {
		t.Fatalf("gateway = %T", gw)
	}

	if gw, err := (payments.Config{Production: true}).Gateway(); err != nil || gw != nil {
		t.Fatalf("unconfigured = %v, %v; want nil, nil", gw, err)
	}
}
