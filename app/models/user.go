package models

import (
	"strconv"

	"github.com/AlexGalhardo/vukem/orm"
)

// User is a registered account (vukem install:auth).
type User struct {
	orm.Model

	Name            string       `json:"name"`
	Email           string       `json:"email"`
	Password        string       `json:"-"`
	EmailVerifiedAt orm.NullTime `json:"email_verified_at"`
	IsAdmin         bool         `json:"is_admin"`

	// Opt-in public contact (organizer card on the public raffle page).
	ShowEmail    bool    `json:"-"`
	WhatsApp     *string `db:"whatsapp" json:"-"`
	ShowWhatsApp bool    `db:"show_whatsapp" json:"-"`
	Telegram     *string `json:"-"`
	ShowTelegram bool    `json:"-"`
}

// AuthID implements auth.Authenticatable.
func (u User) AuthID() string { return strconv.FormatInt(u.ID, 10) }

// AuthPassword implements auth.Authenticatable.
func (u User) AuthPassword() string { return u.Password }

// EmailVerified implements auth.Verifiable.
func (u User) EmailVerified() bool { return u.EmailVerifiedAt.Valid }
