package controllers

import (
	"net/http"
	"slices"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
)

// APIKeyController is the owner's API keys page (vukem auth.Tokens).
type APIKeyController struct {
	tokens *auth.Tokens
}

// NewAPIKeyController returns an APIKeyController.
func NewAPIKeyController(t *auth.Tokens) *APIKeyController { return &APIKeyController{tokens: t} }

// apiKey is an API key as listed on the page; the secret never leaves
// Store.
type apiKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Abilities  []string   `json:"abilities"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// Index lists the owner's API keys.
func (kc *APIKeyController) Index(c *web.Ctx) error { return kc.render(c, "") }

// Store creates an API key and renders the page with its plain token, the
// only time it is shown. Rendering instead of redirecting keeps the token
// out of the session store.
func (kc *APIKeyController) Store(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.APIKeySchema)
	if err != nil {
		return err
	}

	abilities := slices.Clone(in.Abilities)
	slices.Sort(abilities)

	plain, _, err := kc.tokens.Create(c.Context(), u.AuthID(), in.Name, abilities, 0)
	if err != nil {
		return err
	}

	return kc.render(c, plain)
}

// Destroy revokes one of the owner's keys; another owner's id is a 404.
func (kc *APIKeyController) Destroy(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	ok, err := kc.tokens.RevokeFor(c.Context(), u.AuthID(), c.Param("id"))
	if err != nil {
		return err
	}

	if !ok {
		return errs.New(http.StatusNotFound, i18n.T(c.Context(), "apikeys.not_found"))
	}

	inertia.Flash(c, "success", i18n.T(c.Context(), "apikeys.revoked"))

	return c.RedirectRoute("apikeys.index")
}

func (kc *APIKeyController) render(c *web.Ctx, plain string) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	tokens, err := kc.tokens.List(c.Context(), u.AuthID())
	if err != nil {
		return err
	}

	keys := make([]apiKey, 0, len(tokens))
	for _, t := range tokens {
		keys = append(keys, apiKey{ID: t.ID, Name: t.Name, Abilities: t.Abilities, LastUsedAt: t.LastUsedAt})
	}

	props := inertia.Props{"keys": keys, "abilities": requests.Abilities(), "plain_token": nil}
	if plain != "" {
		props["plain_token"] = plain
	}

	return inertia.Render(c, "Settings/ApiKeys", props)
}
