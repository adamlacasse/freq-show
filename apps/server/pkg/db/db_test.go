package db

import (
	"context"
	"testing"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

const (
	newStoreErrFmt = "NewMemoryStore returned error: %v"
	testArtistID   = "test-id"
)

func TestStoreSaveAndGetArtist(t *testing.T) {
	store, err := NewMemoryStore(context.Background())
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	artist := &data.Artist{
		ID:       testArtistID,
		Name:     "Test Artist",
		Genres:   []string{"rock"},
		Related:  []data.RelatedArtist{{ID: "other", Name: "Other Artist"}},
		Aliases:  []string{"Alias"},
		Albums:   []data.Album{{ID: "album-1", Tracks: []data.Track{{Number: 1, Title: "Intro"}}}},
		LifeSpan: data.LifeSpan{Begin: "2000-01-01"},
	}

	if err := store.SaveArtist(context.Background(), artist); err != nil {
		t.Fatalf("SaveArtist returned error: %v", err)
	}

	fetched, err := store.GetArtist(context.Background(), testArtistID)
	if err != nil {
		t.Fatalf("GetArtist returned error: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected artist, got nil")
	}
	if fetched.Name != artist.Name {
		t.Errorf("expected name %q, got %q", artist.Name, fetched.Name)
	}
	if len(fetched.Genres) != 1 || fetched.Genres[0] != "rock" {
		t.Errorf("expected genres to be preserved, got %#v", fetched.Genres)
	}

	// Mutate the fetched copy to ensure the stored record is not modified.
	fetched.Genres[0] = "pop"
	fetched.Related = append(fetched.Related, data.RelatedArtist{ID: "new", Name: "New Artist"})
	fetched.Aliases[0] = "Changed"
	fetched.Albums[0].Tracks[0].Title = "Changed"

	fetchedAgain, err := store.GetArtist(context.Background(), testArtistID)
	if err != nil {
		t.Fatalf("second GetArtist returned error: %v", err)
	}
	if fetchedAgain.Genres[0] != "rock" {
		t.Errorf("expected stored genres untouched, got %#v", fetchedAgain.Genres)
	}
	if len(fetchedAgain.Related) != 1 {
		t.Errorf("expected related slice untouched, got %#v", fetchedAgain.Related)
	}
	if fetchedAgain.Related[0].ID != "other" || fetchedAgain.Related[0].Name != "Other Artist" {
		t.Errorf("expected related artist untouched, got %#v", fetchedAgain.Related)
	}
	if fetchedAgain.Aliases[0] != "Alias" {
		t.Errorf("expected aliases untouched, got %#v", fetchedAgain.Aliases)
	}
	if fetchedAgain.Albums[0].Tracks[0].Title != "Intro" {
		t.Errorf("expected album tracks untouched, got %#v", fetchedAgain.Albums[0].Tracks)
	}
}

func TestStoreSaveArtistValidation(t *testing.T) {
	store, err := NewMemoryStore(context.Background())
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	if err := store.SaveArtist(context.Background(), nil); err == nil {
		t.Fatalf("expected error when saving nil artist")
	}

	if err := store.SaveArtist(context.Background(), &data.Artist{ID: ""}); err == nil {
		t.Fatalf("expected error when saving artist without ID")
	}
}

func TestStoreGetArtistMiss(t *testing.T) {
	store, err := NewMemoryStore(context.Background())
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	artist, err := store.GetArtist(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetArtist returned error: %v", err)
	}
	if artist != nil {
		t.Fatalf("expected nil for missing artist, got %#v", artist)
	}
}

func TestMemoryStoreAlbumCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	const (
		albumID  = "album-123"
		artistID = "artist-123"
	)

	album := &data.Album{ID: albumID, Title: "Album Title", ArtistID: artistID, SecondaryTypes: []string{"Live"}}

	if err := store.SaveAlbum(ctx, album); err != nil {
		t.Fatalf("SaveAlbum returned error: %v", err)
	}

	retrieved, err := store.GetAlbum(ctx, albumID)
	if err != nil {
		t.Fatalf("GetAlbum returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected album to be returned")
	}
	if retrieved == album {
		t.Error("expected a cloned album instance, got original reference")
	}
	retrieved.Title = "Changed"
	retrieved.SecondaryTypes[0] = "Studio"
	if stored := store.albums[albumID].Title; stored == "Changed" {
		t.Errorf("expected stored album to remain unchanged, got %q", stored)
	}
	if stored := store.albums[albumID].SecondaryTypes[0]; stored == "Studio" {
		t.Errorf("expected stored album secondary types to remain unchanged, got %q", stored)
	}
}

func TestMemoryStoreEmbeddingSaveAndGet(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	const (
		mbid  = "album-mbid-1"
		model = "voyage-3-lite"
	)
	vec := []float32{0.1, 0.2, 0.3, 0.4}

	if err := store.SaveEmbedding(ctx, mbid, model, vec); err != nil {
		t.Fatalf("SaveEmbedding returned error: %v", err)
	}

	got, err := store.GetEmbedding(ctx, mbid, model)
	if err != nil {
		t.Fatalf("GetEmbedding returned error: %v", err)
	}
	if len(got) != len(vec) {
		t.Fatalf("expected vector length %d, got %d", len(vec), len(got))
	}
	for i := range vec {
		if got[i] != vec[i] {
			t.Errorf("vec[%d]: expected %v, got %v", i, vec[i], got[i])
		}
	}

	// Mutate the returned slice; the stored copy must not change.
	got[0] = 99.0
	again, err := store.GetEmbedding(ctx, mbid, model)
	if err != nil {
		t.Fatalf("GetEmbedding (post-mutate) returned error: %v", err)
	}
	if again[0] != 0.1 {
		t.Errorf("expected stored vector unchanged after caller mutation, got %v", again[0])
	}

	// Upsert: same key, new vector — must overwrite, not append.
	newVec := []float32{1.0, 2.0, 3.0, 4.0}
	if err := store.SaveEmbedding(ctx, mbid, model, newVec); err != nil {
		t.Fatalf("SaveEmbedding (update) returned error: %v", err)
	}
	updated, err := store.GetEmbedding(ctx, mbid, model)
	if err != nil {
		t.Fatalf("GetEmbedding after update returned error: %v", err)
	}
	if updated[0] != 1.0 {
		t.Errorf("expected updated vector, got %v", updated[0])
	}
}

func TestMemoryStoreEmbeddingMiss(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	got, err := store.GetEmbedding(ctx, "missing", "voyage-3-lite")
	if err != nil {
		t.Fatalf("GetEmbedding returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing embedding, got %v", got)
	}
}

func TestMemoryStoreEmbeddingValidation(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	cases := []struct {
		name  string
		mbid  string
		model string
		vec   []float32
	}{
		{"empty mbid", "", "voyage-3-lite", []float32{0.1}},
		{"empty model", "x", "", []float32{0.1}},
		{"empty vec", "x", "voyage-3-lite", []float32{}},
		{"nil vec", "x", "voyage-3-lite", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := store.SaveEmbedding(ctx, tc.mbid, tc.model, tc.vec); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestMemoryStoreLoadAllForModel(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	saves := []struct {
		mbid  string
		model string
		vec   []float32
	}{
		{"a", "voyage-3-lite", []float32{0.1, 0.2}},
		{"b", "voyage-3-lite", []float32{0.3, 0.4}},
		{"c", "voyage-3-lite", []float32{0.5, 0.6}},
		{"a", "text-embedding-3-small", []float32{1, 2, 3}}, // same mbid, different model — coexists
	}
	for _, s := range saves {
		if err := store.SaveEmbedding(ctx, s.mbid, s.model, s.vec); err != nil {
			t.Fatalf("SaveEmbedding(%s, %s) failed: %v", s.mbid, s.model, err)
		}
	}

	records, err := store.LoadAllForModel(ctx, "voyage-3-lite")
	if err != nil {
		t.Fatalf("LoadAllForModel returned error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records for voyage-3-lite, got %d", len(records))
	}

	other, err := store.LoadAllForModel(ctx, "text-embedding-3-small")
	if err != nil {
		t.Fatalf("LoadAllForModel(other) returned error: %v", err)
	}
	if len(other) != 1 {
		t.Fatalf("expected 1 record for text-embedding-3-small, got %d", len(other))
	}

	missing, err := store.LoadAllForModel(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("LoadAllForModel(missing) returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing model, got %v", missing)
	}
}

func TestMemoryStoreDeleteOtherModels(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	for _, s := range []struct {
		mbid  string
		model string
	}{
		{"a", "voyage-3-lite"},
		{"b", "voyage-3-lite"},
		{"a", "text-embedding-3-small"},
		{"b", "text-embedding-3-small"},
		{"c", "all-MiniLM-L6-v2"},
	} {
		if err := store.SaveEmbedding(ctx, s.mbid, s.model, []float32{0.1}); err != nil {
			t.Fatalf("SaveEmbedding setup failed: %v", err)
		}
	}

	deleted, err := store.DeleteOtherModels(ctx, "voyage-3-lite")
	if err != nil {
		t.Fatalf("DeleteOtherModels returned error: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 deletions, got %d", deleted)
	}

	// Survivors are voyage-3-lite only.
	for _, model := range []string{"text-embedding-3-small", "all-MiniLM-L6-v2"} {
		records, err := store.LoadAllForModel(ctx, model)
		if err != nil {
			t.Fatalf("LoadAllForModel(%s) returned error: %v", model, err)
		}
		if records != nil {
			t.Errorf("expected no records for purged model %s, got %d", model, len(records))
		}
	}

	survivors, err := store.LoadAllForModel(ctx, "voyage-3-lite")
	if err != nil {
		t.Fatalf("LoadAllForModel(survivors) returned error: %v", err)
	}
	if len(survivors) != 2 {
		t.Errorf("expected 2 voyage-3-lite survivors, got %d", len(survivors))
	}
}
