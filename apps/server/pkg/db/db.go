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
}

// Store encapsulates repository behavior with lifecycle management.
type Store interface {
	ArtistRepository
	AlbumRepository
	Close(ctx context.Context) error
}

// MemoryStore is an in-memory persistence layer backing the application during early development.
type MemoryStore struct {
	mu      sync.RWMutex
	artists map[string]*data.Artist
	albums  map[string]*data.Album
}

// NewMemoryStore constructs an in-memory store instance.
func NewMemoryStore(ctx context.Context) (*MemoryStore, error) {
	_ = ctx
	return &MemoryStore{
		artists: make(map[string]*data.Artist),
		albums:  make(map[string]*data.Album),
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

func cloneArtist(src *data.Artist) *data.Artist {
	if src == nil {
		return nil
	}
	copyArtist := *src
	copyArtist.Genres = append([]string(nil), src.Genres...)
	copyArtist.Related = append([]string(nil), src.Related...)
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
