package musicbrainz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

// newPagingTestServer serves a synthetic release-group collection of `total`
// items, honouring the limit/offset query params the client sends. It records
// every offset requested so tests can assert on the paging sequence.
func newPagingTestServer(t *testing.T, total int, offsets *[]int, mu *sync.Mutex) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		mu.Lock()
		*offsets = append(*offsets, offset)
		mu.Unlock()

		items := make([]string, 0, limit)
		for i := offset; i < offset+limit && i < total; i++ {
			items = append(items, fmt.Sprintf(
				`{"id":"rg-%d","title":"Album %d","primary-type":"Album","first-release-date":"1970-01-01"}`, i, i))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"release-groups":[%s],"release-group-count":%d,"release-group-offset":%d}`,
			strings.Join(items, ","), total, offset)
	}))
}

func newPagingTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := New(context.Background(), Config{
		BaseURL:     baseURL,
		AppName:     "freq-show-test",
		AppVersion:  "test",
		Contact:     "test@example.com",
		Timeout:     time.Second,
		MinInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return client
}

// A prolific artist spans several pages. Regression test for the truncation
// bug where only the first 50 release groups were ever retained.
func TestGetAllArtistReleaseGroupsPagesThroughAllResults(t *testing.T) {
	var (
		mu      sync.Mutex
		offsets []int
	)

	server := newPagingTestServer(t, 237, &offsets, &mu)
	defer server.Close()

	client := newPagingTestClient(t, server.URL)

	result, err := client.GetAllArtistReleaseGroups(context.Background(), "artist-id")
	if err != nil {
		t.Fatalf("GetAllArtistReleaseGroups returned error: %v", err)
	}

	if len(result.ReleaseGroups) != 237 {
		t.Fatalf("expected 237 release groups, got %d", len(result.ReleaseGroups))
	}
	if result.Count != 237 {
		t.Fatalf("expected count 237, got %d", result.Count)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []int{0, 100, 200}
	if len(offsets) != len(want) {
		t.Fatalf("expected offsets %v, got %v", want, offsets)
	}
	for i, off := range want {
		if offsets[i] != off {
			t.Fatalf("expected offsets %v, got %v", want, offsets)
		}
	}

	// Verify the tail of the collection actually survived, not just the count.
	if last := result.ReleaseGroups[236]; last.ID != "rg-236" {
		t.Fatalf("expected last release group rg-236, got %q", last.ID)
	}
}

// A single-page artist must not trigger a second upstream request.
func TestGetAllArtistReleaseGroupsStopsAfterSinglePage(t *testing.T) {
	var (
		mu      sync.Mutex
		offsets []int
	)

	server := newPagingTestServer(t, 12, &offsets, &mu)
	defer server.Close()

	client := newPagingTestClient(t, server.URL)

	result, err := client.GetAllArtistReleaseGroups(context.Background(), "artist-id")
	if err != nil {
		t.Fatalf("GetAllArtistReleaseGroups returned error: %v", err)
	}

	if len(result.ReleaseGroups) != 12 {
		t.Fatalf("expected 12 release groups, got %d", len(result.ReleaseGroups))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(offsets) != 1 {
		t.Fatalf("expected exactly 1 upstream request, got %d", len(offsets))
	}
}

// An exact multiple of the page size must not issue a trailing empty request.
func TestGetAllArtistReleaseGroupsHandlesExactPageBoundary(t *testing.T) {
	var (
		mu      sync.Mutex
		offsets []int
	)

	server := newPagingTestServer(t, 200, &offsets, &mu)
	defer server.Close()

	client := newPagingTestClient(t, server.URL)

	result, err := client.GetAllArtistReleaseGroups(context.Background(), "artist-id")
	if err != nil {
		t.Fatalf("GetAllArtistReleaseGroups returned error: %v", err)
	}

	if len(result.ReleaseGroups) != 200 {
		t.Fatalf("expected 200 release groups, got %d", len(result.ReleaseGroups))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(offsets) != 2 {
		t.Fatalf("expected exactly 2 upstream requests, got %d", len(offsets))
	}
}

// A failure partway through must surface as an error. Returning the partial
// slice would let the caller cache a truncated discography permanently.
func TestGetAllArtistReleaseGroupsFailsOnMidPageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 500 is not in isRetryableStatus, so this surfaces immediately.
		if r.URL.Query().Get("offset") != "0" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		items := make([]string, 0, 100)
		for i := 0; i < 100; i++ {
			items = append(items, fmt.Sprintf(`{"id":"rg-%d","title":"Album %d"}`, i, i))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"release-groups":[%s],"release-group-count":250,"release-group-offset":0}`,
			strings.Join(items, ","))
	}))
	defer server.Close()

	client := newPagingTestClient(t, server.URL)

	result, err := client.GetAllArtistReleaseGroups(context.Background(), "artist-id")
	if err == nil {
		t.Fatal("expected an error when a page fails, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result alongside error, got %d release groups", len(result.ReleaseGroups))
	}
}

// A count that overstates the collection must not spin the loop forever.
func TestGetAllArtistReleaseGroupsStopsOnEmptyPageDespiteInflatedCount(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		// Claim 10000 available but serve nothing past the first page.
		if offset > 0 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"release-groups":[],"release-group-count":10000,"release-group-offset":100}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"release-groups":[{"id":"rg-0","title":"Only"}],"release-group-count":10000,"release-group-offset":0}`)
	}))
	defer server.Close()

	client := newPagingTestClient(t, server.URL)

	result, err := client.GetAllArtistReleaseGroups(context.Background(), "artist-id")
	if err != nil {
		t.Fatalf("GetAllArtistReleaseGroups returned error: %v", err)
	}
	if len(result.ReleaseGroups) != 1 {
		t.Fatalf("expected 1 release group, got %d", len(result.ReleaseGroups))
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("expected 2 requests before bailing out, got %d", got)
	}
}

// Paging is not a consistent snapshot; a group appearing on two pages must
// only be counted once.
func TestGetAllArtistReleaseGroupsDeduplicatesAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		items := make([]string, 0, 100)
		for i := 0; i < 100; i++ {
			// Second page repeats the first page's final entry.
			id := offset + i
			if offset > 0 && i == 0 {
				id = 99
			}
			items = append(items, fmt.Sprintf(`{"id":"rg-%d","title":"Album %d"}`, id, id))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"release-groups":[%s],"release-group-count":200,"release-group-offset":%d}`,
			strings.Join(items, ","), offset)
	}))
	defer server.Close()

	client := newPagingTestClient(t, server.URL)

	result, err := client.GetAllArtistReleaseGroups(context.Background(), "artist-id")
	if err != nil {
		t.Fatalf("GetAllArtistReleaseGroups returned error: %v", err)
	}

	if len(result.ReleaseGroups) != 199 {
		t.Fatalf("expected 199 deduplicated release groups, got %d", len(result.ReleaseGroups))
	}

	seen := make(map[string]int)
	for _, rg := range result.ReleaseGroups {
		seen[rg.ID]++
		if seen[rg.ID] > 1 {
			t.Fatalf("release group %q appeared %d times", rg.ID, seen[rg.ID])
		}
	}
}

func TestGetAllArtistReleaseGroupsRejectsEmptyArtistID(t *testing.T) {
	client := newPagingTestClient(t, "http://example.invalid")

	if _, err := client.GetAllArtistReleaseGroups(context.Background(), "   "); err == nil {
		t.Fatal("expected an error for a blank artist id")
	}
}
