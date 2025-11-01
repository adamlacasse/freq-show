package reviews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		UserAgent: "TestAgent/1.0",
		Timeout:   5 * time.Second,
	}

	client := NewClient(cfg)
	if client == nil {
		t.Fatal("Expected client to be created")
	}
	if client.userAgent != "TestAgent/1.0" {
		t.Errorf("Expected user agent %q, got %q", "TestAgent/1.0", client.userAgent)
	}
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient(Config{})

	if client.userAgent == "" {
		t.Error("Expected default user agent to be set")
	}
	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", client.httpClient.Timeout)
	}
}

func TestDiscogsClient_SearchAlbum(t *testing.T) {
	// Mock server for Discogs API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/database/search" {
			t.Errorf("Expected path /database/search, got %s", r.URL.Path)
		}

		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Expected User-Agent header to be set")
		}

		// Mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"results": [
				{
					"id": 249504,
					"type": "release",
					"title": "Nevermind",
					"resource_url": "https://api.discogs.com/releases/249504",
					"year": "1991",
					"community": {
						"have": 15234,
						"want": 1234,
						"rating": {
							"count": 1000,
							"average": 4.5
						}
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := &DiscogsClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		userAgent:  "Test/1.0",
		baseURL:    server.URL,
	}

	ctx := context.Background()
	results, err := client.searchAlbum(ctx, "Nirvana", "Nevermind")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.ID != 249504 {
		t.Errorf("Expected ID 249504, got %d", result.ID)
	}
	if result.Title != "Nevermind" {
		t.Errorf("Expected title 'Nevermind', got %q", result.Title)
	}
	if result.Community.Rating.Average != 4.5 {
		t.Errorf("Expected rating 4.5, got %f", result.Community.Rating.Average)
	}
}

func TestDiscogsClient_GetRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/249504" {
			t.Errorf("Expected path /releases/249504, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 249504,
			"title": "Nevermind",
			"artists": [
				{
					"name": "Nirvana",
					"id": 109713
				}
			],
			"community": {
				"have": 15234,
				"want": 1234,
				"rating": {
					"count": 1000,
					"average": 4.5
				}
			},
			"notes": "Groundbreaking grunge album that defined a generation."
		}`))
	}))
	defer server.Close()

	client := &DiscogsClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		userAgent:  "Test/1.0",
		baseURL:    server.URL,
	}

	ctx := context.Background()
	release, err := client.getRelease(ctx, 249504)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if release.ID != 249504 {
		t.Errorf("Expected ID 249504, got %d", release.ID)
	}
	if release.Title != "Nevermind" {
		t.Errorf("Expected title 'Nevermind', got %q", release.Title)
	}
	if release.Notes != "Groundbreaking grunge album that defined a generation." {
		t.Errorf("Expected notes to be set, got %q", release.Notes)
	}
}

func TestDiscogsClient_ConvertToReview(t *testing.T) {
	release := &DiscogsRelease{
		ID:    249504,
		Title: "Nevermind",
		Community: DiscogsCommunityStat{
			Have: 15234,
			Want: 1234,
			Rating: DiscogsRating{
				Count:   1000,
				Average: 4.5,
			},
		},
		Notes: "Groundbreaking grunge album that defined a generation.",
	}

	client := &DiscogsClient{}
	review := client.convertToReview(release)

	if review.Source != "Discogs" {
		t.Errorf("Expected source 'Discogs', got %q", review.Source)
	}
	if review.Rating != 4.5 {
		t.Errorf("Expected rating 4.5, got %f", review.Rating)
	}
	if review.Text != "Groundbreaking grunge album that defined a generation." {
		t.Errorf("Expected text to be notes, got %q", review.Text)
	}
	if review.Author != "Community" {
		t.Errorf("Expected author 'Community', got %q", review.Author)
	}
	expectedURL := "https://www.discogs.com/release/249504"
	if review.URL != expectedURL {
		t.Errorf("Expected URL %q, got %q", expectedURL, review.URL)
	}
}

func TestDiscogsClient_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &DiscogsClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		userAgent:  "Test/1.0",
		baseURL:    server.URL,
	}

	ctx := context.Background()
	_, err := client.searchAlbum(ctx, "NonExistent", "Album")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestGetAlbumReview_Integration(t *testing.T) {
	// Mock server that handles both search and release requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.URL.Path == "/database/search" {
			w.Write([]byte(`{
				"results": [
					{
						"id": 249504,
						"type": "release",
						"title": "Nevermind"
					}
				]
			}`))
		} else if r.URL.Path == "/releases/249504" {
			w.Write([]byte(`{
				"id": 249504,
				"title": "Nevermind",
				"community": {
					"have": 15234,
					"want": 1234,
					"rating": {
						"count": 1000,
						"average": 4.5
					}
				},
				"notes": "Groundbreaking album."
			}`))
		}
	}))
	defer server.Close()

	cfg := Config{
		UserAgent: "Test/1.0",
		Timeout:   5 * time.Second,
	}

	client := NewClient(cfg)
	client.discogs.baseURL = server.URL

	ctx := context.Background()
	review, err := client.GetAlbumReview(ctx, "Nirvana", "Nevermind")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if review.Source != "Discogs" {
		t.Errorf("Expected source 'Discogs', got %q", review.Source)
	}
	if review.Rating != 4.5 {
		t.Errorf("Expected rating 4.5, got %f", review.Rating)
	}
	if review.Text != "Groundbreaking album." {
		t.Errorf("Expected review text, got %q", review.Text)
	}
}
