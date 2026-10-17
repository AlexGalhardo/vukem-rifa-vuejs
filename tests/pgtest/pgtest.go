// Package pgtest gives each integration test its own PostgreSQL database
// (go test -tags=integration, TEST_DATABASE_URL = a role allowed to create
// databases). It delegates to the driver's pgtest since vukem drivers/postgres
// v0.4.0.
package pgtest

import (
	"testing"

	"github.com/AlexGalhardo/vukem/drivers/postgres/pgtest"
)

// NewDatabase creates an empty throwaway database, drops it in t.Cleanup
// and returns its URL. Call it before vukemtest.New so the app's
// connections close first.
func NewDatabase(t testing.TB) string {
	t.Helper()

	return pgtest.Database(t, "")
}
