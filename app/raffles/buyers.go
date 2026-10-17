package raffles

import (
	"context"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Buyer is the owner's view of who holds a number: a live payment link
// (pending or awaiting payment) or a paid number. Tokens and their hashes
// never leave the server.
type Buyer struct {
	Number    int          `json:"number"`
	Status    string       `json:"status"`
	Name      string       `json:"buyer_name"`
	Email     string       `json:"buyer_email"`
	Phone     string       `json:"buyer_phone"`
	ExpiresAt time.Time    `json:"expires_at"`
	PaidAt    orm.NullTime `json:"paid_at"`
}

// Buyers lists the live and paid reservations of raffleID by number. Only
// the raffle's owner may see them.
func (s *Service) Buyers(ctx context.Context, raffleID int64) ([]Buyer, error) {
	list, err := orm.QueryOf[models.Reservation](s.DB).Where("raffle_id", raffleID).
		WhereIn("status", models.ReservationPending, models.ReservationAwaitingPayment, models.ReservationPaid).
		OrderBy("number", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]Buyer, len(list))
	for i, r := range list {
		out[i] = Buyer{
			Number: r.Number, Status: r.Status, Name: r.BuyerName, Email: r.BuyerEmail, Phone: r.BuyerPhone,
			ExpiresAt: r.ExpiresAt, PaidAt: r.PaidAt,
		}
	}

	return out, nil
}
