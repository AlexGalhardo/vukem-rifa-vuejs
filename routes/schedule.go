package routes

import (
	"context"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/schedule"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// Schedule declares the scheduled tasks. "serve" runs them in-process
// (bootstrap.Background); "schedule:work" runs them in a separate service.
// vukem also schedules its hourly session:prune here.
func Schedule(s *schedule.Schedule, app *vukem.App) {
	s.Func("reservations:expire", func(ctx context.Context) error {
		db, err := app.DB()
		if err != nil {
			return err
		}

		_, err = reservations.New(db, app.Now, app.Queue).ExpireDue(ctx)

		return err
	}).EveryMinute().WithoutOverlapping()

	// Draws raffles whose delayed RunDraw job was lost or never queued.
	s.Func("draws:due", func(ctx context.Context) error {
		db, err := app.DB()
		if err != nil {
			return err
		}

		_, err = draws.New(db, app.Now).RunDue(ctx)

		return err
	}).EveryMinute().WithoutOverlapping()
}
