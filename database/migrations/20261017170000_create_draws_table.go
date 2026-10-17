package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Draws store everything needed to recompute the winner (PLAN.md D12):
// the crypto/rand seed, the participant count and the SHA-256 of the sorted
// paid-number list. One draw per raffle.
func m20261017170000CreateDrawsTable() schema.Migration {
	return schema.Migration{
		ID: "20261017170000_create_draws_table",
		Up: func(s *schema.Schema) error {
			if err := s.Create("draws", func(t *schema.Table) {
				t.ID()
				t.ForeignID("raffle_id").Unique().Constrained("raffles").CascadeOnDelete()
				t.Timestamp("drawn_at")
				t.String("seed", 64)
				t.Integer("participants_count")
				t.String("participants_sha256", 64)
				t.Integer("winning_number").Nullable()
				t.String("algorithm", 16)
				t.String("trigger", 16)
				t.Timestamps()
			}); err != nil {
				return err
			}

			if s.Dialect() != "postgres" {
				return nil
			}

			return s.Raw(`ALTER TABLE draws ADD CONSTRAINT draws_trigger_valid CHECK ("trigger" IN ('deadline','sold_out'))`)
		},
		Down: func(s *schema.Schema) error { return s.DropIfExists("draws") },
	}
}
