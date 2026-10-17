//go:build abacatepay

package abacatepay_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/payments/abacatepay"
)

// Real AbacatePay dev mode: go test -tags=abacatepay -p 1 -run AbacatePaySandbox ./...
// with ABACATEPAY_API_KEY set to a dev-mode key (abc_dev_…).
func TestAbacatePaySandbox_CreateCheckAndSimulatePix(t *testing.T) {
	key := os.Getenv("ABACATEPAY_API_KEY")
	if key == "" {
		t.Skip("ABACATEPAY_API_KEY is not set")
	}

	if !strings.HasPrefix(key, "abc_dev_") {
		t.Skip("refusing to run against a production key (want abc_dev_…)")
	}

	c := abacatepay.New(os.Getenv("ABACATEPAY_BASE_URL"), key, nil)
	ref := "sandbox-" + strconv.FormatInt(time.Now().UnixNano(), 36)

	ch, err := c.CreatePix(t.Context(), abacatepay.PixRequest{
		AmountCents: 100, ExpiresIn: 600, Description: "vukem-rifa sandbox test", ExternalID: ref,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	t.Logf("created %s status=%s devMode=%v", ch.ID, ch.Status, ch.DevMode)

	if ch.Status != abacatepay.StatusPending || !ch.DevMode || ch.AmountCents != 100 {
		t.Fatalf("charge = %+v", ch)
	}

	paid, err := c.SimulatePayment(t.Context(), ch.ID)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}

	t.Logf("simulated %s status=%s", paid.ID, paid.Status)

	st, err := c.Check(t.Context(), ch.ID)
	if err != nil || st.Status != abacatepay.StatusPaid {
		t.Fatalf("check = %+v, %v", st, err)
	}
}
