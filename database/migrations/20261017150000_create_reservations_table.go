package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Reservations are the per-number payment links (PLAN.md §4.2). On
// Postgres a partial unique index allows one live reservation per number.
func m20261017150000CreateReservationsTable() schema.Migration {
	return schema.Migration{
		ID: "20261017150000_create_reservations_table",
		Up: func(s *schema.Schema) error {
			if err := s.Create("reservations", func(t *schema.Table) {
				t.ID()
				t.ForeignID("raffle_id").Constrained("raffles").CascadeOnDelete()
				t.ForeignID("number_id").Constrained("raffle_numbers").CascadeOnDelete()
				t.Integer("number")
				t.String("buyer_name", 100)
				t.String("buyer_email", 255)
				t.String("buyer_phone", 20)
				t.String("token_hash", 64).Unique()
				t.String("status", 20).Default("pending").Index()
				t.Timestamp("expires_at").Index()
				t.Integer("amount_cents")
				t.String("charge_id", 100).Nullable().Unique()
				t.Text("br_code").Nullable()
				t.Text("br_code_base64").Nullable()
				t.Timestamp("paid_at").Nullable()
				t.Timestamps()
				t.Index("number_id", "status")
			}); err != nil {
				return err
			}

			if err := s.Table("raffle_numbers", func(t *schema.Table) {
				t.BigInteger("reservation_id").Nullable()
			}); err != nil {
				return err
			}

			if s.Dialect() != "postgres" {
				return nil
			}

			for _, stmt := range []string{
				`ALTER TABLE reservations ADD CONSTRAINT reservations_status_valid CHECK (status IN ('pending','awaiting_payment','paid','expired','cancelled'))`,
				`ALTER TABLE reservations ADD CONSTRAINT reservations_amount_positive CHECK (amount_cents >= 100)`,
				`CREATE UNIQUE INDEX reservations_one_live_per_number ON reservations (number_id) WHERE status IN ('pending','awaiting_payment')`,
				`ALTER TABLE raffle_numbers ADD CONSTRAINT raffle_numbers_reservation_fk FOREIGN KEY (reservation_id) REFERENCES reservations (id) ON DELETE SET NULL`,
			} {
				if err := s.Raw(stmt); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			if err := s.Table("raffle_numbers", func(t *schema.Table) { t.DropColumn("reservation_id") }); err != nil {
				return err
			}

			return s.DropIfExists("reservations")
		},
	}
}
