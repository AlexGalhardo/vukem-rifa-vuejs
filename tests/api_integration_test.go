//go:build integration

package tests

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/export"
)

// The export reads the same rows on Postgres (timestamptz, append-only
// ledger, quoted "trigger" column) and produces the same golden files.
func TestPostgres_ExportMatchesGolden(t *testing.T) {
	t.Parallel()

	f := newExportFixture(t, postgresEnv(t))
	e := f.build(t)

	var js, csv bytes.Buffer
	if err := export.WriteJSON(&js, e); err != nil {
		t.Fatal(err)
	}

	if err := export.WriteCSV(&csv, e); err != nil {
		t.Fatal(err)
	}

	assertGolden(t, "export.json.golden", js.Bytes())
	assertGolden(t, "export.csv.golden", csv.Bytes())
}

// Idempotency records live in the database cache in production.
func TestPostgres_APIIdempotencyAndPagination(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t, "pg.api@example.com", postgresEnv(t))
	id := f.createAPIRaffle(t, validRaffle())
	f.createAPIRaffle(t, validRaffle())

	f.api.Get("/api/v1/raffles?per_page=1").AssertOK().AssertMatchesOpenAPI().
		AssertJSONCount("data", 1).AssertJSONPath("meta.total", 2).AssertJSONPath("data.0.available", 100)

	buyer := map[string]any{"buyer_name": "Maria Silva", "buyer_email": "maria@example.com", "buyer_phone": "11987654321"}
	link := fmt.Sprintf("/api/v1/raffles/%d/numbers/9/payment-links", id)

	f.api.WithHeader("Idempotency-Key", "pg-1")
	first := f.api.PostJSON(link, buyer).AssertCreated().AssertMatchesOpenAPI()
	replay := f.api.PostJSON(link, buyer).AssertCreated().AssertHeader("Idempotent-Replayed", "true")

	if first.Body() != replay.Body() {
		t.Fatal("replay differs")
	}

	vukemtest.AssertDatabaseCount(t, f.app, "reservations", 1)
}
