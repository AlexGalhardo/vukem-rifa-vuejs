package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// RefundDue is a late or mismatched payment the platform must return.
type RefundDue struct {
	ID          int64      `json:"id"`
	RaffleTitle string     `json:"raffle_title"`
	Number      int        `json:"number"`
	BuyerEmail  string     `json:"buyer_email"`
	AmountCents int64      `json:"amount_cents"`
	Note        string     `json:"note"`
	CreatedAt   time.Time  `json:"created_at"`
	RefundedAt  *time.Time `json:"refunded_at"`
	Reference   string     `json:"reference"`
}

// RefundsDue lists refund_due entries, pending ones first, newest first.
func RefundsDue(ctx context.Context, ex database.Executor) (pending, done []RefundDue, err error) {
	rows, err := ex.QueryContext(ctx, `SELECT l.id, COALESCE(r.title, ''), COALESCE(res.number, 0), COALESCE(res.buyer_email, ''),
		l.amount_cents, l.note, l.created_at, f.created_at, COALESCE(f.reference, '')
		FROM ledger_entries l
		LEFT JOIN raffles r ON r.id = l.raffle_id
		LEFT JOIN reservations res ON res.id = l.reservation_id
		LEFT JOIN refunds f ON f.ledger_entry_id = l.id
		WHERE l.kind = '`+models.LedgerRefundDue+`'
		ORDER BY l.created_at DESC`)
	if err != nil {
		return nil, nil, fmt.Errorf("list refunds due: %w", err)
	}
	defer rows.Close()

	pending, done = []RefundDue{}, []RefundDue{}

	for rows.Next() {
		var (
			d        RefundDue
			refunded sql.NullTime
		)

		if err := rows.Scan(&d.ID, &d.RaffleTitle, &d.Number, &d.BuyerEmail, &d.AmountCents, &d.Note, &d.CreatedAt, &refunded, &d.Reference); err != nil {
			return nil, nil, err
		}

		if refunded.Valid {
			t := refunded.Time
			d.RefundedAt = &t
			done = append(done, d)
		} else {
			pending = append(pending, d)
		}
	}

	return pending, done, rows.Err()
}

// MarkRefunded records that the admin returned refund_due entry id outside
// the app. Unknown entries are 404; an entry already refunded is 409.
func MarkRefunded(ctx context.Context, db *database.DB, id int64, reference string, adminID int64, now time.Time) error {
	return db.Tx(ctx, func(tx *database.Tx) error {
		p := tx.Dialect().Placeholder

		var kind string
		if err := tx.QueryRowContext(ctx, "SELECT kind FROM ledger_entries WHERE id = "+p(1), id).Scan(&kind); err != nil || kind != models.LedgerRefundDue {
			return errs.New(http.StatusNotFound, i18n.T(ctx, "admin.refunds.not_found"))
		}

		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM refunds WHERE ledger_entry_id = "+p(1), id).Scan(&n); err != nil {
			return err
		}

		if n > 0 {
			return errs.New(http.StatusConflict, i18n.T(ctx, "admin.refunds.already"))
		}

		// The unique ledger_entry_id index settles a concurrent double submit.
		if _, err := tx.ExecContext(ctx, "INSERT INTO refunds (ledger_entry_id, reference, recorded_by, created_at, updated_at) VALUES ("+
			p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", "+p(5)+")", id, reference, adminID, now.UTC(), now.UTC()); err != nil {
			return errs.New(http.StatusConflict, i18n.T(ctx, "admin.refunds.already"))
		}

		return nil
	})
}

// PendingRefunds counts refund_due entries not yet marked refunded and the
// age of the oldest, for the hourly warning.
func PendingRefunds(ctx context.Context, ex database.Executor, now time.Time) (count int, oldest time.Duration, err error) {
	const pending = ` FROM ledger_entries l LEFT JOIN refunds f ON f.ledger_entry_id = l.id
		WHERE l.kind = '` + models.LedgerRefundDue + `' AND f.id IS NULL`

	if err := ex.QueryRowContext(ctx, "SELECT COUNT(*)"+pending).Scan(&count); err != nil {
		return 0, 0, fmt.Errorf("count pending refunds: %w", err)
	}

	if count == 0 {
		return 0, 0, nil
	}

	// No MIN(): SQLite returns aggregated timestamps as text.
	var first time.Time
	if err := ex.QueryRowContext(ctx, "SELECT l.created_at"+pending+" ORDER BY l.created_at LIMIT 1").Scan(&first); err != nil {
		return 0, 0, fmt.Errorf("oldest pending refund: %w", err)
	}

	return count, now.Sub(first), nil
}
