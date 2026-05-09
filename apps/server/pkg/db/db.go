package db

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

// ArtistRepository defines persistence operations for artist entities.
type ArtistRepository interface {
	GetArtist(ctx context.Context, id string) (*data.Artist, error)
	SaveArtist(ctx context.Context, artist *data.Artist) error
}

// AlbumRepository defines persistence operations for album entities.
type AlbumRepository interface {
	GetAlbum(ctx context.Context, id string) (*data.Album, error)
	SaveAlbum(ctx context.Context, album *data.Album) error
	ListAlbumsMissingEmbedding(ctx context.Context, model string, limit int) ([]data.Album, error)
}

// EmbeddingRepository defines persistence operations for album embedding vectors.
// The `(mbid, model)` composite key lets multiple model versions coexist in
// the table during a rolling reindex.
type EmbeddingRepository interface {
	GetEmbedding(ctx context.Context, mbid, model string) ([]float32, error)
	SaveEmbedding(ctx context.Context, mbid, model string, vec []float32) error
	LoadAllForModel(ctx context.Context, model string) ([]EmbeddingRecord, error)
	DeleteOtherModels(ctx context.Context, keepModel string) (int, error)
}

// EmbeddingRecord is a single (mbid, vector) pair as returned by LoadAllForModel.
// The model name is implicit in the query that produced the slice.
type EmbeddingRecord struct {
	MBID string
	Vec  []float32
}

// Store encapsulates repository behavior with lifecycle management.
type Store interface {
	ArtistRepository
	AlbumRepository
	EmbeddingRepository
	Close(ctx context.Context) error
}

// MemoryStore is an in-memory persistence layer backing the application during early development.
type MemoryStore struct {
	mu         sync.RWMutex
	artists    map[string]*data.Artist
	albums     map[string]*data.Album
	embeddings map[string]map[string][]float32 // [model][mbid] -> vec
}

// NewMemoryStore constructs an in-memory store instance.
func NewMemoryStore(ctx context.Context) (*MemoryStore, error) {
	_ = ctx
	return &MemoryStore{
		artists:    make(map[string]*data.Artist),
		albums:     make(map[string]*data.Album),
		embeddings: make(map[string]map[string][]float32),
	}, nil
}

// Close releases store resources. Included for future symmetry once a real database is in use.
func (s *MemoryStore) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

// GetArtist retrieves an artist by ID if present.
func (s *MemoryStore) GetArtist(ctx context.Context, id string) (*data.Artist, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	artist, ok := s.artists[id]
	if !ok {
		return nil, nil
	}
	return cloneArtist(artist), nil
}

// SaveArtist persists (or updates) an artist record.
func (s *MemoryStore) SaveArtist(ctx context.Context, artist *data.Artist) error {
	_ = ctx
	if artist == nil {
		return errors.New("db: artist cannot be nil")
	}
	if strings.TrimSpace(artist.ID) == "" {
		return errors.New("db: artist id required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.artists[artist.ID] = cloneArtist(artist)
	return nil
}

// GetAlbum retrieves an album by ID if present.
func (s *MemoryStore) GetAlbum(ctx context.Context, id string) (*data.Album, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	album, ok := s.albums[id]
	if !ok {
		return nil, nil
	}
	return cloneAlbum(album), nil
}

// SaveAlbum persists (or updates) an album record.
func (s *MemoryStore) SaveAlbum(ctx context.Context, album *data.Album) error {
	_ = ctx
	if album == nil {
		return errors.New("db: album cannot be nil")
	}
	if strings.TrimSpace(album.ID) == "" {
		return errors.New("db: album id required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.albums[album.ID] = cloneAlbum(album)
	return nil
}

// ListAlbumsMissingEmbedding returns albums that do not yet have an embedding
// row for the supplied model. A non-positive limit means no limit.
func (s *MemoryStore) ListAlbumsMissingEmbedding(ctx context.Context, model string, limit int) ([]data.Album, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	var albums []data.Album
	for id, album := range s.albums {
		if byModel, ok := s.embeddings[model]; ok {
			if _, exists := byModel[id]; exists {
				continue
			}
		}
		albums = append(albums, *cloneAlbum(album))
		if limit > 0 && len(albums) >= limit {
			break
		}
	}
	return albums, nil
}

func cloneArtist(src *data.Artist) *data.Artist {
	if src == nil {
		return nil
	}
	copyArtist := *src
	copyArtist.Genres = append([]string(nil), src.Genres...)
	copyArtist.Related = append([]data.RelatedArtist(nil), src.Related...)
	copyArtist.Aliases = append([]string(nil), src.Aliases...)
	copyArtist.Albums = cloneAlbums(src.Albums)
	return &copyArtist
}

func cloneAlbums(src []data.Album) []data.Album {
	if len(src) == 0 {
		return nil
	}
	albums := make([]data.Album, len(src))
	for i := range src {
		albums[i] = *cloneAlbum(&src[i])
	}
	return albums
}

func cloneAlbum(src *data.Album) *data.Album {
	if src == nil {
		return nil
	}
	copyAlbum := *src
	copyAlbum.SecondaryTypes = append([]string(nil), src.SecondaryTypes...)
	copyAlbum.Tracks = cloneTracks(src.Tracks)
	copyAlbum.Review = cloneReview(src.Review)
	return &copyAlbum
}

func cloneTracks(src []data.Track) []data.Track {
	if len(src) == 0 {
		return nil
	}
	tracks := make([]data.Track, len(src))
	copy(tracks, src)
	return tracks
}

func cloneReview(src data.Review) data.Review {
	return src
}

// GetEmbedding retrieves an album embedding for the given (mbid, model) pair.
// Returns (nil, nil) if no row exists.
func (s *MemoryStore) GetEmbedding(ctx context.Context, mbid, model string) ([]float32, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	byModel, ok := s.embeddings[model]
	if !ok {
		return nil, nil
	}
	vec, ok := byModel[mbid]
	if !ok {
		return nil, nil
	}
	out := make([]float32, len(vec))
	copy(out, vec)
	return out, nil
}

// SaveEmbedding upserts an embedding vector for (mbid, model).
func (s *MemoryStore) SaveEmbedding(ctx context.Context, mbid, model string, vec []float32) error {
	_ = ctx
	if strings.TrimSpace(mbid) == "" {
		return errors.New("db: embedding mbid required")
	}
	if strings.TrimSpace(model) == "" {
		return errors.New("db: embedding model required")
	}
	if len(vec) == 0 {
		return errors.New("db: embedding vector cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	byModel, ok := s.embeddings[model]
	if !ok {
		byModel = make(map[string][]float32)
		s.embeddings[model] = byModel
	}
	stored := make([]float32, len(vec))
	copy(stored, vec)
	byModel[mbid] = stored
	return nil
}

// LoadAllForModel returns every (mbid, vec) pair currently stored for the
// given model. Caller-owned slices — modifying them does not mutate the store.
func (s *MemoryStore) LoadAllForModel(ctx context.Context, model string) ([]EmbeddingRecord, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	byModel, ok := s.embeddings[model]
	if !ok {
		return nil, nil
	}
	records := make([]EmbeddingRecord, 0, len(byModel))
	for mbid, vec := range byModel {
		out := make([]float32, len(vec))
		copy(out, vec)
		records = append(records, EmbeddingRecord{MBID: mbid, Vec: out})
	}
	return records, nil
}

// DeleteOtherModels removes every embedding whose model != keepModel and
// returns the count of deleted records. Used by `cmd/reindex --prune-old`
// after a rolling model swap.
func (s *MemoryStore) DeleteOtherModels(ctx context.Context, keepModel string) (int, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0
	for model, byMBID := range s.embeddings {
		if model == keepModel {
			continue
		}
		deleted += len(byMBID)
		delete(s.embeddings, model)
	}
	return deleted, nil
}
