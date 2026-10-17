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

// brPhone is a Brazilian number in E.164: +55, a two-digit area code
// (11..99) and 8 (landline) or 9 (mobile, starting with 9) digits.
var brPhone = regexp.MustCompile(`^\+55[1-9][1-9](9\d{8}|[2-8]\d{7})$`)

// NormalizePhone turns the usual ways to write a Brazilian number
// ("(11) 98765-4321", "11987654321", "+55 11 98765 4321") into E.164
// ("+5511987654321"); anything else keeps its digits and fails brPhone.
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
	"buyer_phone": z.String().Preprocess(NormalizePhone).Regex(brPhone),
})

// PublicBuyer is the buyer form on the public raffle page: the Buyer fields,
// the LGPD consent and a honeypot field that people never see.
type PublicBuyer struct {
	Name    string `json:"buyer_name"  form:"buyer_name"`
	Email   string `json:"buyer_email" form:"buyer_email"`
	Phone   string `json:"buyer_phone" form:"buyer_phone"`
	Consent bool   `json:"consent"     form:"consent"`
	Website string `json:"website"     form:"website"`
}

// Buyer returns the buyer data of the form.
func (b PublicBuyer) Buyer() Buyer { return Buyer{Name: b.Name, Email: b.Email, Phone: b.Phone} }

// PublicBuyerSchema validates PublicBuyer; consent must be given.
var PublicBuyerSchema = z.Struct[PublicBuyer](z.Shape{
	"buyer_name":  z.String().Trim().Min(2).Max(100),
	"buyer_email": z.String().Trim().ToLower().Max(255).Email(),
	"buyer_phone": z.String().Preprocess(NormalizePhone).Regex(brPhone),
	"consent":     z.Bool(),
	"website":     z.String().Max(200).Optional(),
}).Refine(func(b PublicBuyer) error {
	if !b.Consent {
		return z.FieldErrorKey("consent", "consent_required", "Accept the privacy notice to continue.", nil)
	}

	return nil
})
