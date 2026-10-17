package models

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// Reservation statuses.
const (
	ReservationPending         = "pending"          // link generated, buyer has not confirmed
	ReservationAwaitingPayment = "awaiting_payment" // PIX charge created
	ReservationPaid            = "paid"
	ReservationExpired         = "expired"
	ReservationCancelled       = "cancelled"
)

// ReservationTTL is how long a payment link holds its number.
const ReservationTTL = 10 * time.Minute

// Reservation is a payment link holding one number for a buyer.
type Reservation struct {
	orm.Model

	RaffleID     int64        `json:"-"`
	NumberID     int64        `json:"-"`
	Number       int          `json:"number"`
	BuyerName    string       `json:"buyer_name"`
	BuyerEmail   string       `json:"buyer_email"`
	BuyerPhone   string       `json:"buyer_phone"`
	TokenHash    string       `json:"-"`
	Status       string       `json:"status"`
	ExpiresAt    time.Time    `json:"expires_at"`
	AmountCents  int          `json:"amount_cents"`
	ChargeID     *string      `json:"-"`
	BrCode       *string      `json:"-"`
	BrCodeBase64 *string      `db:"br_code_base64" json:"-"`
	PaidAt       orm.NullTime `json:"paid_at"`
}

// Live reports whether the reservation still holds its number at now.
func (r Reservation) Live(now time.Time) bool {
	return (r.Status == ReservationPending || r.Status == ReservationAwaitingPayment) && now.Before(r.ExpiresAt)
}
