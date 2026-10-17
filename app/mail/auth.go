package mail

import (
	"context"

	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/mail"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// VerifyEmail asks a new user to confirm their address.
type VerifyEmail struct {
	User    models.User
	Link    string
	Subject string
}

// NewVerifyEmail builds the mail in the request locale.
//
// workaround(vukem v0.21.1): Envelope() has no context, so the subject is translated here.
func NewVerifyEmail(ctx context.Context, u models.User, link string) VerifyEmail {
	return VerifyEmail{User: u, Link: link, Subject: i18n.T(ctx, "mail.verify.subject")}
}

// Envelope implements mail.Mailable.
func (m VerifyEmail) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.User.Email}, Subject: m.Subject}
}

// Content implements mail.Mailable.
func (m VerifyEmail) Content() mail.Content { return mail.View("mail/verify_email", m) }

// ResetPassword carries a password reset link.
type ResetPassword struct {
	User    models.User
	Link    string
	Subject string
}

// NewResetPassword builds the mail in the request locale.
func NewResetPassword(ctx context.Context, u models.User, link string) ResetPassword {
	return ResetPassword{User: u, Link: link, Subject: i18n.T(ctx, "mail.reset.subject")}
}

// Envelope implements mail.Mailable.
func (m ResetPassword) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.User.Email}, Subject: m.Subject}
}

// Content implements mail.Mailable.
func (m ResetPassword) Content() mail.Content { return mail.View("mail/reset_password", m) }
