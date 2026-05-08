package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
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

func TestSQLiteStoreSaveAndGetEmbedding(t *testing.T) {
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

	const (
		mbid  = "album-emb-1"
		model = "voyage-3-lite"
	)
	vec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	if err := store.SaveEmbedding(context.Background(), mbid, model, vec); err != nil {
		t.Fatalf("SaveEmbedding returned error: %v", err)
	}

	got, err := store.GetEmbedding(context.Background(), mbid, model)
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

	// Upsert with a different vector for the same (mbid, model) pair.
	newVec := []float32{1.0, 2.0, 3.0}
	if err := store.SaveEmbedding(context.Background(), mbid, model, newVec); err != nil {
		t.Fatalf("SaveEmbedding (update) returned error: %v", err)
	}
	updated, err := store.GetEmbedding(context.Background(), mbid, model)
	if err != nil {
		t.Fatalf("GetEmbedding after update returned error: %v", err)
	}
	if len(updated) != len(newVec) {
		t.Fatalf("expected updated length %d, got %d", len(newVec), len(updated))
	}
	if updated[0] != 1.0 {
		t.Errorf("expected updated[0]=1.0, got %v", updated[0])
	}

	// Different model coexists for the same mbid.
	otherVec := []float32{9.0, 9.0, 9.0, 9.0, 9.0, 9.0}
	if err := store.SaveEmbedding(context.Background(), mbid, "text-embedding-3-small", otherVec); err != nil {
		t.Fatalf("SaveEmbedding (other model) returned error: %v", err)
	}
	mainStill, err := store.GetEmbedding(context.Background(), mbid, model)
	if err != nil {
		t.Fatalf("GetEmbedding voyage after other-model save returned error: %v", err)
	}
	if mainStill[0] != 1.0 {
		t.Errorf("expected voyage row untouched after other-model save, got %v", mainStill[0])
	}
}

func TestSQLiteStoreEmbeddingMiss(t *testing.T) {
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

	got, err := store.GetEmbedding(context.Background(), "missing", "voyage-3-lite")
	if err != nil {
		t.Fatalf("GetEmbedding returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing embedding, got %v", got)
	}
}

func TestSQLiteStoreLoadAllForModelAndPrune(t *testing.T) {
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

	for _, s := range []struct {
		mbid, model string
		vec         []float32
	}{
		{"a", "voyage-3-lite", []float32{0.1, 0.2}},
		{"b", "voyage-3-lite", []float32{0.3, 0.4}},
		{"a", "text-embedding-3-small", []float32{1, 2, 3}},
		{"c", "all-MiniLM-L6-v2", []float32{9, 9, 9, 9}},
	} {
		if err := store.SaveEmbedding(context.Background(), s.mbid, s.model, s.vec); err != nil {
			t.Fatalf("SaveEmbedding(%s,%s) failed: %v", s.mbid, s.model, err)
		}
	}

	voy, err := store.LoadAllForModel(context.Background(), "voyage-3-lite")
	if err != nil {
		t.Fatalf("LoadAllForModel returned error: %v", err)
	}
	if len(voy) != 2 {
		t.Fatalf("expected 2 voyage rows, got %d", len(voy))
	}
	// Vector contents survive the round trip.
	for _, r := range voy {
		if len(r.Vec) != 2 {
			t.Errorf("expected 2-dim vec for %s, got %d", r.MBID, len(r.Vec))
		}
	}

	deleted, err := store.DeleteOtherModels(context.Background(), "voyage-3-lite")
	if err != nil {
		t.Fatalf("DeleteOtherModels returned error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deletions, got %d", deleted)
	}

	// Survivors are voyage rows only.
	survivors, err := store.LoadAllForModel(context.Background(), "voyage-3-lite")
	if err != nil {
		t.Fatalf("LoadAllForModel(survivors) returned error: %v", err)
	}
	if len(survivors) != 2 {
		t.Errorf("expected 2 voyage survivors, got %d", len(survivors))
	}
	for _, m := range []string{"text-embedding-3-small", "all-MiniLM-L6-v2"} {
		gone, err := store.LoadAllForModel(context.Background(), m)
		if err != nil {
			t.Fatalf("LoadAllForModel(%s) returned error: %v", m, err)
		}
		if len(gone) != 0 {
			t.Errorf("expected no rows for purged model %s, got %d", m, len(gone))
		}
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
