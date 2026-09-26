package resend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewRequiresAPIKeyAndFrom(t *testing.T) {
	if _, err := New(Config{From: "a@example.com"}); err == nil {
		t.Fatal("expected error when APIKey is missing")
	}
	if _, err := New(Config{APIKey: "key"}); err == nil {
		t.Fatal("expected error when From is missing")
	}
	if _, err := New(Config{APIKey: "key", From: "a@example.com"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendMagicLinkSendsExpectedRequest(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotMethod string
		gotBody   sendEmailRequest
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email-1"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		APIKey:  "test-key",
		From:    "FreqShow <onboarding@resend.dev>",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = client.SendMagicLink(context.Background(), "user@example.com", "https://api.example.com/auth/verify?token=abc123")
	if err != nil {
		t.Fatalf("SendMagicLink returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/emails" {
		t.Errorf("expected path /emails, got %s", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("expected bearer auth header, got %q", gotAuth)
	}
	if gotBody.From != "FreqShow <onboarding@resend.dev>" {
		t.Errorf("unexpected from address: %q", gotBody.From)
	}
	if len(gotBody.To) != 1 || gotBody.To[0] != "user@example.com" {
		t.Errorf("unexpected recipients: %#v", gotBody.To)
	}
	if gotBody.Subject == "" {
		t.Error("expected a non-empty subject")
	}
	if !strings.Contains(gotBody.HTML, "https://api.example.com/auth/verify?token=abc123") {
		t.Errorf("expected HTML body to contain the link, got %q", gotBody.HTML)
	}
	if !strings.Contains(gotBody.Text, "https://api.example.com/auth/verify?token=abc123") {
		t.Errorf("expected text body to contain the link, got %q", gotBody.Text)
	}
}

func TestSendMagicLinkSurfacesAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "bad-key", From: "a@example.com", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = client.SendMagicLink(context.Background(), "user@example.com", "https://api.example.com/auth/verify?token=abc123")
	if err == nil {
		t.Fatal("expected an error for a non-2xx response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention the status code, got %v", err)
	}
}

func TestSendMagicLinkRequiresRecipientAndLink(t *testing.T) {
	client, err := New(Config{APIKey: "key", From: "a@example.com"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := client.SendMagicLink(context.Background(), "", "https://example.com/verify"); err == nil {
		t.Fatal("expected error for empty recipient")
	}
	if err := client.SendMagicLink(context.Background(), "user@example.com", ""); err == nil {
		t.Fatal("expected error for empty link")
	}
}
