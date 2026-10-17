package abacatepay

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"sync"
)

// Fake is an in-memory Gateway for development and e2e (ABACATEPAY_FAKE,
// refused in production). Charges are deterministic per ExternalID, like
// the real API's idempotency on externalId; the QR image is a pattern, not
// a payable PIX code.
type Fake struct {
	mu      sync.Mutex
	charges map[string]Charge
}

var _ Gateway = (*Fake)(nil)

// NewFake returns an empty fake gateway.
func NewFake() *Fake { return &Fake{charges: map[string]Charge{}} }

// CreatePix implements Gateway.
func (f *Fake) CreatePix(_ context.Context, in PixRequest) (Charge, error) {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d", in.ExternalID, in.AmountCents))
	id := "pix_char_fake_" + hex.EncodeToString(sum[:12])

	f.mu.Lock()
	defer f.mu.Unlock()

	if ch, ok := f.charges[id]; ok {
		return ch, nil
	}

	code := "00020101021226950014br.gov.bcb.pix2573fake.abacatepay.local/" + id + "5204000053039865802BR6304FAKE"
	ch := Charge{
		ID: id, AmountCents: in.AmountCents, Status: StatusPending, DevMode: true,
		BrCode: code, BrCodeBase64: qrPattern(sum),
	}
	f.charges[id] = ch

	return ch, nil
}

// Check implements Gateway.
func (f *Fake) Check(_ context.Context, id string) (Charge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch, ok := f.charges[id]
	if !ok {
		return Charge{}, &APIError{Status: http.StatusNotFound, Message: "charge not found"}
	}

	return ch, nil
}

// SimulatePayment implements Gateway.
func (f *Fake) SimulatePayment(_ context.Context, id string) (Charge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch, ok := f.charges[id]
	if !ok {
		return Charge{}, &APIError{Status: http.StatusNotFound, Message: "charge not found"}
	}

	ch.Status = StatusPaid
	f.charges[id] = ch

	return ch, nil
}

// qrPattern draws a 25x25 module pattern from seed as a PNG data: URL.
func qrPattern(seed [32]byte) string {
	const modules, scale = 25, 8

	img := image.NewGray(image.Rect(0, 0, modules*scale, modules*scale))

	for y := range modules {
		for x := range modules {
			bit := seed[(y*modules+x)/8%len(seed)] >> ((y*modules + x) % 8) & 1

			c := color.Gray{Y: 255}
			if bit == 1 || finder(x, y, modules) {
				c = color.Gray{Y: 0}
			}

			for dy := range scale {
				for dx := range scale {
					img.SetGray(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img) // encoding an in-memory image cannot fail

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// finder reports whether (x, y) is on the border of a corner square, so the
// pattern looks like a QR code.
func finder(x, y, n int) bool {
	for _, c := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
		dx, dy := x-c[0], y-c[1]
		if dx >= 0 && dx < 7 && dy >= 0 && dy < 7 && (dx == 0 || dx == 6 || dy == 0 || dy == 6 || (dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4)) {
			return true
		}
	}

	return false
}
