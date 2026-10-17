// Package ledger holds the money rules: the 90/10 split of each paid number
// and owner balances over the append-only ledger_entries (PLAN.md §4.4).
package ledger

import (
	"context"

	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// FeePercent is the platform fee on each paid number.
const FeePercent = 10

// Split divides a paid amount in centavos into the owner share and the
// platform fee; integer math, the fee rounds down.
func Split(amountCents int64) (ownerShare, fee int64) {
	fee = amountCents * FeePercent / 100
	return amountCents - fee, fee
}

// Balance is what the platform owes userID: owner shares minus payouts and
// refunds, in centavos.
func Balance(ctx context.Context, ex database.Executor, userID int64) (int64, error) {
	p := ex.Dialect().Placeholder

	var balance int64
	err := ex.QueryRowContext(ctx, "SELECT COALESCE(SUM(amount_cents), 0) FROM ledger_entries WHERE user_id = "+p(1)+
		" AND kind IN ('"+models.LedgerSaleOwnerShare+"','"+models.LedgerPayout+"','"+models.LedgerRefund+"')", userID).Scan(&balance)

	return balance, err
}
