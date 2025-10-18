package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/adamlacasse/freq-show/pkg/data"
)

const (
	sqliteTestID      = "sqlite-test"
	sqliteDBName      = "freqshow.db"
	sqliteQuerySuffix = "?_fk=1"
	sqliteNewErrFmt   = "NewSQLiteStore returned error: %v"
	sqliteCloseErrFmt = "Close returned error: %v"
	sqliteAlbumID     = "album-1"
)

func TestSQLiteStoreSaveAndGetArtist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, sqliteDBName) + sqliteQuerySuffix

	store, err := NewSQLiteStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf(sqliteNewErrFmt, err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf(sqliteCloseErrFmt, err)
		}
	}()

	artist := &data.Artist{ID: sqliteTestID, Name: "SQLite Artist"}
	if err := store.SaveArtist(context.Background(), artist); err != nil {
		t.Fatalf("SaveArtist returned error: %v", err)
	}

	fetched, err := store.GetArtist(context.Background(), sqliteTestID)
	if err != nil {
		t.Fatalf("GetArtist returned error: %v", err)
	}
	if fetched == nil || fetched.Name != "SQLite Artist" {
		t.Fatalf("unexpected artist payload: %#v", fetched)
	}

	artist.Name = "Updated"
	if err := store.SaveArtist(context.Background(), artist); err != nil {
		t.Fatalf("SaveArtist (update) returned error: %v", err)
	}

	updated, err := store.GetArtist(context.Background(), sqliteTestID)
	if err != nil {
		t.Fatalf("GetArtist after update returned error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
}

func TestSQLiteStoreMissingArtist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, sqliteDBName) + sqliteQuerySuffix

	store, err := NewSQLiteStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf(sqliteNewErrFmt, err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf(sqliteCloseErrFmt, err)
		}
	}()

	artist, err := store.GetArtist(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetArtist returned error: %v", err)
	}
	if artist != nil {
		t.Fatalf("expected nil for missing artist, got %#v", artist)
	}
}

func TestSQLiteStoreSaveAndGetAlbum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, sqliteDBName) + sqliteQuerySuffix

	store, err := NewSQLiteStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf(sqliteNewErrFmt, err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf(sqliteCloseErrFmt, err)
		}
	}()

	album := &data.Album{ID: sqliteAlbumID, Title: "SQLite Album", ArtistID: "artist-1"}
	if err := store.SaveAlbum(context.Background(), album); err != nil {
		t.Fatalf("SaveAlbum returned error: %v", err)
	}

	fetched, err := store.GetAlbum(context.Background(), sqliteAlbumID)
	if err != nil {
		t.Fatalf("GetAlbum returned error: %v", err)
	}
	if fetched == nil || fetched.Title != "SQLite Album" {
		t.Fatalf("unexpected album payload: %#v", fetched)
	}

	album.Title = "Updated"
	if err := store.SaveAlbum(context.Background(), album); err != nil {
		t.Fatalf("SaveAlbum (update) returned error: %v", err)
	}

	updated, err := store.GetAlbum(context.Background(), sqliteAlbumID)
	if err != nil {
		t.Fatalf("GetAlbum after update returned error: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}
}
