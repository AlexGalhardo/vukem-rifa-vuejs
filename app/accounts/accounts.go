// Package accounts wires authentication: guards, tokens, password resets
// and email verification. It is your code (vukem install:auth): change the
// rules freely.
package accounts

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/auth/hash"
	"github.com/AlexGalhardo/vukem/cache"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/session"
	"github.com/AlexGalhardo/vukem/signed"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
)

// Accounts holds the authentication services.
type Accounts struct {
	DB     *database.DB
	Mailer *mail.Mailer
	Users  *auth.ORMProvider[models.User]
	Web    *auth.SessionGuard[models.User]
	API    *auth.TokenGuard[models.User]
	Tokens *auth.Tokens
	Hasher hash.Hasher
	Signer *signed.Signer
	// URL is the public base URL used in emails (APP_URL).
	URL string
	// ResetTTL is how long password reset links work.
	ResetTTL time.Duration
	// VerifyTTL is how long email verification links work.
	VerifyTTL time.Duration
	now       func() time.Time
	sessions  func() (*session.Manager, error)
	cache     func() (*cache.Cache, error)
}

// RevokeOtherSessions signs userID out of every browser session except
// keep (the current session id, "" for all). Call it whenever the
// credentials change. Cookie sessions have no server-side record and are
// left alone; production uses SESSION_DRIVER=database.
func (a *Accounts) RevokeOtherSessions(ctx context.Context, userID int64, keep string) error {
	m, err := a.sessions()
	if err != nil {
		return err
	}

	err = m.RevokeOthers(ctx, a.Web.SessionUserID(strconv.FormatInt(userID, 10)), keep)
	if errors.Is(err, session.ErrRevokeUnsupported) {
		return nil
	}

	return err
}

// HasherFor returns the password hasher: argon2id (vukem defaults: 64 MiB,
// t=1, p=4) everywhere but the test kind, where a fast fake keeps tests quick.
func HasherFor(kind vukem.Kind) hash.Hasher {
	if kind == vukem.KindTest {
		return hash.Fake()
	}

	return hash.Default()
}

// New builds Accounts from the app.
func New(app *vukem.App) (*Accounts, error) {
	db, err := app.DB()
	if err != nil {
		return nil, err
	}

	mailer, err := app.Mailer()
	if err != nil {
		return nil, err
	}

	hasher := HasherFor(app.Kind)

	users := auth.NewORMProvider[models.User](db)
	tokens := auth.NewTokens(db)

	web := auth.NewSessionGuard("web", users, hasher, auth.NewRememberStore(db))
	web.InsecureCookies = !app.Config.IsProduction() && !strings.HasPrefix(app.Config.URL, "https://")

	return &Accounts{
		DB: db, Mailer: mailer, Users: users, Web: web, Tokens: tokens, Hasher: hasher,
		API:       auth.NewTokenGuard("api", users, tokens),
		Signer:    signed.New(app.Key.Derive("signed.urls", 32)),
		URL:       strings.TrimSuffix(app.Config.URL, "/"),
		ResetTTL:  time.Hour,
		VerifyTTL: time.Hour,
		now:       time.Now,
		sessions:  app.Sessions,
		cache:     app.Cache,
	}, nil
}
