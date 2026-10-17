package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Admins record payouts to raffle owners; set by hand in the database.
func m20261017130000AddIsAdminToUsers() schema.Migration {
	return schema.Migration{
		ID: "20261017130000_add_is_admin_to_users",
		Up: func(s *schema.Schema) error {
			return s.Table("users", func(t *schema.Table) {
				t.Boolean("is_admin").Default(false)
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("users", func(t *schema.Table) { t.DropColumn("is_admin") })
		},
	}
}
