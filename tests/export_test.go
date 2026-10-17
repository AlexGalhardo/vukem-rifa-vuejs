package tests

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/export"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// goldenNow is the fake clock of the export fixture.
var goldenNow = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

// exportFixture is a raffle of 10 numbers with: a paid number (2) and its
// ledger split, a live link (5) for a buyer whose name is a formula, a late
// payment flagged for refund (7, expired) and a draw. Every timestamp and
// the slug are pinned so the output is byte-for-byte stable.
type exportFixture struct {
	app    *vukemtest.App
	client *vukemtest.Client
	raffle models.Raffle
}

func newExportFixture(t *testing.T, opts ...vukemtest.Options) exportFixture {
	t.Helper()

	app := vukemtest.New(t, bootstrap.New, opts...)
	c := owner(t, app, "dona.export@example.com")

	fields := validRaffle()
	fields["number_count"] = 10
	fields["title"] = `=HYPERLINK("https://evil.example","clique")`
	fields["description"] = "@SUM(1+1) prêmio"
	id := createRaffle(t, app, c, fields)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	exec := func(query string, args ...any) {
		t.Helper()

		// Same fixture on SQLite and Postgres: number the ? placeholders.
		var b strings.Builder

		n := 0
		for _, r := range query {
			if r == '?' {
				n++
				b.WriteString(db.Dialect().Placeholder(n))

				continue
			}

			b.WriteRune(r)
		}

		if _, err := db.ExecContext(t.Context(), b.String(), args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	exec("UPDATE raffles SET slug = 'golden0001', starts_at = ?, ends_at = ?, created_at = ?, updated_at = ? WHERE id = ?",
		start, start.AddDate(0, 0, 30), start, start, id)

	r, err := orm.QueryOf[models.Raffle](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	svc := reservations.New(db, func() time.Time { return goldenNow }, nil)

	for _, b := range []struct {
		number int
		buyer  requests.Buyer
	}{
		{2, requests.Buyer{Name: "Maria Silva", Email: "maria@example.com", Phone: "+5511987654321"}},
		{5, requests.Buyer{Name: "-2+3 João", Email: "joao@example.com", Phone: "+5521912345678"}},
		{7, requests.Buyer{Name: "Ana Souza", Email: "ana@example.com", Phone: "+5531998765432"}},
	} {
		if _, err := svc.Reserve(t.Context(), r, b.number, b.buyer); err != nil {
			t.Fatal(err)
		}
	}

	paidAt := goldenNow.Add(3 * time.Minute)
	exec("UPDATE reservations SET created_at = ?, updated_at = ?", goldenNow, goldenNow)
	exec("UPDATE reservations SET status = 'paid', paid_at = ?, charge_id = 'pix_char_123' WHERE number = 2", paidAt)
	exec("UPDATE raffle_numbers SET status = 'paid', paid_at = ? WHERE raffle_id = ? AND number = 2", paidAt, id)
	exec("UPDATE raffles SET locked_at = ? WHERE id = ?", paidAt, id)
	exec("UPDATE reservations SET status = 'expired', charge_id = 'pix_char_456' WHERE number = 7")
	exec("UPDATE raffle_numbers SET status = 'available', reservation_id = NULL WHERE raffle_id = ? AND number = 7", id)

	ledger := func(kind string, reservation, amount int64, note string) {
		exec("INSERT INTO ledger_entries (user_id, raffle_id, reservation_id, kind, amount_cents, note, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?)", r.UserID, id, reservation, kind, amount, note, paidAt, paidAt)
	}
	ledger(models.LedgerSaleOwnerShare, 1, 900, "")
	ledger(models.LedgerPlatformFee, 1, 100, "")
	ledger(models.LedgerRefundDue, 3, 1000, `=cmd|' /C calc'!A0`)

	exec(`INSERT INTO draws (raffle_id, drawn_at, seed, participants_count, participants_sha256, winning_number, algorithm, "trigger", created_at, updated_at) `+
		"VALUES (?, ?, ?, 1, ?, 2, 'v1', 'deadline', ?, ?)", id, start.AddDate(0, 0, 30),
		strings.Repeat("ab", 32), strings.Repeat("cd", 32), start, start)

	r, err = orm.QueryOf[models.Raffle](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	return exportFixture{app: app, client: c, raffle: r}
}

func (f exportFixture) build(t *testing.T) export.Export {
	t.Helper()

	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	e, err := export.New(db, func() time.Time { return goldenNow }).Build(t.Context(), f.raffle)
	if err != nil {
		t.Fatal(err)
	}

	return e
}

// assertGolden compares got with tests/testdata/name; UPDATE_GOLDEN=1
// rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(path) //nolint:gosec // fixed test file
	if err != nil {
		t.Fatalf("%v (run with UPDATE_GOLDEN=1 to create it)", err)
	}

	// Git may check text files out with CRLF on Windows.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(got, want) {
		t.Fatalf("%s differs from the golden file:\n--- got ---\n%s", name, got)
	}
}

func TestExport_JSONGolden(t *testing.T) {
	t.Parallel()

	f := newExportFixture(t)

	var buf bytes.Buffer
	if err := export.WriteJSON(&buf, f.build(t)); err != nil {
		t.Fatal(err)
	}

	assertGolden(t, "export.json.golden", buf.Bytes())
}

func TestExport_CSVGolden(t *testing.T) {
	t.Parallel()

	f := newExportFixture(t)

	var buf bytes.Buffer
	if err := export.WriteCSV(&buf, f.build(t)); err != nil {
		t.Fatal(err)
	}

	// encoding/csv ends records with \n; normalize like the golden reader.
	assertGolden(t, "export.csv.golden", buf.Bytes())

	rd := csv.NewReader(&buf)
	rd.FieldsPerRecord = -1

	rows, err := rd.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range rows {
		for _, cell := range row {
			if cell != "" && strings.ContainsRune("=+-@\t\r", rune(cell[0])) && !isInteger(cell) {
				t.Fatalf("unguarded formula cell %q in row %v", cell, row)
			}
		}
	}
}

func isInteger(s string) bool {
	var n int64
	_, err := fmt.Sscan(s, &n)

	return err == nil && fmt.Sprint(n) == s
}

func TestExport_TextGuardsFormulas(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"=1+1": "'=1+1", "+55 11": "'+55 11", "-2": "'-2", "@SUM(A1)": "'@SUM(A1)",
		"\tx": "'\tx", "\rx": "'\rx", "Maria": "Maria", "": "", " =x": " =x",
	}
	for in, want := range tests {
		if got := export.Text(in); got != want {
			t.Errorf("Text(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExport_BrowserDownloads(t *testing.T) {
	t.Parallel()

	f := newExportFixture(t)
	base := fmt.Sprintf("/raffles/%d/export", f.raffle.ID)

	f.client.Get(base+".json").AssertOK().
		AssertHeaderContains("Content-Type", "application/json").
		AssertHeader("Content-Disposition", `attachment; filename="rifa-golden0001-export.json"`).
		AssertHeaderContains("Cache-Control", "no-store").
		AssertJSONPath("raffle.slug", "golden0001").
		AssertJSONPath("reservations.2.refund_due", true)

	res := f.client.Get(base+".csv").AssertOK().
		AssertHeaderContains("Content-Type", "text/csv").
		AssertHeader("Content-Disposition", `attachment; filename="rifa-golden0001-export.csv"`).
		AssertHeaderContains("Cache-Control", "no-store")

	if !strings.Contains(res.Body(), `'=HYPERLINK`) {
		t.Fatalf("CSV title not guarded:\n%s", res.Body())
	}
}

func TestExport_OtherOwnerGetsNotFound(t *testing.T) {
	t.Parallel()

	f := newExportFixture(t)
	other := owner(t, f.app, "outro.export@example.com")

	for _, ext := range []string{"json", "csv"} {
		res := other.Get(fmt.Sprintf("/raffles/%d/export.%s", f.raffle.ID, ext)).AssertNotFound()
		res.AssertDontSee("maria@example.com")
	}

	// Guests are sent to the login page.
	f.app.Client(t).Get(fmt.Sprintf("/raffles/%d/export.csv", f.raffle.ID)).AssertRedirect("/login")
}
