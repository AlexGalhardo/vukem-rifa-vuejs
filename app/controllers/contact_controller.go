package controllers

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// ContactController edits the organizer's opt-in public contact.
type ContactController struct{ db *database.DB }

// NewContactController returns a ContactController.
func NewContactController(db *database.DB) *ContactController { return &ContactController{db: db} }

func contactProps(u models.User) map[string]any {
	str := func(p *string) string {
		if p == nil {
			return ""
		}

		return *p
	}

	return map[string]any{
		"show_email": u.ShowEmail, "email": u.Email,
		"whatsapp": str(u.WhatsApp), "show_whatsapp": u.ShowWhatsApp,
		"telegram": str(u.Telegram), "show_telegram": u.ShowTelegram,
	}
}

// Edit shows the contact settings (all off until the organizer opts in).
func (cc *ContactController) Edit(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	return inertia.Render(c, "Settings/Contact", inertia.Props{"contact": contactProps(u)})
}

// Update saves the contact settings and audits which fields changed (never
// their values).
func (cc *ContactController) Update(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.ContactSchema)
	if err != nil {
		return err
	}

	opt := func(s string) *string {
		if s == "" {
			return nil
		}

		return &s
	}

	before := contactProps(u)

	if _, err := orm.QueryOf[models.User](cc.db).Where("id", u.ID).Update(c.Context(), orm.Set{
		"show_email": in.ShowEmail, "whatsapp": opt(in.WhatsApp), "show_whatsapp": in.ShowWhatsApp,
		"telegram": opt(in.Telegram), "show_telegram": in.ShowTelegram,
	}); err != nil {
		return err
	}

	u.ShowEmail, u.WhatsApp, u.ShowWhatsApp = in.ShowEmail, opt(in.WhatsApp), in.ShowWhatsApp
	u.Telegram, u.ShowTelegram = opt(in.Telegram), in.ShowTelegram

	var changed []string

	for k, v := range contactProps(u) {
		if before[k] != v {
			changed = append(changed, k)
		}
	}

	web.AuditEvent(c.Request(), "contact.updated", "actor_id", u.ID, "fields", changed)
	inertia.Flash(c, "success", i18n.T(c.Context(), "contact.saved"))

	return c.Redirect("/settings/contact")
}
