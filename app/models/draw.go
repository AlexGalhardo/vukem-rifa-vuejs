package models

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// Draw triggers.
const (
	DrawDeadline = "deadline"
	DrawSoldOut  = "sold_out"
)

// Draw is the auditable result of a raffle (PLAN.md D12). WinningNumber is
// nil when no number was paid.
type Draw struct {
	orm.Model

	RaffleID           int64     `json:"-"`
	DrawnAt            time.Time `json:"drawn_at"`
	Seed               string    `json:"seed"`
	ParticipantsCount  int       `json:"participants_count"`
	ParticipantsSHA256 string    `db:"participants_sha256" json:"participants_sha256"`
	WinningNumber      *int      `json:"winning_number"`
	Algorithm          string    `json:"algorithm"`
	Trigger            string    `json:"trigger"`
}
