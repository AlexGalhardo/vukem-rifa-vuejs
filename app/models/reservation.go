package models

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// Reservation origins: who generated the payment link.
const (
	OriginOwner  = "owner"
	OriginPublic = "public"
	OriginAPI    = "api"
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

	RaffleID     int64                 `json:"-"`
	NumberID     int64                 `json:"-"`
	Number       int                   `json:"number"`
	BuyerName    string                `json:"buyer_name"`
	BuyerEmail   string                `json:"buyer_email"`
	BuyerPhone   string                `json:"buyer_phone"`
	TokenHash    string                `json:"-"`
	TokenSealed  orm.Encrypted[string] `db:"token_sealed" json:"-"`
	Status       string                `json:"status"`
	ExpiresAt    time.Time             `json:"expires_at"`
	AmountCents  int                   `json:"amount_cents"`
	ChargeID     *string               `json:"-"`
	BrCode       *string               `json:"-"`
	BrCodeBase64 *string               `db:"br_code_base64" json:"-"`
	PaidAt       orm.NullTime          `json:"paid_at"`
	// Locale is the buyer's page locale, set on confirm ("" = APP_LOCALE).
	Locale string `json:"-"`
	// Origin is who generated the link (OriginOwner, OriginPublic, OriginAPI).
	Origin string `json:"origin"`
	// BuyerUserID links the reservation to the buyer's account (purchase
	// history): the signed-in buyer, or a verified account with the email.
	BuyerUserID *int64 `json:"-"`
	// ConsentAt and ConsentVersion record the LGPD notice a public buyer
	// accepted (nil for owner/API links).
	ConsentAt      orm.NullTime `json:"-"`
	ConsentVersion *string      `json:"-"`
}

// Live reports whether the reservation still holds its number at now.
func (r Reservation) Live(now time.Time) bool {
	return (r.Status == ReservationPending || r.Status == ReservationAwaitingPayment) && now.Before(r.ExpiresAt)
}
