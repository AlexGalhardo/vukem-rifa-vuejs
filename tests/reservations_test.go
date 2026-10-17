package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/ledger"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/requests"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/reservations"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

// reservationFixture is an app on a fake clock with one raffle (100
// numbers, R$ 10) and its reservation service.
type reservationFixture struct {
	app    *vukemtest.App
	clock  *vukemtest.FakeClock
	raffle models.Raffle
	svc    *reservations.Service
}

func newReservationFixture(t *testing.T, opts ...vukemtest.Options) reservationFixture {
	t.Helper()

	// Start at the real time: the test cookie jar drops cookies that expired
	// by the wall clock.
	clock := vukemtest.NewFakeClock(time.Now().UTC().Truncate(time.Second))

	var o vukemtest.Options
	if len(opts) > 0 {
		o = opts[0]
	}

	o.Clock = clock.Now
	app := vukemtest.New(t, bootstrap.New, o)
	c := owner(t, app, "dono@example.com")
	id := createRaffle(t, app, c, validRaffle())

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	r, err := orm.QueryOf[models.Raffle](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	return reservationFixture{app: app, clock: clock, raffle: r, svc: reservations.New(db, clock.Now, app.Queue)}
}

func (f reservationFixture) numberStatus(t *testing.T, number int) string {
	t.Helper()

	n, err := orm.QueryOf[models.RaffleNumber](f.svc.DB).Where("raffle_id", f.raffle.ID).Where("number", number).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return n.Status
}

var buyer = requests.Buyer{Name: "Maria Silva", Email: "maria@example.com", Phone: "+5511987654321"}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()

	if err == nil || errs.Status(err) != status {
		t.Fatalf("err = %v, want HTTP %d", err, status)
	}
}

func TestReservation_ReserveCreatesHashedLink(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t)

	link, err := f.svc.Reserve(t.Context(), f.raffle, 7, buyer)
	if err != nil {
		t.Fatal(err)
	}

	if len(link.Token) < 43 {
		t.Fatalf("token %q is shorter than 256 bits", link.Token)
	}

	res := link.Reservation
	if res.TokenHash == link.Token || res.TokenHash != reservations.HashToken(link.Token) {
		t.Fatal("token must be stored as its SHA-256 only")
	}

	if res.Status != models.ReservationPending || res.AmountCents != 1000 ||
		!res.ExpiresAt.Equal(f.clock.Now().Add(10*time.Minute)) {
		t.Fatalf("reservation = %+v", res)
	}

	if got := f.numberStatus(t, 7); got != models.NumberReserved {
		t.Fatalf("number status = %q, want reserved", got)
	}

	got, err := f.svc.ByToken(t.Context(), link.Token)
	if err != nil || got.ID != res.ID {
		t.Fatalf("ByToken = %v, %v", got.ID, err)
	}

	_, err = f.svc.ByToken(t.Context(), "not-a-token")
	wantStatus(t, err, http.StatusNotFound)
}

func TestReservation_RefusesUnavailableNumbers(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t)

	if _, err := f.svc.Reserve(t.Context(), f.raffle, 1, buyer); err != nil {
		t.Fatal(err)
	}

	_, err := f.svc.Reserve(t.Context(), f.raffle, 1, buyer)
	wantStatus(t, err, http.StatusConflict)

	for _, n := range []int{0, 101} {
		_, err = f.svc.Reserve(t.Context(), f.raffle, n, buyer)
		wantStatus(t, err, http.StatusNotFound)
	}

	ended := f.raffle
	ended.EndsAt = f.clock.Now()
	_, err = f.svc.Reserve(t.Context(), ended, 2, buyer)
	wantStatus(t, err, http.StatusConflict)

	archived := f.raffle
	archived.Status = models.RaffleArchived
	_, err = f.svc.Reserve(t.Context(), archived, 2, buyer)
	wantStatus(t, err, http.StatusConflict)
}

func TestReservation_ExpiresAfterTenMinutes(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t)

	link, err := f.svc.Reserve(t.Context(), f.raffle, 3, buyer)
	if err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(10*time.Minute - time.Second)

	if ok, err := f.svc.Expire(t.Context(), link.Reservation.ID); err != nil || ok {
		t.Fatalf("Expire before due = %v, %v", ok, err)
	}

	f.clock.Advance(time.Second)

	n, err := f.svc.ExpireDue(t.Context())
	if err != nil || n != 1 {
		t.Fatalf("ExpireDue = %d, %v; want 1", n, err)
	}

	if got := f.numberStatus(t, 3); got != models.NumberAvailable {
		t.Fatalf("number status = %q, want available", got)
	}

	res, _ := f.svc.ByToken(t.Context(), link.Token)
	if res.Status != models.ReservationExpired || res.Live(f.clock.Now()) {
		t.Fatalf("reservation status = %q", res.Status)
	}

	// The delayed job running after the sweeper is a no-op.
	if ok, err := f.svc.Expire(t.Context(), link.Reservation.ID); err != nil || ok {
		t.Fatalf("second Expire = %v, %v", ok, err)
	}

	// The number can be sold again with a new link.
	if _, err := f.svc.Reserve(t.Context(), f.raffle, 3, buyer); err != nil {
		t.Fatal(err)
	}
}

func TestReservation_AwaitingPaymentGetsGrace(t *testing.T) {
	t.Parallel()

	res := models.Reservation{Status: models.ReservationAwaitingPayment, ExpiresAt: time.Unix(1000, 0)}
	if reservations.Expired(res, res.ExpiresAt.Add(30*time.Second)) {
		t.Fatal("awaiting payment expired inside the grace period")
	}

	if !reservations.Expired(res, res.ExpiresAt.Add(reservations.PaymentGrace)) {
		t.Fatal("awaiting payment did not expire after the grace period")
	}

	res.Status = models.ReservationPaid
	if reservations.Expired(res, res.ExpiresAt.Add(time.Hour)) {
		t.Fatal("paid reservations never expire")
	}
}

func TestReservation_Cancel(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t)

	link, err := f.svc.Reserve(t.Context(), f.raffle, 9, buyer)
	if err != nil {
		t.Fatal(err)
	}

	if err := f.svc.Cancel(t.Context(), f.raffle.ID, 9); err != nil {
		t.Fatal(err)
	}

	if got := f.numberStatus(t, 9); got != models.NumberAvailable {
		t.Fatalf("number status = %q, want available", got)
	}

	res, _ := f.svc.ByToken(t.Context(), link.Token)
	if res.Status != models.ReservationCancelled {
		t.Fatalf("status = %q, want cancelled", res.Status)
	}

	wantStatus(t, f.svc.Cancel(t.Context(), f.raffle.ID, 9), http.StatusConflict)
}

func TestReservation_ExpiryNeverReleasesPaidNumbers(t *testing.T) {
	t.Parallel()

	f := newReservationFixture(t)

	link, err := f.svc.Reserve(t.Context(), f.raffle, 4, buyer)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.DB.ExecContext(t.Context(), "UPDATE reservations SET status = 'paid' WHERE id = ?", link.Reservation.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.DB.ExecContext(t.Context(), "UPDATE raffle_numbers SET status = 'paid' WHERE raffle_id = ? AND number = 4", f.raffle.ID); err != nil {
		t.Fatal(err)
	}

	f.clock.Advance(time.Hour)

	if n, err := f.svc.ExpireDue(t.Context()); err != nil || n != 0 {
		t.Fatalf("ExpireDue = %d, %v; want 0", n, err)
	}

	if got := f.numberStatus(t, 4); got != models.NumberPaid {
		t.Fatalf("number status = %q, want paid", got)
	}
}

func TestBuyer_PhoneFormats(t *testing.T) {
	t.Parallel()

	ok := map[string]string{
		"(11) 98765-4321":   "+5511987654321",
		"11987654321":       "+5511987654321",
		"+55 11 98765 4321": "+5511987654321",
		"+5511987654321":    "+5511987654321",
		"55 21 3333-4444":   "+552133334444",
		"(21) 3333-4444":    "+552133334444",
	}
	for in, want := range ok {
		got, err := requests.BuyerSchema.Parse(map[string]any{"buyer_name": "Ana", "buyer_email": "Ana@Example.com", "buyer_phone": in})
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}

		if got.Phone != want || got.Email != "ana@example.com" {
			t.Errorf("%q = %q (%q), want %q", in, got.Phone, got.Email, want)
		}
	}

	for _, in := range []string{"", "123", "(01) 98765-4321", "11 8765-432", "+1 415 555 1234", "11 98765-43210", "abc"} {
		if _, err := requests.BuyerSchema.Parse(map[string]any{"buyer_name": "Ana", "buyer_email": "a@b.co", "buyer_phone": in}); err == nil {
			t.Errorf("%q: accepted", in)
		}
	}
}

func TestLedger_Split(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ amount, owner, fee int64 }{
		{100, 90, 10}, {1000, 900, 100}, {105, 95, 10}, {109, 99, 10}, {10000000, 9000000, 1000000},
	} {
		owner, fee := ledger.Split(tc.amount)
		if owner != tc.owner || fee != tc.fee || owner+fee != tc.amount {
			t.Errorf("Split(%d) = %d, %d; want %d, %d", tc.amount, owner, fee, tc.owner, tc.fee)
		}
	}
}
