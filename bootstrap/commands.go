package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	vukem "github.com/AlexGalhardo/vukem"
)

// Commands lists the application commands; vukem make:command appends here.
func Commands() []vukem.Command {
	return []vukem.Command{makeAdmin()}
}

// makeAdmin grants users.is_admin (payouts page) to the account with the
// given email: "app make:admin owner@example.com".
func makeAdmin() vukem.Command {
	return vukem.CommandFunc{
		CommandName: "make:admin",
		Desc:        "Grant admin (payouts) to the user with the given email",
		Fn: func(ctx context.Context, app *vukem.App, args []string) error {
			if len(args) != 1 || !strings.Contains(args[0], "@") {
				return errors.New("usage: make:admin <email>")
			}

			db, err := app.DB()
			if err != nil {
				return err
			}

			p := db.Dialect().Placeholder

			out, err := db.ExecContext(ctx, "UPDATE users SET is_admin = "+p(1)+" WHERE email = "+p(2),
				true, strings.ToLower(strings.TrimSpace(args[0])))
			if err != nil {
				return fmt.Errorf("make admin: %w", err)
			}

			if n, err := out.RowsAffected(); err != nil || n != 1 {
				return fmt.Errorf("make admin: no user with email %q", args[0])
			}

			app.Log.InfoContext(ctx, "make:admin: admin granted", "email", args[0])

			return nil
		},
	}
}
