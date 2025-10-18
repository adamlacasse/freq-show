package db

import (
	"context"
	"testing"

	"github.com/adamlacasse/freq-show/pkg/data"
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
		Related:  []string{"other"},
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
	fetched.Related = append(fetched.Related, "new")
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
