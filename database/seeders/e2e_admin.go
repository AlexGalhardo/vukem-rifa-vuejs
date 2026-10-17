package seeders

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth/hash"
	"github.com/AlexGalhardo/vukem/database"
)

// E2EAdmin creates a verified admin for the e2e suite (the admin payouts
// page under axe) when both E2E_ADMIN_EMAIL and E2E_ADMIN_PASSWORD are set,
// and does nothing otherwise. Only CI sets them; db:seed also refuses to run
// in production without --force. Running it again resets the password.
func E2EAdmin() vukem.Seeder {
	return vukem.SeederFunc(func(ctx context.Context, db *database.DB) error {
		email := strings.ToLower(strings.TrimSpace(os.Getenv("E2E_ADMIN_EMAIL")))
		password := os.Getenv("E2E_ADMIN_PASSWORD")

		if email == "" || password == "" {
			return nil
		}

		if len(password) < 12 {
			return fmt.Errorf("e2e admin: E2E_ADMIN_PASSWORD must have at least 12 characters")
		}

		h, err := hash.Default().Make(password)
		if err != nil {
			return fmt.Errorf("e2e admin: hash: %w", err)
		}

		p := db.Dialect().Placeholder
		now := time.Now().UTC()

		var n int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = "+p(1), email).Scan(&n); err != nil {
			return fmt.Errorf("e2e admin: %w", err)
		}

		if n > 0 {
			_, err = db.ExecContext(ctx, "UPDATE users SET password = "+p(1)+", is_admin = "+p(2)+
				", email_verified_at = "+p(3)+", updated_at = "+p(4)+" WHERE email = "+p(5), h, true, now, now, email)
		} else {
			_, err = db.ExecContext(ctx, "INSERT INTO users (name, email, password, email_verified_at, is_admin, created_at, updated_at) VALUES ("+
				p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", "+p(5)+", "+p(6)+", "+p(7)+")", "E2E Admin", email, h, now, true, now, now)
		}

		if err != nil {
			return fmt.Errorf("e2e admin: %w", err)
		}

		return nil
	})
}
