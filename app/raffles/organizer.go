package raffles

import (
	"context"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// ContactLink is an opt-in contact shown on the public page.
type ContactLink struct {
	URL     string `json:"url"`
	Display string `json:"display"`
}

// SplitEmail is the organizer email in two parts, joined only in the
// browser (and on a click) so the page HTML never holds the address.
type SplitEmail struct {
	User   string `json:"user"`
	Domain string `json:"domain"`
}

// Organizer is who runs a raffle, for buyers to check it is the right one:
// the name, how long they have been here and how many raffles they ran,
// plus only the contacts they opted in to show.
type Organizer struct {
	Name         string       `json:"name"`
	MemberSince  time.Time    `json:"member_since"`
	RafflesTotal int64        `json:"raffles_total"`
	RafflesDone  int64        `json:"raffles_finished"`
	Email        *SplitEmail  `json:"email"`
	WhatsApp     *ContactLink `json:"whatsapp"`
	Telegram     *ContactLink `json:"telegram"`
}

// OrganizerOf returns the public organizer card of raffle r.
func (s *Service) OrganizerOf(ctx context.Context, r models.Raffle) (Organizer, error) {
	u, err := orm.QueryOf[models.User](s.DB).Find(ctx, r.UserID)
	if err != nil {
		return Organizer{}, err
	}

	total, err := orm.QueryOf[models.Raffle](s.DB).Where("user_id", u.ID).Count(ctx)
	if err != nil {
		return Organizer{}, err
	}

	done, err := orm.QueryOf[models.Raffle](s.DB).Where("user_id", u.ID).
		WhereIn("status", models.RaffleDrawn, models.RaffleClosed).Count(ctx)
	if err != nil {
		return Organizer{}, err
	}

	o := Organizer{Name: u.Name, MemberSince: u.CreatedAt, RafflesTotal: total, RafflesDone: done}

	if u.ShowEmail {
		if user, domain, ok := strings.Cut(u.Email, "@"); ok {
			o.Email = &SplitEmail{User: user, Domain: domain}
		}
	}

	if u.ShowWhatsApp && u.WhatsApp != nil && *u.WhatsApp != "" {
		o.WhatsApp = &ContactLink{URL: "https://wa.me/" + strings.TrimPrefix(*u.WhatsApp, "+"), Display: displayPhone(*u.WhatsApp)}
	}

	if u.ShowTelegram && u.Telegram != nil && *u.Telegram != "" {
		o.Telegram = &ContactLink{URL: "https://t.me/" + *u.Telegram, Display: "@" + *u.Telegram}
	}

	return o, nil
}

// displayPhone formats an E.164 Brazilian number as "+55 11 98765-4321".
func displayPhone(e164 string) string {
	d := strings.TrimPrefix(e164, "+55")
	if len(d) < 10 {
		return e164
	}

	area, rest := d[:2], d[2:]
	split := len(rest) - 4

	return "+55 " + area + " " + rest[:split] + "-" + rest[split:]
}
