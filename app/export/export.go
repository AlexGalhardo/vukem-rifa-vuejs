// Package export builds the owner's audit export of one raffle: summary,
// every number, every reservation (buyer data, amounts, charge), the
// ledger rows (late payments flagged as refund_due) and the draw, as JSON
// or CSV (PLAN.md Phase 7).
package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Service builds exports.
type Service struct {
	DB  *database.DB
	Now func() time.Time
}

// New returns a Service; now is the app clock.
func New(db *database.DB, now func() time.Time) *Service { return &Service{DB: db, Now: now} }

// Export is the whole audit document of one raffle. The validate tags
// only describe it in the OpenAPI document (keys always present).
type Export struct {
	GeneratedAt  time.Time     `json:"generated_at" validate:"required"`
	Raffle       Raffle        `json:"raffle" validate:"required"`
	Numbers      []Number      `json:"numbers"`
	Reservations []Reservation `json:"reservations"`
	Ledger       []LedgerEntry `json:"ledger"`
	Draw         *Draw         `json:"draw"`
}

// Raffle is the raffle summary with number counts and ledger totals in
// centavos.
type Raffle struct {
	ID               int64      `json:"id" validate:"required"`
	Slug             string     `json:"slug" validate:"required"`
	Title            string     `json:"title" validate:"required"`
	Description      string     `json:"description"`
	Status           string     `json:"status" validate:"required"`
	NumberCount      int        `json:"number_count" validate:"required"`
	PriceReais       int        `json:"price_reais" validate:"required"`
	PriceCents       int64      `json:"price_cents" validate:"required"`
	DurationDays     int        `json:"duration_days" validate:"required"`
	StartsAt         time.Time  `json:"starts_at" validate:"required"`
	EndsAt           time.Time  `json:"ends_at" validate:"required"`
	LockedAt         *time.Time `json:"locked_at"`
	ArchivedAt       *time.Time `json:"archived_at"`
	CreatedAt        time.Time  `json:"created_at" validate:"required"`
	Available        int        `json:"available" validate:"required"`
	Reserved         int        `json:"reserved" validate:"required"`
	Paid             int        `json:"paid" validate:"required"`
	OwnerShareCents  int64      `json:"owner_share_cents" validate:"required"`
	PlatformFeeCents int64      `json:"platform_fee_cents" validate:"required"`
	RefundDueCents   int64      `json:"refund_due_cents" validate:"required"`
}

// Number is one raffle number and its state.
type Number struct {
	Number        int        `json:"number" validate:"required"`
	Status        string     `json:"status" validate:"required"`
	PaidAt        *time.Time `json:"paid_at"`
	ReservationID *int64     `json:"reservation_id"`
}

// Reservation is one payment link and its buyer.
type Reservation struct {
	ID          int64      `json:"id" validate:"required"`
	Number      int        `json:"number" validate:"required"`
	BuyerName   string     `json:"buyer_name" validate:"required"`
	BuyerEmail  string     `json:"buyer_email" validate:"required"`
	BuyerPhone  string     `json:"buyer_phone" validate:"required"`
	Status      string     `json:"status" validate:"required"`
	AmountCents int64      `json:"amount_cents" validate:"required"`
	ChargeID    *string    `json:"charge_id"`
	ExpiresAt   time.Time  `json:"expires_at" validate:"required"`
	PaidAt      *time.Time `json:"paid_at"`
	CreatedAt   time.Time  `json:"created_at" validate:"required"`
	// RefundDue marks a payment confirmed after the link expired: the
	// platform owes the buyer a refund (PLAN.md §4.3).
	RefundDue bool `json:"refund_due" validate:"required"`
}

// LedgerEntry is one append-only money movement of the raffle.
type LedgerEntry struct {
	ID            int64     `json:"id" validate:"required"`
	Kind          string    `json:"kind" validate:"required"`
	AmountCents   int64     `json:"amount_cents" validate:"required"`
	ReservationID *int64    `json:"reservation_id"`
	Note          string    `json:"note"`
	RefundDue     bool      `json:"refund_due" validate:"required"`
	CreatedAt     time.Time `json:"created_at" validate:"required"`
}

// Draw is the auditable draw result; WinningNumber is nil when nothing was
// paid.
type Draw struct {
	DrawnAt            time.Time `json:"drawn_at" validate:"required"`
	Trigger            string    `json:"trigger" validate:"required"`
	Algorithm          string    `json:"algorithm" validate:"required"`
	Seed               string    `json:"seed" validate:"required"`
	ParticipantsCount  int       `json:"participants_count" validate:"required"`
	ParticipantsSHA256 string    `json:"participants_sha256" validate:"required"`
	WinningNumber      *int      `json:"winning_number"`
}

// Build loads the export of r. The caller has already checked that the
// signed-in owner owns r; ledger rows are additionally scoped by owner.
func (s *Service) Build(ctx context.Context, r models.Raffle) (Export, error) {
	out := Export{GeneratedAt: s.Now().UTC(), Raffle: raffle(r)}

	numbers, err := orm.QueryOf[models.RaffleNumber](s.DB).Where("raffle_id", r.ID).OrderBy("number", orm.Asc).Get(ctx)
	if err != nil {
		return out, fmt.Errorf("export numbers: %w", err)
	}

	out.Numbers = make([]Number, 0, len(numbers))

	for _, n := range numbers {
		out.Numbers = append(out.Numbers, Number{
			Number: n.Number, Status: n.Status, PaidAt: ptrTime(n.PaidAt), ReservationID: n.ReservationID,
		})

		switch n.Status {
		case models.NumberAvailable:
			out.Raffle.Available++
		case models.NumberReserved:
			out.Raffle.Reserved++
		case models.NumberPaid:
			out.Raffle.Paid++
		}
	}

	ledger, err := orm.QueryOf[models.LedgerEntry](s.DB).Where("raffle_id", r.ID).Where("user_id", r.UserID).
		OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return out, fmt.Errorf("export ledger: %w", err)
	}

	refundDue := map[int64]bool{}
	out.Ledger = make([]LedgerEntry, 0, len(ledger))

	for _, e := range ledger {
		due := e.Kind == models.LedgerRefundDue
		if due && e.ReservationID != nil {
			refundDue[*e.ReservationID] = true
		}

		out.Ledger = append(out.Ledger, LedgerEntry{
			ID: e.ID, Kind: e.Kind, AmountCents: e.AmountCents, ReservationID: e.ReservationID,
			Note: e.Note, RefundDue: due, CreatedAt: e.CreatedAt.UTC(),
		})

		switch e.Kind {
		case models.LedgerSaleOwnerShare:
			out.Raffle.OwnerShareCents += e.AmountCents
		case models.LedgerPlatformFee:
			out.Raffle.PlatformFeeCents += e.AmountCents
		case models.LedgerRefundDue:
			out.Raffle.RefundDueCents += e.AmountCents
		}
	}

	reservations, err := orm.QueryOf[models.Reservation](s.DB).Where("raffle_id", r.ID).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return out, fmt.Errorf("export reservations: %w", err)
	}

	out.Reservations = make([]Reservation, 0, len(reservations))

	for _, res := range reservations {
		out.Reservations = append(out.Reservations, Reservation{
			ID: res.ID, Number: res.Number, BuyerName: res.BuyerName, BuyerEmail: res.BuyerEmail,
			BuyerPhone: res.BuyerPhone, Status: res.Status, AmountCents: int64(res.AmountCents),
			ChargeID: res.ChargeID, ExpiresAt: res.ExpiresAt.UTC(), PaidAt: ptrTime(res.PaidAt),
			CreatedAt: res.CreatedAt.UTC(), RefundDue: refundDue[res.ID],
		})
	}

	d, err := orm.QueryOf[models.Draw](s.DB).Where("raffle_id", r.ID).First(ctx)

	switch {
	case errors.Is(err, orm.ErrNotFound):
	case err != nil:
		return out, fmt.Errorf("export draw: %w", err)
	default:
		out.Draw = &Draw{
			DrawnAt: d.DrawnAt.UTC(), Trigger: d.Trigger, Algorithm: d.Algorithm, Seed: d.Seed,
			ParticipantsCount: d.ParticipantsCount, ParticipantsSHA256: d.ParticipantsSHA256, WinningNumber: d.WinningNumber,
		}
	}

	return out, nil
}

func raffle(r models.Raffle) Raffle {
	return Raffle{
		ID: r.ID, Slug: r.Slug, Title: r.Title, Description: r.Description, Status: r.Status,
		NumberCount: r.NumberCount, PriceReais: r.PriceReais, PriceCents: int64(r.PriceReais) * 100,
		DurationDays: r.DurationDays, StartsAt: r.StartsAt.UTC(), EndsAt: r.EndsAt.UTC(),
		LockedAt: ptrTime(r.LockedAt), ArchivedAt: ptrTime(r.ArchivedAt), CreatedAt: r.CreatedAt.UTC(),
	}
}

func ptrTime(t orm.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}

	v := t.V.UTC()

	return &v
}

// Filename is the download name of the export, e.g. "rifa-abc123-export.csv".
func Filename(e Export, ext string) string { return "rifa-" + e.Raffle.Slug + "-export." + ext }

// WriteJSON writes e as indented JSON.
func WriteJSON(w io.Writer, e Export) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(e)
}

// WriteCSV writes e as CSV sections separated by a blank line. The first
// column of every row is the record type and each section starts with its
// header row, so one file holds the whole export and still opens in a
// spreadsheet. Text cells are guarded against formula injection (Text);
// numbers and timestamps are written by the app and never start a formula.
func WriteCSV(w io.Writer, e Export) error {
	cw := csv.NewWriter(w)
	r := e.Raffle

	rows := [][]string{
		{
			"record", "id", "slug", "title", "description", "status", "number_count", "price_cents", "duration_days",
			"starts_at", "ends_at", "locked_at", "archived_at", "available", "reserved", "paid",
			"owner_share_cents", "platform_fee_cents", "refund_due_cents", "generated_at",
		},
		{
			"raffle", itoa(r.ID), Text(r.Slug), Text(r.Title), Text(r.Description), Text(r.Status),
			itoa(r.NumberCount), itoa(r.PriceCents), itoa(r.DurationDays), ts(r.StartsAt), ts(r.EndsAt),
			tsPtr(r.LockedAt), tsPtr(r.ArchivedAt), itoa(r.Available), itoa(r.Reserved), itoa(r.Paid),
			itoa(r.OwnerShareCents), itoa(r.PlatformFeeCents), itoa(r.RefundDueCents), ts(e.GeneratedAt),
		},
		{},
		{"record", "number", "status", "paid_at", "reservation_id"},
	}

	for _, n := range e.Numbers {
		rows = append(rows, []string{"number", itoa(n.Number), Text(n.Status), tsPtr(n.PaidAt), idPtr(n.ReservationID)})
	}

	rows = append(rows, []string{}, []string{
		"record", "id", "number", "buyer_name", "buyer_email", "buyer_phone",
		"status", "amount_cents", "charge_id", "expires_at", "paid_at", "created_at", "refund_due",
	})

	for _, res := range e.Reservations {
		charge := ""
		if res.ChargeID != nil {
			charge = *res.ChargeID
		}

		rows = append(rows, []string{
			"reservation", itoa(res.ID), itoa(res.Number), Text(res.BuyerName),
			Text(res.BuyerEmail), Text(res.BuyerPhone), Text(res.Status), itoa(res.AmountCents), Text(charge),
			ts(res.ExpiresAt), tsPtr(res.PaidAt), ts(res.CreatedAt), strconv.FormatBool(res.RefundDue),
		})
	}

	rows = append(rows, []string{}, []string{
		"record", "id", "kind", "amount_cents", "reservation_id", "note",
		"refund_due", "created_at",
	})

	for _, l := range e.Ledger {
		rows = append(rows, []string{
			"ledger", itoa(l.ID), Text(l.Kind), itoa(l.AmountCents), idPtr(l.ReservationID),
			Text(l.Note), strconv.FormatBool(l.RefundDue), ts(l.CreatedAt),
		})
	}

	rows = append(rows, []string{}, []string{
		"record", "drawn_at", "trigger", "algorithm", "seed",
		"participants_count", "participants_sha256", "winning_number",
	})

	if d := e.Draw; d != nil {
		winner := ""
		if d.WinningNumber != nil {
			winner = itoa(*d.WinningNumber)
		}

		rows = append(rows, []string{
			"draw", ts(d.DrawnAt), Text(d.Trigger), Text(d.Algorithm), Text(d.Seed),
			itoa(d.ParticipantsCount), Text(d.ParticipantsSHA256), winner,
		})
	}

	for _, row := range rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	cw.Flush()

	return cw.Error()
}

// Text guards a CSV text cell against formula injection (OWASP CSV
// injection): a cell starting with =, +, -, @, tab or carriage return gets
// a leading apostrophe so spreadsheets show it as text.
func Text(s string) string {
	if s == "" {
		return s
	}

	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}

	return s
}

func itoa[N int | int64](n N) string { return strconv.FormatInt(int64(n), 10) }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func tsPtr(t *time.Time) string {
	if t == nil {
		return ""
	}

	return ts(*t)
}

func idPtr(id *int64) string {
	if id == nil {
		return ""
	}

	return itoa(*id)
}
