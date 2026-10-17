package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Opt-in contact details the organizer may show on their public raffle
// pages (owner request). Everything is off by default.
func m20261017250000AddPublicContactToUsers() schema.Migration {
	return schema.Migration{
		ID: "20261017250000_add_public_contact_to_users",
		Up: func(s *schema.Schema) error {
			return s.Table("users", func(t *schema.Table) {
				t.Boolean("show_email").Default(false)
				t.String("whatsapp", 20).Nullable()
				t.Boolean("show_whatsapp").Default(false)
				t.String("telegram", 40).Nullable()
				t.Boolean("show_telegram").Default(false)
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("users", func(t *schema.Table) {
				t.DropColumn("show_email", "whatsapp", "show_whatsapp", "telegram", "show_telegram")
			})
		},
	}
}
