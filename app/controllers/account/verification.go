package account

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/orm"
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

// Verify handles the signed link from the verification email. Signed in
// as the link's account, it goes to the dashboard; signed out, to the login
// page (the link proves the mailbox, not the password); signed in as
// someone else, 403.
func (h *Handlers) Verify(c *web.Ctx) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return err
	}

	u, err := orm.QueryOf[models.User](h.a.DB).Find(c.Context(), id)
	if err != nil {
		return errs.New(http.StatusNotFound, i18n.T(c.Context(), "auth.verify.wrong_account"))
	}

	// No auth middleware on this route: ask the guard directly.
	signedIn := h.a.Web.Check(c)

	var current models.User
	if signedIn {
		if current, err = h.a.Web.User(c); err != nil {
			return err
		}
	}

	if signedIn && current.ID != u.ID {
		return errs.New(http.StatusForbidden, i18n.T(c.Context(), "auth.verify.wrong_account"))
	}

	if err := h.a.MarkVerified(c.Context(), h.a.DB, &u); err != nil {
		return err
	}

	inertia.Flash(c, "status", i18n.T(c.Context(), "auth.verify.done"))

	if !signedIn {
		return c.Redirect("/login")
	}

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
