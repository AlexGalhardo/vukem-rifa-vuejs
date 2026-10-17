package account

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
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

// Verify handles the signed link from the verification email.
func (h *Handlers) Verify(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if c.Param("id") != u.AuthID() {
		return errs.New(403, i18n.T(c.Context(), "auth.verify.wrong_account"))
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
