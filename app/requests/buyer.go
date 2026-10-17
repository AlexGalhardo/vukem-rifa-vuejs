package requests

import (
	"regexp"
	"strings"

	"github.com/AlexGalhardo/vukem/z"
)

// Buyer is the data the owner fills in for one number before generating its
// payment link.
type Buyer struct {
	Name  string `json:"buyer_name"  form:"buyer_name"`
	Email string `json:"buyer_email" form:"buyer_email"`
	Phone string `json:"buyer_phone" form:"buyer_phone"`
}

// brPhone accepts the usual ways to write a Brazilian number: optional +55,
// a two-digit area code (11..99, optional parentheses) and 8 (landline) or
// 9 (mobile, starting with 9) digits, with optional spaces, dots or dashes.
//
// workaround(vukem v0.22.0): z Transform runs after the checks, so the raw input is validated and then normalized.
var brPhone = regexp.MustCompile(`^(\+?55[\s.-]?)?\(?[1-9][1-9]\)?[\s.-]?(9\d{4}|[2-8]\d{3})[\s.-]?\d{4}$`)

// NormalizePhone turns a number accepted by brPhone ("(11) 98765-4321",
// "11987654321", "+55 11 98765 4321") into E.164 ("+5511987654321").
func NormalizePhone(s string) string {
	var b strings.Builder

	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	d := b.String()
	if len(d) == 10 || len(d) == 11 {
		d = "55" + d
	}

	return "+" + d
}

// BuyerSchema validates Buyer and normalizes the phone to E.164.
var BuyerSchema = z.Struct[Buyer](z.Shape{
	"buyer_name":  z.String().Trim().Min(2).Max(100),
	"buyer_email": z.String().Trim().ToLower().Max(255).Email(),
	"buyer_phone": z.String().Trim().Regex(brPhone).Transform(NormalizePhone),
})
