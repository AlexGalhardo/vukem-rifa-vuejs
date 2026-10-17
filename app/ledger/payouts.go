package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// OwnerBalance is one owner's balance for the admin payouts page.
type OwnerBalance struct {
	UserID       int64  `json:"user_id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	BalanceCents int64  `json:"balance_cents"`
}

// Balances lists owners with a non-zero balance, largest first.
func Balances(ctx context.Context, ex database.Executor) ([]OwnerBalance, error) {
	rows, err := ex.QueryContext(ctx, "SELECT u.id, u.name, u.email, SUM(l.amount_cents) AS balance"+
		" FROM ledger_entries l JOIN users u ON u.id = l.user_id"+
		" WHERE l.kind IN ('"+models.LedgerSaleOwnerShare+"','"+models.LedgerPayout+"','"+models.LedgerRefund+"')"+
		" GROUP BY u.id, u.name, u.email HAVING SUM(l.amount_cents) <> 0 ORDER BY balance DESC, u.id")
	if err != nil {
		return nil, fmt.Errorf("owner balances: %w", err)
	}
	defer rows.Close()

	out := []OwnerBalance{}

	for rows.Next() {
		var b OwnerBalance
		if err := rows.Scan(&b.UserID, &b.Name, &b.Email, &b.BalanceCents); err != nil {
			return nil, err
		}

		out = append(out, b)
	}

	return out, rows.Err()
}

// RecordPayout writes the payout and its negative ledger entry in one
// transaction. The owner's row is locked first (Postgres) so two admins
// (or a double submit) cannot both pay out the same balance. An amount
// above the balance is a 422 on amount_cents; an unknown owner a 404.
func RecordPayout(ctx context.Context, db *database.DB, in requests.Payout, adminID int64) (models.Payout, error) {
	var payout models.Payout

	err := db.Tx(ctx, func(tx *database.Tx) error {
		p := tx.Dialect().Placeholder

		lock := ""
		if tx.Dialect().Name() == "postgres" {
			lock = " FOR UPDATE"
		}

		var id int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE id = "+p(1)+lock, in.UserID).Scan(&id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errs.New(http.StatusNotFound, i18n.T(ctx, "admin.payouts.owner_not_found"))
			}

			return fmt.Errorf("lock owner %d: %w", in.UserID, err)
		}

		balance, err := Balance(ctx, tx, in.UserID)
		if err != nil {
			return err
		}

		if in.AmountCents > balance {
			return errs.NewValidation().Add("amount_cents", i18n.T(ctx, "admin.payouts.exceeds_balance"))
		}

		payout = models.Payout{UserID: in.UserID, AmountCents: in.AmountCents, Reference: in.Reference, RecordedBy: adminID}
		if err := orm.Create(ctx, tx, &payout); err != nil {
			return fmt.Errorf("create payout: %w", err)
		}

		entry := models.LedgerEntry{
			UserID: in.UserID, Kind: models.LedgerPayout, AmountCents: -in.AmountCents,
			Note: truncate(fmt.Sprintf("payout %d: %s", payout.ID, in.Reference), 255), CreatedBy: &adminID,
		}

		return orm.Create(ctx, tx, &entry)
	})

	return payout, err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n]
}
