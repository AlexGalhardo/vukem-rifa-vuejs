// Package bootstrap wires the application together. There is no hidden
// registration: everything the app uses is listed here.
package bootstrap

import (
	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/drivers/postgres"
	"github.com/AlexGalhardo/vukem/inertia"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/jobs"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/listeners"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/database/migrations"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/database/seeders"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/resources"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/routes"
)

// New builds the application; cmd/app and the tests call it.
func New(opts ...vukem.Option) (*vukem.App, error) {
	base := []vukem.Option{
		vukem.WithDatabaseDrivers(postgres.Driver()),
		vukem.WithProviders(Providers()...),
		vukem.WithMigrations(migrations.All()),
		vukem.WithSeeders(seeders.All()...),
		vukem.WithListeners(listeners.All()...),
		vukem.WithJobs(jobs.All()...),
		vukem.WithJobsFunc(Jobs),
		vukem.WithSchedule(routes.Schedule),
		vukem.WithViews(resources.Views()),
		vukem.WithTranslations(resources.Lang()),
		vukem.WithVite(resources.Public()),
		vukem.WithInertia(inertia.Config{}),
	}

	for _, fn := range routes.All() {
		base = append(base, vukem.WithRoutes(fn))
	}

	app, err := vukem.New(append(base, opts...)...)
	if err != nil {
		return nil, err
	}

	app.AddCommand(Commands()...)

	return app, nil
}
