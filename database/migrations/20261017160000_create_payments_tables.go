package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Payment events (webhook audit + idempotency), the append-only ledger and
// manual payouts (PLAN.md §4.2, §4.4). On Postgres the ledger rejects
// UPDATE and DELETE with a trigger.
//
// workaround(vukem v0.21.1): schema.Table has no Check(); constraints go through Raw.
func m20261017160000CreatePaymentsTables() schema.Migration {
	return schema.Migration{
		ID: "20261017160000_create_payments_tables",
		Up: func(s *schema.Schema) error {
			if err := s.Create("payment_events", func(t *schema.Table) {
				t.ID()
				t.String("provider", 32)
				t.String("event_id", 191).Unique()
				t.String("event_type", 64)
				t.ForeignID("reservation_id").Nullable().Constrained("reservations").NullOnDelete()
				t.Text("payload")
				t.Timestamp("received_at")
				t.Timestamp("processed_at").Nullable()
				t.Timestamps()
			}); err != nil {
				return err
			}

			if err := s.Create("ledger_entries", func(t *schema.Table) {
				t.ID()
				t.ForeignID("user_id").Constrained("users")
				t.ForeignID("raffle_id").Nullable().Constrained("raffles")
				t.ForeignID("reservation_id").Nullable().Constrained("reservations")
				t.String("kind", 32)
				t.BigInteger("amount_cents")
				t.String("note", 255).Default("")
				t.ForeignID("created_by").Nullable().Constrained("users")
				t.Timestamps()
				t.Index("user_id", "kind")
				t.Index("raffle_id")
			}); err != nil {
				return err
			}

			if err := s.Create("payouts", func(t *schema.Table) {
				t.ID()
				t.ForeignID("user_id").Constrained("users")
				t.BigInteger("amount_cents")
				t.String("reference", 255)
				t.ForeignID("recorded_by").Constrained("users")
				t.Timestamps()
				t.Index("user_id")
			}); err != nil {
				return err
			}

			if s.Dialect() != "postgres" {
				return nil
			}

			for _, stmt := range []string{
				`ALTER TABLE ledger_entries ADD CONSTRAINT ledger_entries_kind_valid CHECK (kind IN ('sale_owner_share','platform_fee','payout','refund_due','refund'))`,
				`ALTER TABLE ledger_entries ADD CONSTRAINT ledger_entries_amount_nonzero CHECK (amount_cents <> 0)`,
				`ALTER TABLE payouts ADD CONSTRAINT payouts_amount_positive CHECK (amount_cents > 0)`,
				`CREATE FUNCTION ledger_entries_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	RAISE EXCEPTION 'ledger_entries is append-only';
END;
$$`,
				`CREATE TRIGGER ledger_entries_no_update_delete BEFORE UPDATE OR DELETE ON ledger_entries
	FOR EACH ROW EXECUTE FUNCTION ledger_entries_append_only()`,
			} {
				if err := s.Raw(stmt); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			if s.Dialect() == "postgres" {
				if err := s.Raw(`DROP TRIGGER IF EXISTS ledger_entries_no_update_delete ON ledger_entries`); err != nil {
					return err
				}

				if err := s.Raw(`DROP FUNCTION IF EXISTS ledger_entries_append_only()`); err != nil {
					return err
				}
			}

			for _, table := range []string{"payouts", "ledger_entries", "payment_events"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
