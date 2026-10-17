package migrations

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/schema"
)

// API keys are soft revoked since vukem v0.26.0 (auth.Tokens sets
// revoked_at instead of deleting the row, so the audit keeps them).
func m20261017230000AddRevokedAtToAccessTokens() schema.Migration {
	return auth.TokensRevokedAtMigration("20261017230000_add_revoked_at_to_access_tokens")
}
