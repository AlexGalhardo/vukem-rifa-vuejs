package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Raffles and their numbers (1..N, pre-created). CHECK constraints are
// added on Postgres, the production database; SQLite (unit tests) relies on
// the z schemas.
//
// workaround(vukem v0.21.1): schema.Table has no Check(); constraints go through Raw.
func m20261017140000CreateRafflesTables() schema.Migration {
	return schema.Migration{
		ID: "20261017140000_create_raffles_tables",
		Up: func(s *schema.Schema) error {
			if err := s.Create("raffles", func(t *schema.Table) {
				t.ID()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("slug", 32).Unique()
				t.String("title", 120)
				t.Text("description")
				t.Integer("number_count")
				t.Integer("price_reais")
				t.Integer("duration_days")
				t.Timestamp("starts_at")
				t.Timestamp("ends_at").Index()
				t.String("status", 16).Default("active").Index()
				t.Timestamp("locked_at").Nullable()
				t.Timestamp("archived_at").Nullable()
				t.Timestamps()
			}); err != nil {
				return err
			}

			if err := s.Create("raffle_numbers", func(t *schema.Table) {
				t.ID()
				t.ForeignID("raffle_id").Constrained("raffles").CascadeOnDelete()
				t.Integer("number")
				t.String("status", 16).Default("available")
				t.Timestamp("paid_at").Nullable()
				t.Timestamps()
				t.UniqueIndex("raffle_id", "number")
				t.Index("raffle_id", "status")
			}); err != nil {
				return err
			}

			if s.Dialect() != "postgres" {
				return nil
			}

			for _, stmt := range []string{
				`ALTER TABLE raffles ADD CONSTRAINT raffles_number_count_range CHECK (number_count BETWEEN 10 AND 1000)`,
				`ALTER TABLE raffles ADD CONSTRAINT raffles_price_range CHECK (price_reais BETWEEN 1 AND 100000)`,
				`ALTER TABLE raffles ADD CONSTRAINT raffles_duration_range CHECK (duration_days BETWEEN 7 AND 60)`,
				`ALTER TABLE raffles ADD CONSTRAINT raffles_status_valid CHECK (status IN ('active','drawn','closed','archived'))`,
				`ALTER TABLE raffles ADD CONSTRAINT raffles_title_length CHECK (char_length(title) BETWEEN 3 AND 120)`,
				`ALTER TABLE raffles ADD CONSTRAINT raffles_description_length CHECK (char_length(description) <= 5000)`,
				`ALTER TABLE raffles ADD CONSTRAINT raffles_ends_after_start CHECK (ends_at > starts_at)`,
				`ALTER TABLE raffle_numbers ADD CONSTRAINT raffle_numbers_number_positive CHECK (number >= 1)`,
				`ALTER TABLE raffle_numbers ADD CONSTRAINT raffle_numbers_status_valid CHECK (status IN ('available','reserved','paid'))`,
			} {
				if err := s.Raw(stmt); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			if err := s.DropIfExists("raffle_numbers"); err != nil {
				return err
			}

			return s.DropIfExists("raffles")
		},
	}
}
