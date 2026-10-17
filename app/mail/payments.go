package mail

import "github.com/AlexGalhardo/vukem/mail"

// PaymentConfirmed tells the buyer their PIX was received and the number
// is theirs.
type PaymentConfirmed struct {
	To          string
	Name        string
	RaffleTitle string
	Number      int
	Amount      string
	PublicURL   string
	Subject     string
}

// Envelope implements mail.Mailable.
func (m PaymentConfirmed) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.To}, Subject: m.Subject}
}

// Content implements mail.Mailable.
func (m PaymentConfirmed) Content() mail.Content { return mail.View("mail/payment_confirmed", m) }

// NumberSold tells the owner a number was paid.
type NumberSold struct {
	To          string
	OwnerName   string
	BuyerName   string
	RaffleTitle string
	Number      int
	Amount      string
	ManageURL   string
	Subject     string
}

// Envelope implements mail.Mailable.
func (m NumberSold) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.To}, Subject: m.Subject}
}

// Content implements mail.Mailable.
func (m NumberSold) Content() mail.Content { return mail.View("mail/number_sold", m) }
