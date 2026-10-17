package bootstrap

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/queue"
)

// Background runs the database queue worker inside "serve", so one Railway
// service handles HTTP, reservation expiry, draws and emails (PLAN.md D15).
// vukem runs the scheduler in serve itself (SCHEDULE_IN_SERVE, default
// true). Set BACKGROUND_IN_SERVE=false when "queue:work" runs as a separate
// service, and SCHEDULE_IN_SERVE=false for a separate "schedule:work".
//
// workaround(vukem v0.24.0): serve only starts a worker for QUEUE_CONNECTION=memory.
func Background() vukem.Provider {
	return vukem.ProviderFunc(func(app *vukem.App) error {
		if app.Kind != vukem.KindWeb || app.Source.Get("BACKGROUND_IN_SERVE") == "false" ||
			app.Source.Get("QUEUE_CONNECTION") != "database" {
			return nil
		}

		app.AddRunner("queue-worker", vukem.RunnerFunc(func(ctx context.Context) error {
			q, err := app.Queue()
			if err != nil {
				return err
			}

			return q.Work(ctx, queue.WorkOptions{Queues: []string{"default"}, Concurrency: 4})
		}))

		return nil
	})
}
