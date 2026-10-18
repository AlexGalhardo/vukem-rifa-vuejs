package account

import (
	"errors"
	"net/http"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/signed"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Notice asks the user to check their inbox.
func (*Handlers) Notice(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if u.EmailVerified() {
		return c.Redirect("/dashboard")
	}

	return inertia.Render(c, "Auth/VerifyEmail", nil)
}

// Verify handles the signed link from the verification email. It needs
// the login of the link's account (route middleware: signed out, the login
// comes back here), so someone who signed up with another person's address
// cannot get it verified by that person's click (vukem v0.27.0, A20). Signed
// in as someone else: 403.
func (h *Handlers) Verify(c *web.Ctx) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if u.ID != id {
		return errs.New(http.StatusForbidden, i18n.T(c.Context(), "auth.verify.wrong_account"))
	}

	if u.EmailVerified() {
		return c.Redirect("/dashboard")
	}

	if err := h.a.Signer.Verify(c.Request()); err != nil {
		if errors.Is(err, signed.ErrExpired) {
			inertia.Flash(c, "status", i18n.T(c.Context(), "auth.verify.expired"))

			return c.Redirect("/email/verify")
		}

		web.SecurityEvent(c.Request(), "url.signature_invalid")

		return errs.New(http.StatusForbidden, i18n.T(c.Context(), "auth.verify.invalid"))
	}

	if err := h.a.MarkVerified(c.Context(), h.a.DB, &u); err != nil {
		return err
	}

	inertia.Flash(c, "status", i18n.T(c.Context(), "auth.verify.done"))

	return c.Redirect("/dashboard")
}

// Resend sends a new verification email.
func (h *Handlers) Resend(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := h.a.SendVerification(c.Context(), u); err != nil {
		return err
	}

	inertia.Flash(c, "status", i18n.T(c.Context(), "auth.verify.resent"))

	return c.Redirect("/email/verify")
}
