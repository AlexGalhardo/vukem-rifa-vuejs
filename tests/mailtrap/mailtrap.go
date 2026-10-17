// Package mailtrap reads delivered emails from a Mailtrap Email Sandbox
// through its REST API, so integration and e2e tests can assert on real
// SMTP delivery (dev/staging) and follow verification or reset links.
//
// Configure with MAILTRAP_API_TOKEN and MAILTRAP_SANDBOX_ID; tests that
// need it call FromEnv, which skips when the token is unset.
package mailtrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// BaseURL is the Mailtrap API root.
const BaseURL = "https://mailtrap.io"

// Message is the subset of a sandbox message the tests use.
type Message struct {
	ID      int64  `json:"id"`
	Subject string `json:"subject"`
	ToEmail string `json:"to_email"`
	SentAt  string `json:"sent_at"`
	TxtPath string `json:"txt_path"`
}

// Client talks to one sandbox.
type Client struct {
	BaseURL   string
	Token     string
	SandboxID string
	HTTP      *http.Client
}

// FromEnv returns a client from MAILTRAP_API_TOKEN and MAILTRAP_SANDBOX_ID,
// or skips the test when they are unset.
func FromEnv(t testing.TB) *Client {
	t.Helper()

	token, sandbox := os.Getenv("MAILTRAP_API_TOKEN"), os.Getenv("MAILTRAP_SANDBOX_ID")
	if token == "" || sandbox == "" {
		t.Skip("MAILTRAP_API_TOKEN and MAILTRAP_SANDBOX_ID not set")
	}

	return &Client{BaseURL: BaseURL, Token: token, SandboxID: sandbox, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Api-Token", c.Token)
	req.Header.Set("Accept", "application/json")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("mailtrap %s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("mailtrap %s %s: status %d: %.200s", method, path, res.StatusCode, body)
	}

	if out == nil {
		return nil
	}

	if s, ok := out.(*string); ok {
		*s = string(body)
		return nil
	}

	return json.Unmarshal(body, out)
}

// Messages lists the newest messages, optionally filtered by search.
func (c *Client) Messages(ctx context.Context, search string) ([]Message, error) {
	path := "/api/sandboxes/" + url.PathEscape(c.SandboxID) + "/messages"
	if search != "" {
		path += "?search=" + url.QueryEscape(search)
	}

	var msgs []Message

	return msgs, c.do(ctx, http.MethodGet, path, &msgs)
}

// Text returns the plain-text body of a message.
func (c *Client) Text(ctx context.Context, m Message) (string, error) {
	path := m.TxtPath
	if path == "" {
		path = fmt.Sprintf("/api/sandboxes/%s/messages/%d/body.txt", url.PathEscape(c.SandboxID), m.ID)
	}

	var body string

	return body, c.do(ctx, http.MethodGet, path, &body)
}

// Clean deletes every message in the sandbox (run before a suite; the
// sandbox is rate limited, so keep email tests serial).
func (c *Client) Clean(ctx context.Context) error {
	return c.do(ctx, http.MethodPatch, "/api/sandboxes/"+url.PathEscape(c.SandboxID)+"/clean", nil)
}

var linkRe = regexp.MustCompile(`https?://[^\s)"<>]+`)

// ErrNoMail reports that no matching mail arrived before the deadline.
var ErrNoMail = errors.New("mailtrap: no matching mail")

// WaitLink polls until a mail to `to` arrives and returns its first link.
func (c *Client) WaitLink(ctx context.Context, to string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		msgs, err := c.Messages(ctx, to)
		if err != nil && ctx.Err() == nil {
			return "", err
		}

		for _, m := range msgs {
			if !strings.EqualFold(m.ToEmail, to) {
				continue
			}

			text, err := c.Text(ctx, m)
			if err != nil {
				return "", err
			}

			if link := linkRe.FindString(text); link != "" {
				return link, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("%w to %s", ErrNoMail, to)
		case <-time.After(2 * time.Second):
		}
	}
}
