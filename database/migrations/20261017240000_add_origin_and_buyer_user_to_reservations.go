package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Where a payment link came from (owner page, public raffle page, API) and
// the buyer's account, when the buyer is signed in or owns a verified
// account with the buyer email (purchase history).
//
// workaround(vukem v0.21.1): schema.Table has no Check(); the CHECK goes through Raw.
func m20261017240000AddOriginAndBuyerUserToReservations() schema.Migration {
	return schema.Migration{
		ID: "20261017240000_add_origin_and_buyer_user_to_reservations",
		Up: func(s *schema.Schema) error {
			if err := s.Table("reservations", func(t *schema.Table) {
				t.String("origin", 16).Default("owner")
				t.ForeignID("buyer_user_id").Nullable().Constrained("users").NullOnDelete()
				t.Index("buyer_user_id")
				t.Index("raffle_id", "buyer_email")
			}); err != nil {
				return err
			}

			if s.Dialect() != "postgres" {
				return nil
			}

			return s.Raw(`ALTER TABLE reservations ADD CONSTRAINT reservations_origin_valid CHECK (origin IN ('owner','public','api'))`)
		},
		Down: func(s *schema.Schema) error {
			return s.Table("reservations", func(t *schema.Table) { t.DropColumn("origin", "buyer_user_id") })
		},
	}
}
