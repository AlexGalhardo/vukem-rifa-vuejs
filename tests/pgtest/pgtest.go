// Package pgtest gives each integration test its own PostgreSQL database.
//
// vukemtest.New defaults to an in-memory SQLite database; the app is
// Postgres-only in production, so integration tests (go test -tags=integration)
// override DATABASE_URL with a fresh database created here from
// TEST_DATABASE_URL (a role allowed to create databases).
package pgtest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx"
)

// NewDatabase creates an empty throwaway database, removes it in t.Cleanup
// and returns its URL.
func NewDatabase(t testing.TB) string {
	t.Helper()

	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Fatal("pgtest: set TEST_DATABASE_URL (compose: postgres://rifa:rifa@localhost:5432/rifa?sslmode=disable)")
	}

	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatalf("pgtest: open: %v", err)
	}

	if err := admin.PingContext(t.Context()); err != nil {
		t.Fatalf("pgtest: postgres unreachable at TEST_DATABASE_URL: %v", err)
	}

	// Lowercase letters and digits only: safe to splice into DDL, which
	// cannot take parameters.
	name := "rifa_test_" + strings.ToLower(rand.Text()[:12])
	if _, err := admin.ExecContext(t.Context(), "CREATE DATABASE "+name); err != nil {
		t.Fatalf("pgtest: create database: %v", err)
	}

	t.Cleanup(func() {
		// t.Context is already cancelled during Cleanup.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, _ = admin.ExecContext(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		_ = admin.Close()
	})

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("pgtest: parse url: %v", err)
	}

	u.Path = "/" + name

	return u.String()
}
