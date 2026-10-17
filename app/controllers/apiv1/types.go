// Package apiv1 is the REST API v1 (PLAN.md §5): owner-scoped raffles,
// numbers, payment links, exports and balance, authenticated by API keys
// (vukem auth.Tokens). Input and output types carry `validate` tags: vukem
// z validates them (input on every request, output in development and
// tests) and vukem openapi documents them.
package apiv1

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// ListRaffles is the query of GET /raffles.
type ListRaffles struct {
	Page    int `json:"page"     validate:"min=1"         doc:"Page number, from 1 (default 1)"`
	PerPage int `json:"per_page" validate:"min=1,max=100" doc:"Items per page (default 20, max 100)"`
}

// RaffleID addresses one raffle.
type RaffleID struct {
	ID int64 `json:"id" validate:"required,min=1"`
}

// CreateRaffle is the body of POST /raffles; the rules mirror
// requests.RaffleSchema, which also runs (a test keeps both in sync).
type CreateRaffle struct {
	Title        string `json:"title"         validate:"required,min=3,max=120"`
	Description  string `json:"description"   validate:"max=5000"`
	NumberCount  int    `json:"number_count"  validate:"required,min=10,max=1000"`
	PriceReais   int    `json:"price_reais"   validate:"required,min=1,max=100000" doc:"Price per number in whole reais"`
	DurationDays int    `json:"duration_days" validate:"required,min=7,max=60"`
}

// UpdateRaffle is the body of PATCH /raffles/{id}: only the fields sent
// change. Number count, price and duration are locked after the first paid
// number.
type UpdateRaffle struct {
	ID           int64   `json:"id"            validate:"required,min=1"`
	Title        *string `json:"title"         validate:"min=3,max=120"`
	Description  *string `json:"description"   validate:"max=5000"`
	NumberCount  *int    `json:"number_count"  validate:"min=10,max=1000"`
	PriceReais   *int    `json:"price_reais"   validate:"min=1,max=100000"`
	DurationDays *int    `json:"duration_days" validate:"min=7,max=60"`
}

// CreatePaymentLink is the body of POST /raffles/{id}/numbers/{number}/payment-links:
// the buyer the number is reserved for. The phone is normalized to E.164
// by requests.BuyerSchema.
type CreatePaymentLink struct {
	ID         int64  `json:"id"          validate:"required,min=1"`
	Number     int    `json:"number"      validate:"required,min=1"`
	BuyerName  string `json:"buyer_name"  validate:"required,min=2,max=100"`
	BuyerEmail string `json:"buyer_email" validate:"required,email,max=255"`
	BuyerPhone string `json:"buyer_phone" validate:"required,max=32" doc:"Brazilian phone, e.g. (11) 98765-4321"`
}

// ExportQuery is the query of GET /raffles/{id}/export.
type ExportQuery struct {
	ID     int64  `json:"id"     validate:"required,min=1"`
	Format string `json:"format" validate:"oneof=json csv" doc:"json (default) or csv"`
}

// Raffle is a raffle as the API returns it.
type Raffle struct {
	ID           int64      `json:"id"            validate:"required"`
	Slug         string     `json:"slug"          validate:"required"`
	Title        string     `json:"title"         validate:"required"`
	Description  string     `json:"description"`
	Status       string     `json:"status"        validate:"required,oneof=active drawn closed archived"`
	NumberCount  int        `json:"number_count"  validate:"required"`
	PriceReais   int        `json:"price_reais"   validate:"required"`
	DurationDays int        `json:"duration_days" validate:"required"`
	StartsAt     time.Time  `json:"starts_at"     validate:"required"`
	EndsAt       time.Time  `json:"ends_at"       validate:"required"`
	Locked       bool       `json:"locked"        validate:"required"`
	LockedAt     *time.Time `json:"locked_at"`
	ArchivedAt   *time.Time `json:"archived_at"`
	PublicURL    string     `json:"public_url"    validate:"required"`
	Available    int        `json:"available"     validate:"required"`
	Reserved     int        `json:"reserved"      validate:"required"`
	Paid         int        `json:"paid"          validate:"required"`
	CreatedAt    time.Time  `json:"created_at"    validate:"required"`
}

// RafflePage is one page of raffles (data may be empty: z "required"
// means present since vukem v0.25.0).
type RafflePage struct {
	Data  []Raffle  `json:"data"  validate:"required"`
	Meta  PageMeta  `json:"meta"  validate:"required"`
	Links PageLinks `json:"links" validate:"required"`
}

// PageMeta describes a page.
type PageMeta struct {
	Page     int   `json:"page"      validate:"required"`
	PerPage  int   `json:"per_page"  validate:"required"`
	Total    int64 `json:"total"     validate:"required"`
	LastPage int   `json:"last_page" validate:"required"`
}

// PageLinks point to the neighbour pages (null at the ends).
type PageLinks struct {
	Self string  `json:"self" validate:"required"`
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

// RaffleNumber is one raffle number.
type RaffleNumber struct {
	Number int        `json:"number"  validate:"required"`
	Status string     `json:"status"  validate:"required,oneof=available reserved paid"`
	PaidAt *time.Time `json:"paid_at"`
}

// NumberList is every number of a raffle, in order.
type NumberList struct {
	Data []RaffleNumber `json:"data" validate:"required"`
}

// PaymentLink is a new payment link: the buyer pays at URL until
// ExpiresAt (10 minutes).
type PaymentLink struct {
	URL       string    `json:"url"        validate:"required,url"`
	ExpiresAt time.Time `json:"expires_at" validate:"required"`
	Number    int       `json:"number"     validate:"required"`
}

// Balance is what the platform owes the owner, in centavos.
type Balance struct {
	BalanceCents int64  `json:"balance_cents" validate:"required"`
	Currency     string `json:"currency"      validate:"required,oneof=BRL"`
}

func toRaffle(r models.Raffle, counts map[string]int, baseURL string) Raffle {
	return Raffle{
		ID: r.ID, Slug: r.Slug, Title: r.Title, Description: r.Description, Status: r.Status,
		NumberCount: r.NumberCount, PriceReais: r.PriceReais, DurationDays: r.DurationDays,
		StartsAt: r.StartsAt.UTC(), EndsAt: r.EndsAt.UTC(), Locked: r.Locked(),
		LockedAt: ptrTime(r.LockedAt), ArchivedAt: ptrTime(r.ArchivedAt),
		PublicURL: baseURL + "/r/" + r.Slug,
		Available: counts[models.NumberAvailable], Reserved: counts[models.NumberReserved], Paid: counts[models.NumberPaid],
		CreatedAt: r.CreatedAt.UTC(),
	}
}

func ptrTime(t orm.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}

	v := t.V.UTC()

	return &v
}
