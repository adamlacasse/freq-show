// Package resend sends transactional email via the Resend API
// (https://resend.com/docs/api-reference/emails/send-email). Used to
// deliver magic-link sign-in emails; Resend's free tier is enough for the
// expected volume.
package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.resend.com"

// Client sends email through the Resend API.
type Client struct {
	httpClient *http.Client
	apiKey     string
	from       string
	baseURL    string
}

// Config configures a Client.
type Config struct {
	// APIKey is the Resend API key (RESEND_API_KEY).
	APIKey string
	// From is the sender address, e.g. "FreqShow <onboarding@resend.dev>".
	// Must be a verified sender/domain in Resend, with the exception of
	// Resend's own onboarding@resend.dev test sender.
	From    string
	Timeout time.Duration
	// BaseURL overrides the Resend API base URL. Empty uses the default
	// production endpoint; tests point this at an httptest server.
	BaseURL string
}

// New constructs a Resend client. Returns an error if APIKey or From is
// empty, since every send would otherwise fail at request time anyway.
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("resend: api key required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return nil, errors.New("resend: from address required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		apiKey:     cfg.APIKey,
		from:       cfg.From,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}, nil
}

type sendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text"`
}

// SendMagicLink emails a login link to toEmail.
func (c *Client) SendMagicLink(ctx context.Context, toEmail, link string) error {
	if strings.TrimSpace(toEmail) == "" {
		return errors.New("resend: recipient email required")
	}
	if strings.TrimSpace(link) == "" {
		return errors.New("resend: link required")
	}

	body := sendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: "Your FreqShow sign-in link",
		HTML:    magicLinkHTML(link),
		Text:    magicLinkText(link),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("resend: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emails", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("resend: api error (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func magicLinkText(link string) string {
	return "Sign in to FreqShow!:\n\n" + link +
		"\n\nThis link expires in 15 minutes. If you didn't request it, you can safely ignore this email."
}

func magicLinkHTML(link string) string {
	return fmt.Sprintf(
		`<p>Sign in to FreqShow!:</p><p><a href="%s">%s</a></p><p>This link expires in 15 minutes. If you didn't request it, you can safely ignore this email.</p>`,
		link, link,
	)
}
