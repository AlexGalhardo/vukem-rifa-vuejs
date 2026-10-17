package migrations

import "github.com/AlexGalhardo/vukem/schema"

// The buyer has no account, so the locale of the payment page is stored on
// the reservation when the buyer confirms; the queued confirmation email is
// sent in it.
func m20261017190000AddLocaleToReservations() schema.Migration {
	return schema.Migration{
		ID: "20261017190000_add_locale_to_reservations",
		Up: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) {
				t.String("locale", 8).Default("")
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) { t.DropColumn("locale") })
		},
	}
}
