package models

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// Ledger entry kinds (PLAN.md §4.4). Amounts are signed centavos: sales and
// fees are positive, payouts and refunds negative.
const (
	LedgerSaleOwnerShare = "sale_owner_share"
	LedgerPlatformFee    = "platform_fee"
	LedgerPayout         = "payout"
	LedgerRefundDue      = "refund_due"
	LedgerRefund         = "refund"
)

// PaymentEvent is one received webhook delivery, stored before handling.
type PaymentEvent struct {
	orm.Model

	Provider      string       `json:"provider"`
	EventID       string       `json:"event_id"`
	EventType     string       `json:"event_type"`
	ReservationID *int64       `json:"-"`
	Payload       string       `json:"-"`
	ReceivedAt    time.Time    `json:"received_at"`
	ProcessedAt   orm.NullTime `json:"processed_at"`
}

// LedgerEntry is an append-only money movement for an owner.
type LedgerEntry struct {
	orm.Model

	UserID        int64  `json:"-"`
	RaffleID      *int64 `json:"raffle_id"`
	ReservationID *int64 `json:"reservation_id"`
	Kind          string `json:"kind"`
	AmountCents   int64  `json:"amount_cents"`
	Note          string `json:"note"`
	CreatedBy     *int64 `json:"-"`
}

// Payout is a manual transfer to an owner recorded by an admin.
type Payout struct {
	orm.Model

	UserID      int64  `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Reference   string `json:"reference"`
	RecordedBy  int64  `json:"recorded_by"`
}
