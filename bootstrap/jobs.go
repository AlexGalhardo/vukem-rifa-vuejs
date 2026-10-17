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

// Jobs lists the queue handlers that need app services (database, clock),
// passed to vukem.WithJobsFunc. It lives in bootstrap because app/jobs only
// holds the job types the services import. It must not touch app.Queue
// while building (the services receive it as a func and call it later).
func Jobs(app *vukem.App) []queue.Registration {
	db, err := app.DB()
	if err != nil {
		// vukem.New fails earlier without a database; keep the app booting.
		app.Log.Error("jobs: no database", "err", err)
		return nil
	}

	res := reservations.New(db, app.Now, app.Queue)
	draw := draws.New(db, app.Now)
	pay := payments.New(db, app.Now, nil, app.Queue, app.Log)

	return []queue.Registration{
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
	}
}
