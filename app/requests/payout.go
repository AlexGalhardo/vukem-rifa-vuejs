package requests

import "github.com/AlexGalhardo/vukem/z"

// MaxPayoutCents bounds one payout (R$ 1.000.000,00).
const MaxPayoutCents = 100_000_000

// Payout is a payout an admin records after paying an owner outside the
// app; the amount is in centavos and is checked against the balance.
type Payout struct {
	UserID      int64  `json:"user_id"      form:"user_id"`
	AmountCents int64  `json:"amount_cents" form:"amount_cents"`
	Reference   string `json:"reference"    form:"reference"`
}

// PayoutSchema validates Payout.
var PayoutSchema = z.Struct[Payout](z.Shape{
	"user_id":      z.Int64().Positive(),
	"amount_cents": z.Int64().Positive().Max(MaxPayoutCents),
	"reference":    z.String().Trim().Min(1).Max(255),
})
