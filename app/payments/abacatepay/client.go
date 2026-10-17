// Package abacatepay is a small client for the AbacatePay API v2 PIX
// endpoints the raffle needs (transparent checkout create, check and the
// dev-mode simulate-payment) plus webhook parsing. Every response is
// validated with z before use (PLAN.md D7).
package abacatepay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/z"
)

// DefaultBaseURL is the single AbacatePay server for production and dev
// mode (the API key decides the environment).
const DefaultBaseURL = "https://api.abacatepay.com/v2"

// DefaultTimeout bounds every API call.
const DefaultTimeout = 10 * time.Second

// maxResponse bounds the response body read from the API.
const maxResponse = 1 << 20

// PIX charge statuses (GET /transparents/check).
const (
	StatusPending      = "PENDING"
	StatusPaid         = "PAID"
	StatusExpired      = "EXPIRED"
	StatusCancelled    = "CANCELLED"
	StatusUnderDispute = "UNDER_DISPUTE"
	StatusRefunded     = "REFUNDED"
	StatusRedeemed     = "REDEEMED"
	StatusApproved     = "APPROVED"
	StatusFailed       = "FAILED"
)

var statuses = []string{
	StatusPending, StatusPaid, StatusExpired, StatusCancelled, StatusUnderDispute,
	StatusRefunded, StatusRedeemed, StatusApproved, StatusFailed,
}

// PixRequest is a PIX charge to create. Amounts are integer centavos.
type PixRequest struct {
	AmountCents int64
	// ExpiresIn is the QR code lifetime in seconds.
	ExpiresIn   int
	Description string
	// ExternalID is our reference (the reservation id).
	ExternalID string
}

// Charge is a PIX charge as returned by the API.
type Charge struct {
	ID          string
	AmountCents int64
	Status      string
	DevMode     bool
	// BrCode is the copy-and-paste PIX code.
	BrCode string
	// BrCodeBase64 is the QR code image as a data: URL.
	BrCodeBase64 string
}

// Gateway creates and inspects PIX charges. Client talks to AbacatePay;
// Fake is the deterministic stand-in for development and e2e.
type Gateway interface {
	CreatePix(ctx context.Context, in PixRequest) (Charge, error)
	Check(ctx context.Context, id string) (Charge, error)
	SimulatePayment(ctx context.Context, id string) (Charge, error)
}

// APIError is a non-successful API answer.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("abacatepay: HTTP %d: %s", e.Status, e.Message)
}

// ErrInvalidResponse wraps responses that fail validation.
var ErrInvalidResponse = errors.New("abacatepay: invalid response")

// Client is the AbacatePay HTTP client.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

var _ Gateway = (*Client)(nil)

// New returns a client for baseURL (DefaultBaseURL when empty). hc nil uses
// an http.Client with DefaultTimeout.
func New(baseURL, apiKey string, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	if hc == nil {
		hc = &http.Client{Timeout: DefaultTimeout}
	}

	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: hc}
}

// BaseURL returns the API base URL.
func (c *Client) BaseURL() string { return c.baseURL }

type pixBody struct {
	Method string  `json:"method"`
	Data   pixData `json:"data"`
}

type pixData struct {
	Amount      int64  `json:"amount"`
	ExpiresIn   int    `json:"expiresIn,omitempty"`
	Description string `json:"description,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreatePix creates a PIX charge (POST /transparents/create).
func (c *Client) CreatePix(ctx context.Context, in PixRequest) (Charge, error) {
	body, err := json.Marshal(pixBody{Method: "PIX", Data: pixData{
		Amount: in.AmountCents, ExpiresIn: in.ExpiresIn, Description: in.Description, ExternalID: in.ExternalID,
	}})
	if err != nil {
		return Charge{}, err
	}

	data, err := c.do(ctx, http.MethodPost, "/transparents/create", nil, body)
	if err != nil {
		return Charge{}, fmt.Errorf("create pix: %w", err)
	}

	return parseCharge(data)
}

// Check returns the status of a PIX charge (GET /transparents/check).
func (c *Client) Check(ctx context.Context, id string) (Charge, error) {
	data, err := c.do(ctx, http.MethodGet, "/transparents/check", url.Values{"id": {id}}, nil)
	if err != nil {
		return Charge{}, fmt.Errorf("check pix: %w", err)
	}

	st, err := checkSchema.Parse(data)
	if err != nil {
		return Charge{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	return Charge{ID: st.ID, Status: st.Status}, nil
}

// SimulatePayment pays a dev-mode PIX charge
// (POST /transparents/simulate-payment); production keys get an error.
func (c *Client) SimulatePayment(ctx context.Context, id string) (Charge, error) {
	data, err := c.do(ctx, http.MethodPost, "/transparents/simulate-payment", url.Values{"id": {id}}, []byte(`{"metadata":{}}`))
	if err != nil {
		return Charge{}, fmt.Errorf("simulate payment: %w", err)
	}

	return parseCharge(data)
}

type chargeDTO struct {
	ID           string `json:"id"`
	Amount       int64  `json:"amount"`
	Status       string `json:"status"`
	DevMode      bool   `json:"devMode"`
	BrCode       string `json:"brCode"`
	BrCodeBase64 string `json:"brCodeBase64"`
}

type checkDTO struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// The QR image is rendered as <img src>, so only a raster data: URL is
// accepted (CSP img-src 'self' data:).
var chargeSchema = z.Struct[chargeDTO](z.Shape{
	"id":           z.String().Min(1).Max(191),
	"amount":       z.Int64().Positive(),
	"status":       z.String().OneOf(statuses...),
	"devMode":      z.Bool().Optional(),
	"brCode":       z.String().Min(1).Max(1024),
	"brCodeBase64": z.String().StartsWith("data:image/png;base64,").Max(512 * 1024),
})

var checkSchema = z.Struct[checkDTO](z.Shape{
	"id":     z.String().Min(1).Max(191),
	"status": z.String().OneOf(statuses...),
})

func parseCharge(data any) (Charge, error) {
	ch, err := chargeSchema.Parse(data)
	if err != nil {
		return Charge{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	return Charge{
		ID: ch.ID, AmountCents: ch.Amount, Status: ch.Status, DevMode: ch.DevMode,
		BrCode: ch.BrCode, BrCodeBase64: ch.BrCodeBase64,
	}, nil
}

// envelope is every API answer: {"data": …, "success": bool, "error": …}.
type envelope struct {
	Data    any  `json:"data"`
	Success bool `json:"success"`
	Error   any  `json:"error"`
}

// do sends the request and returns the decoded "data" of a successful
// envelope.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte) (any, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, rd)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponse))
	if err != nil {
		return nil, err
	}

	var env envelope

	decodeErr := json.Unmarshal(raw, &env)

	if res.StatusCode >= http.StatusMultipleChoices || (decodeErr == nil && !env.Success) {
		msg := http.StatusText(res.StatusCode)
		if s, ok := env.Error.(string); ok && s != "" {
			msg = s
		}

		return nil, &APIError{Status: res.StatusCode, Message: truncate(msg, 200)}
	}

	if decodeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, decodeErr)
	}

	return env.Data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n]
}
