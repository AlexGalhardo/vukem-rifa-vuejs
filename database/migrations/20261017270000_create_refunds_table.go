package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Refunds an admin returned outside the app (AbacatePay dashboard) for
// refund_due ledger entries; one per entry. The append-only ledger is not
// touched: a late payment never credited the owner.
func m20261017270000CreateRefundsTable() schema.Migration {
	return schema.Migration{
		ID: "20261017270000_create_refunds_table",
		Up: func(s *schema.Schema) error {
			return s.Create("refunds", func(t *schema.Table) {
				t.ID()
				t.ForeignID("ledger_entry_id").Unique().Constrained("ledger_entries")
				t.String("reference", 255)
				t.ForeignID("recorded_by").Constrained("users")
				t.Timestamps()
			})
		},
		Down: func(s *schema.Schema) error { return s.DropIfExists("refunds") },
	}
}
