package bootstrap

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/i18n"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/queue"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/jobs"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Jobs registers the queue handlers that need app services (database,
// clock).
//
// workaround(vukem v0.22.0): jobs.All() / WithJobs get no *App, so handlers needing the database are registered here.
func Jobs() vukem.Provider {
	return vukem.ProviderFunc(func(app *vukem.App) error {
		q, err := app.Queue()
		if err != nil {
			return err
		}

		db, err := app.DB()
		if err != nil {
			return err
		}

		res := reservations.New(db, app.Now, app.Queue)
		draw := draws.New(db, app.Now)
		pay := payments.New(db, app.Now, nil, app.Queue, app.Log)

		return q.Register(
			queue.Handle(func(ctx context.Context, j jobs.ExpireReservation) error {
				_, err := res.Expire(ctx, j.ReservationID)
				return err
			}),
			queue.Handle(func(ctx context.Context, j jobs.RunDraw) error {
				_, _, err := draw.Run(ctx, j.RaffleID, j.Trigger)
				return err
			}),
			queue.Handle(func(ctx context.Context, j jobs.SendPaymentEmails) error {
				return payments.SendEmails(i18n.WithBundle(ctx, app.I18n), pay, j.ReservationID, app.Config.URL, app.I18n.Default(), func(ctx context.Context, m mail.Mailable) error { return app.MailLater(ctx, m) })
			}),
		)
	})
}
