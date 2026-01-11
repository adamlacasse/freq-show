package wikipedia

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_GetArtistBiography_FallsBackWhenDirectTitleIsNotArtist(t *testing.T) {
	// Simulate the edge case:
	// - /page/summary/Nirvana exists but is the religious concept.
	// - /page/summary/Nirvana%20(band) exists and is the biography we want.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/page/summary/Nirvana":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"type": "standard",
				"title": "Nirvana",
				"extract": "Nirvana is the ultimate goal of Buddhism, representing liberation from suffering."
			}`))
			return
		case "/page/summary/Nirvana (band)", "/page/summary/Nirvana%20(band)":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"type": "standard",
				"title": "Nirvana (band)",
				"extract": "Nirvana was an American rock band formed in Aberdeen, Washington, in 1987. Founded by Kurt Cobain and Krist Novoselic."
			}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	client, err := New(context.Background(), Config{
		BaseURL:   server.URL,
		UserAgent: "test",
		Timeout:   2 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	bio, err := client.GetArtistBiography(context.Background(), "Nirvana")
	if err != nil {
		t.Fatalf("GetArtistBiography error: %v", err)
	}

	if strings.Contains(strings.ToLower(bio), "buddh") || strings.Contains(strings.ToLower(bio), "liberation") {
		t.Fatalf("expected band biography, got concept bio: %q", bio)
	}
	if !strings.Contains(bio, "American rock band") {
		t.Fatalf("expected band biography, got: %q", bio)
	}
}
