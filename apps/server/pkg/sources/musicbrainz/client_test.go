package musicbrainz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientSpacesRequestsAcrossCalls(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []time.Time
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, time.Now())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":[],"offset":0,"count":0}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), Config{
		BaseURL:     server.URL,
		AppName:     "freq-show-test",
		AppVersion:  "test",
		Contact:     "test@example.com",
		Timeout:     time.Second,
		MinInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	var wg sync.WaitGroup
	for _, query := range []string{"radiohead", "bjork"} {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			if _, err := client.SearchArtists(context.Background(), query, 10, 0); err != nil {
				t.Errorf("SearchArtists returned error: %v", err)
			}
		}(query)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if len(requests) != 2 {
		t.Fatalf("expected 2 upstream requests, got %d", len(requests))
	}

	if delta := requests[1].Sub(requests[0]); delta < 45*time.Millisecond {
		t.Fatalf("expected requests to be spaced by at least 45ms, got %s", delta)
	}
}

func TestClientRetriesRetryableResponses(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":[],"offset":0,"count":0}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), Config{
		BaseURL:     server.URL,
		AppName:     "freq-show-test",
		AppVersion:  "test",
		Contact:     "test@example.com",
		Timeout:     time.Second,
		MinInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := client.SearchArtists(context.Background(), "massive attack", 10, 0); err != nil {
		t.Fatalf("SearchArtists returned error: %v", err)
	}

	if got := attempts.Load(); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestClientReturnsRateLimitAfterRetryableFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := New(context.Background(), Config{
		BaseURL:     server.URL,
		AppName:     "freq-show-test",
		AppVersion:  "test",
		Contact:     "test@example.com",
		Timeout:     time.Second,
		MinInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := client.SearchArtists(context.Background(), "portishead", 10, 0); !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}
