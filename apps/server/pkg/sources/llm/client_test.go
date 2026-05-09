package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOpenAIChatCompleteSendsMessagesAndParsesContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Model != DefaultOpenAIModel {
			t.Fatalf("expected default model, got %q", body.Model)
		}
		if len(body.Messages) != 2 || body.Messages[0].Role != "system" || body.Messages[1].Role != "user" {
			t.Fatalf("unexpected messages: %#v", body.Messages)
		}
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[{"message":{"content":"try this record"}}]}`))
	}))
	defer server.Close()

	client, err := NewFromConfig(Config{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("NewFromConfig returned error: %v", err)
	}

	resp, err := client.ChatComplete(context.Background(), ChatRequest{
		SystemPrompt: "system",
		UserPrompt:   "user",
		MaxTokens:    50,
	})
	if err != nil {
		t.Fatalf("ChatComplete returned error: %v", err)
	}
	if resp.Content != "try this record" || resp.Model != "gpt-test" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestHFChatCompleteFallsBackAndPinsWorkingModel(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		switch attempts.Add(1) {
		case 1:
			if body.Model != DefaultHFModelCandidates[0] {
				t.Fatalf("expected first candidate, got %q", body.Model)
			}
			w.WriteHeader(http.StatusNotFound)
		case 2, 3:
			if body.Model != DefaultHFModelCandidates[1] {
				t.Fatalf("expected pinned second candidate, got %q", body.Model)
			}
			_, _ = w.Write([]byte(`{"model":"qwen-routed","choices":[{"message":{"content":"ok"}}]}`))
		default:
			t.Fatalf("unexpected extra attempt")
		}
	}))
	defer server.Close()

	client, err := NewFromConfig(Config{
		Provider: "huggingface",
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("NewFromConfig returned error: %v", err)
	}

	req := ChatRequest{UserPrompt: "hello", MaxTokens: 20}
	if _, err := client.ChatComplete(context.Background(), req); err != nil {
		t.Fatalf("first ChatComplete returned error: %v", err)
	}
	if _, err := client.ChatComplete(context.Background(), req); err != nil {
		t.Fatalf("second ChatComplete returned error: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
	if model := client.Model(); model != DefaultHFModelCandidates[1] {
		t.Fatalf("expected pinned model %q, got %q", DefaultHFModelCandidates[1], model)
	}
}

func TestAnthropicChatCompleteSendsSystemAndParsesTextBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("expected api key header, got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Fatalf("expected anthropic version, got %q", got)
		}
		var body anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.System != "system" || len(body.Messages) != 1 || body.Messages[0].Content != "user" {
			t.Fatalf("unexpected request body: %#v", body)
		}
		_, _ = w.Write([]byte(`{"model":"claude-test","content":[{"type":"text","text":"first "},{"type":"text","text":"second"}]}`))
	}))
	defer server.Close()

	client, err := NewFromConfig(Config{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("NewFromConfig returned error: %v", err)
	}

	resp, err := client.ChatComplete(context.Background(), ChatRequest{
		SystemPrompt: "system",
		UserPrompt:   "user",
		MaxTokens:    50,
	})
	if err != nil {
		t.Fatalf("ChatComplete returned error: %v", err)
	}
	if resp.Content != "first second" || resp.Model != "claude-test" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestLLMRateLimitIsTyped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := NewFromConfig(Config{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("NewFromConfig returned error: %v", err)
	}

	_, err = client.ChatComplete(context.Background(), ChatRequest{UserPrompt: "hello", MaxTokens: 10})
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}
