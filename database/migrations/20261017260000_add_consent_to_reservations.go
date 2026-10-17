package migrations

import "github.com/AlexGalhardo/vukem/schema"

// LGPD: when a public buyer accepted the privacy notice and which version
// of the text they saw.
func m20261017260000AddConsentToReservations() schema.Migration {
	return schema.Migration{
		ID: "20261017260000_add_consent_to_reservations",
		Up: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) {
				t.Timestamp("consent_at").Nullable()
				t.String("consent_version", 32).Nullable()
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) { t.DropColumn("consent_at", "consent_version") })
		},
	}
}
