package embeddings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestVoyageEncodeUsesQueryAndBatchUsesDocument(t *testing.T) {
	var inputTypes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer auth, got %q", got)
		}
		var body voyageEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		inputTypes = append(inputTypes, body.InputType)
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[0.1,0.2]}]}`))
	}))
	defer server.Close()

	client, err := NewFromConfig(Config{
		Provider: "voyage",
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("NewFromConfig returned error: %v", err)
	}

	if _, err := client.Encode(context.Background(), "late-night dub"); err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if _, err := client.EncodeBatch(context.Background(), []string{"record metadata"}); err != nil {
		t.Fatalf("EncodeBatch returned error: %v", err)
	}

	if len(inputTypes) != 2 || inputTypes[0] != "query" || inputTypes[1] != "document" {
		t.Fatalf("unexpected voyage input types: %#v", inputTypes)
	}
}

func TestOpenAIEncodeBatchParsesIndexedEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Model != DefaultOpenAIModel {
			t.Fatalf("expected default model, got %q", body.Model)
		}
		if body.EncodingFormat != "float" {
			t.Fatalf("expected float encoding, got %q", body.EncodingFormat)
		}
		_, _ = w.Write([]byte(`{"data":[{"index":1,"embedding":[0.3,0.4]},{"index":0,"embedding":[0.1,0.2]}]}`))
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

	vecs, err := client.EncodeBatch(context.Background(), []string{"one", "two"})
	if err != nil {
		t.Fatalf("EncodeBatch returned error: %v", err)
	}
	if len(vecs) != 2 || vecs[0][0] != float32(0.1) || vecs[1][0] != float32(0.3) {
		t.Fatalf("unexpected vectors: %#v", vecs)
	}
}

func TestHFEncodeRetriesModelLoadingOnce(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/sentence-transformers/all-MiniLM-L6-v2") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"estimated_time":0.001}`))
			return
		}
		_, _ = w.Write([]byte(`[[0.1,0.2,0.3]]`))
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	vec, err := client.Encode(ctx, "hello")
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(vec))
	}
	if attempts.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestRateLimitIsTyped(t *testing.T) {
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

	if _, err := client.Encode(context.Background(), "hello"); !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}
