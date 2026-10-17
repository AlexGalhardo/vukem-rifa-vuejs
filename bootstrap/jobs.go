package bootstrap

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/queue"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/jobs"
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

		return q.Register(
			queue.Handle(func(ctx context.Context, j jobs.ExpireReservation) error {
				_, err := res.Expire(ctx, j.ReservationID)
				return err
			}),
			queue.Handle(func(ctx context.Context, j jobs.RunDraw) error {
				_, _, err := draw.Run(ctx, j.RaffleID, j.Trigger)
				return err
			}),
		)
	})
}
