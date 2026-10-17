package payments

import (
	"errors"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

// Config is the AbacatePay configuration (ABACATEPAY_* variables).
type Config struct {
	APIKey  string
	BaseURL string
	// Fake selects the in-memory gateway (development and e2e only).
	Fake       bool
	Production bool
}

// ErrFakeInProduction refuses to boot a production app on the fake gateway.
var ErrFakeInProduction = errors.New("payments: ABACATEPAY_FAKE=true is refused in production")

// ConfigFrom reads the configuration with get (app.Source.Get).
func ConfigFrom(get func(string) string, production bool) Config {
	return Config{
		APIKey:     get("ABACATEPAY_API_KEY"),
		BaseURL:    get("ABACATEPAY_BASE_URL"),
		Fake:       get("ABACATEPAY_FAKE") == "true",
		Production: production,
	}
}

// Gateway returns the configured gateway: the fake when asked for outside
// production, the real client when an API key is set, and nil (payments
// unavailable, the app still boots) otherwise.
func (c Config) Gateway() (abacatepay.Gateway, error) {
	switch {
	case c.Fake && c.Production:
		return nil, ErrFakeInProduction
	case c.Fake:
		return abacatepay.NewFake(), nil
	case c.APIKey != "":
		return abacatepay.New(c.BaseURL, c.APIKey, nil), nil
	default:
		return nil, nil
	}
}
