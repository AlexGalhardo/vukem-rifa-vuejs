package models

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// Raffle statuses.
const (
	RaffleActive   = "active"
	RaffleDrawn    = "drawn"
	RaffleClosed   = "closed" // deadline passed with no paid number
	RaffleArchived = "archived"
)

// Number statuses (state machine in PLAN.md §4.3).
const (
	NumberAvailable = "available"
	NumberReserved  = "reserved"
	NumberPaid      = "paid"
)

// Raffle is an owner's raffle with numbers 1..NumberCount.
type Raffle struct {
	orm.Model

	UserID       int64        `json:"-"`
	Slug         string       `json:"slug"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	NumberCount  int          `json:"number_count"`
	PriceReais   int          `json:"price_reais"`
	DurationDays int          `json:"duration_days"`
	StartsAt     time.Time    `json:"starts_at"`
	EndsAt       time.Time    `json:"ends_at"`
	Status       string       `json:"status"`
	LockedAt     orm.NullTime `json:"locked_at"`
	ArchivedAt   orm.NullTime `json:"archived_at"`
}

// Locked reports whether a number was paid: price, number count and
// duration can no longer change.
func (r Raffle) Locked() bool { return r.LockedAt.Valid }

// RaffleNumber is one number of a raffle.
type RaffleNumber struct {
	orm.Model

	RaffleID int64        `json:"-"`
	Number   int          `json:"number"`
	Status   string       `json:"status"`
	PaidAt   orm.NullTime `json:"paid_at"`
}
