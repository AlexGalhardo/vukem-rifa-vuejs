package seeders

import (
	"context"
	"errors"
	"fmt"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/draws"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
)

// DrawDemoSlug is the public slug of the demo raffle e2e/draw.spec.ts reads.
const DrawDemoSlug = "demodraw01"

// DrawDemoPaid are the demo raffle's paid numbers (the draw participants).
var DrawDemoPaid = []int{2, 5, 9}

// DrawDemo creates a finished, drawn raffle (/r/demodraw01) with three paid
// numbers, so the e2e suite can check the public result without waiting
// for a real deadline. vukem db:seed refuses to run in production without
// --force; the demo owner has no password and cannot log in. Running it
// again is a no-op.
func DrawDemo() vukem.Seeder {
	return vukem.SeederFunc(func(ctx context.Context, db *database.DB) error {
		_, err := orm.QueryOf[models.Raffle](db).Where("slug", DrawDemoSlug).First(ctx)
		if err == nil {
			return nil
		}

		if !errors.Is(err, orm.ErrNotFound) {
			return err
		}

		now := time.Now().UTC()
		u := models.User{Name: "Demo", Email: "draw-demo@example.invalid"}
		u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = now, true

		if err := orm.Create(ctx, db, &u); err != nil {
			return fmt.Errorf("demo owner: %w", err)
		}

		r, err := raffles.New(db, time.Now, nil).Create(ctx, u.ID, requests.Raffle{
			Title: "Rifa de demonstração", Description: "Sorteio de exemplo para conferência.",
			NumberCount: 10, PriceReais: 5, DurationDays: 7,
		})
		if err != nil {
			return fmt.Errorf("demo raffle: %w", err)
		}

		res := reservations.New(db, time.Now, nil)
		p := db.Dialect().Placeholder

		for _, n := range DrawDemoPaid {
			link, err := res.Reserve(ctx, r, n, requests.Buyer{
				Name: fmt.Sprintf("Comprador %d", n), Email: fmt.Sprintf("buyer%d@example.invalid", n), Phone: "+5511987654321",
			})
			if err != nil {
				return err
			}

			if _, err := db.ExecContext(ctx, "UPDATE reservations SET status = 'paid', paid_at = "+p(1)+" WHERE id = "+p(2),
				now, link.Reservation.ID); err != nil {
				return err
			}

			if _, err := db.ExecContext(ctx, "UPDATE raffle_numbers SET status = 'paid', paid_at = "+p(1)+" WHERE id = "+p(2),
				now, link.Reservation.NumberID); err != nil {
				return err
			}
		}

		if _, err := db.ExecContext(ctx, "UPDATE raffles SET slug = "+p(1)+", locked_at = "+p(2)+", starts_at = "+p(3)+
			", ends_at = "+p(4)+" WHERE id = "+p(5), DrawDemoSlug, now, now.AddDate(0, 0, -8), now.Add(-time.Hour), r.ID); err != nil {
			return err
		}

		_, _, err = draws.New(db, time.Now).Run(ctx, r.ID, models.DrawDeadline)

		return err
	})
}
