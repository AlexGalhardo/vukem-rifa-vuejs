package bootstrap

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/queue"
)

// Background runs the database queue worker and the scheduler inside
// "serve", so one Railway service handles HTTP, reservation expiry, draws
// and session pruning (PLAN.md D15). BACKGROUND_IN_SERVE=false turns it off
// when "queue:work" and "schedule:work" run as separate services; the
// scheduler's WithoutOverlapping locks live in CACHE_STORE=database.
//
// workaround(vukem v0.22.0): serve only starts a worker for QUEUE_CONNECTION=memory and never the scheduler.
func Background() vukem.Provider {
	return vukem.ProviderFunc(func(app *vukem.App) error {
		if app.Kind != vukem.KindWeb || app.Source.Get("BACKGROUND_IN_SERVE") == "false" {
			return nil
		}

		if app.Source.Get("QUEUE_CONNECTION") == "database" {
			app.AddRunner("queue-worker", vukem.RunnerFunc(func(ctx context.Context) error {
				q, err := app.Queue()
				if err != nil {
					return err
				}

				return q.Work(ctx, queue.WorkOptions{Queues: []string{"default"}, Concurrency: 4})
			}))
		}

		app.AddRunner("scheduler", vukem.RunnerFunc(func(ctx context.Context) error {
			s, err := app.Schedule()
			if err != nil {
				return err
			}

			return s.Work(ctx, app.Now)
		}))

		return nil
	})
}
