package requests

import "github.com/AlexGalhardo/vukem/z"

// Raffle limits (owner decisions, PLAN.md §2).
const (
	MinNumbers  = 10
	MaxNumbers  = 1000
	MinPrice    = 1 // AbacatePay minimum charge is R$ 1,00
	MaxPrice    = 100000
	MinDuration = 7
	MaxDuration = 60
)

// Raffle is the create/update form; prices are whole reais.
type Raffle struct {
	Title        string `json:"title"         form:"title"`
	Description  string `json:"description"   form:"description"`
	NumberCount  int    `json:"number_count"  form:"number_count"`
	PriceReais   int    `json:"price_reais"   form:"price_reais"`
	DurationDays int    `json:"duration_days" form:"duration_days"`
}

// RaffleSchema validates Raffle.
var RaffleSchema = z.Struct[Raffle](z.Shape{
	"title":         z.String().Trim().Min(3).Max(120),
	"description":   z.String().Trim().Max(5000).Optional(),
	"number_count":  z.Int().Min(MinNumbers).Max(MaxNumbers),
	"price_reais":   z.Int().Min(MinPrice).Max(MaxPrice),
	"duration_days": z.Int().Min(MinDuration).Max(MaxDuration),
})
