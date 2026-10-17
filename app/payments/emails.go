package payments

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/i18n"
	vmail "github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/mail"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// MailLater queues a rendered mail (vukem App.MailLater).
type MailLater func(ctx context.Context, m vmail.Mailable) error

// SendEmails queues the buyer's "payment confirmed" mail (in the locale of
// the payment page) and the owner's "number sold" mail (in defaultLocale:
// owners have no stored locale). ctx must carry the i18n bundle.
func SendEmails(ctx context.Context, s *Service, reservationID int64, appURL, defaultLocale string, send MailLater) error {
	res, err := orm.QueryOf[models.Reservation](s.DB).Find(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("load reservation %d: %w", reservationID, err)
	}

	if res.Status != models.ReservationPaid {
		return nil
	}

	raffle, err := orm.QueryOf[models.Raffle](s.DB).Find(ctx, res.RaffleID)
	if err != nil {
		return fmt.Errorf("load raffle %d: %w", res.RaffleID, err)
	}

	owner, err := orm.QueryOf[models.User](s.DB).Find(ctx, raffle.UserID)
	if err != nil {
		return fmt.Errorf("load owner %d: %w", raffle.UserID, err)
	}

	appURL = strings.TrimRight(appURL, "/")

	buyerLocale := res.Locale
	if buyerLocale == "" {
		buyerLocale = defaultLocale
	}

	bctx := i18n.WithLocale(ctx, buyerLocale)
	if err := send(bctx, mail.PaymentConfirmed{
		To: res.BuyerEmail, Name: res.BuyerName, RaffleTitle: raffle.Title, Number: res.Number,
		Amount: FormatBRL(int64(res.AmountCents), buyerLocale), PublicURL: appURL + "/r/" + raffle.Slug,
		Subject: i18n.T(bctx, "mail.payment_confirmed.subject", "number", res.Number),
	}); err != nil {
		return fmt.Errorf("queue buyer mail: %w", err)
	}

	octx := i18n.WithLocale(ctx, defaultLocale)
	if err := send(octx, mail.NumberSold{
		To: owner.Email, OwnerName: owner.Name, BuyerName: res.BuyerName, RaffleTitle: raffle.Title, Number: res.Number,
		Amount: FormatBRL(int64(res.AmountCents), defaultLocale), ManageURL: appURL + "/raffles/" + strconv.FormatInt(raffle.ID, 10),
		Subject: i18n.T(octx, "mail.payment_number_sold.subject", "number", res.Number),
	}); err != nil {
		return fmt.Errorf("queue owner mail: %w", err)
	}

	return nil
}

// FormatBRL formats centavos as Brazilian reais: "R$ 1.234,56" in pt-BR,
// "R$1,234.56" otherwise.
func FormatBRL(cents int64, locale string) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}

	whole := strconv.FormatInt(cents/100, 10)
	frac := fmt.Sprintf("%02d", cents%100)

	thousands, decimal, prefix := ",", ".", "R$"
	if strings.HasPrefix(locale, "pt") {
		thousands, decimal, prefix = ".", ",", "R$ "
	}

	var b strings.Builder

	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteString(thousands)
		}

		b.WriteRune(r)
	}

	return sign + prefix + b.String() + decimal + frac
}
