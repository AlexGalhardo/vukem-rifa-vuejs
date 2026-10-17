package bootstrap

import (
	"errors"
	"strings"

	vukem "github.com/AlexGalhardo/vukem"
)

// ErrInsecureAppURL refuses to boot production on a non-HTTPS APP_URL:
// links in emails (verification, reset, payment links) would travel and be
// opened in clear text, and Secure cookies would never come back.
var ErrInsecureAppURL = errors.New("bootstrap: APP_URL must be an https:// URL in production")

// ProductionGuards fails the boot on production settings that would weaken
// the app (OWASP A02/A05).
func ProductionGuards() vukem.Provider {
	return vukem.ProviderFunc(func(app *vukem.App) error {
		if app.Config.IsProduction() && !strings.HasPrefix(app.Config.URL, "https://") {
			return ErrInsecureAppURL
		}

		return nil
	})
}
