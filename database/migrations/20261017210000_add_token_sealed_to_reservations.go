package migrations

import "github.com/AlexGalhardo/vukem/schema"

// The plain payment-link token, sealed with APP_KEY (AES-256-GCM, orm
// Encrypted), so the raffle owner can see and share each live link for its
// ten minutes. Lookups still use token_hash; rows created before this
// migration have no sealed token and are simply not listed.
func m20261017210000AddTokenSealedToReservations() schema.Migration {
	return schema.Migration{
		ID: "20261017210000_add_token_sealed_to_reservations",
		Up: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) { t.Text("token_sealed").Nullable() })
		},
		Down: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) { t.DropColumn("token_sealed") })
		},
	}
}
