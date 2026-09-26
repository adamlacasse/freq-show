package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

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

// CollectionRepository defines persistence operations for user collections.
type CollectionRepository interface {
	AddAlbumToCollection(ctx context.Context, userID, albumID, format string) error
	UpdateCollectionItem(ctx context.Context, userID, albumID, format, customArtistName, customTitle string, customYear int) error
	RemoveAlbumFromCollection(ctx context.Context, userID, albumID string) error
	GetUserCollection(ctx context.Context, userID string) ([]data.CollectionItem, error)
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

// AuthRepository defines persistence operations for magic-link auth: user
// accounts, one-time login tokens, and sessions. Callers hash the raw
// bearer token (login token or session cookie value) before it ever reaches
// this interface — implementations only ever see and store the hash, so a
// database dump alone can't be replayed as a working credential.
type AuthRepository interface {
	// GetOrCreateUserByEmail returns the user for a verified email address,
	// creating the account on first login. Email matching is
	// case-insensitive.
	GetOrCreateUserByEmail(ctx context.Context, email string) (*data.User, error)

	// SaveLoginToken stores a freshly issued magic-link token, identified
	// by tokenHash (the caller's hash of the raw emailed token), with its
	// expiry.
	SaveLoginToken(ctx context.Context, tokenHash, email string, expiresAt time.Time) error

	// ConsumeLoginToken atomically marks a login token used and returns the
	// email it was issued for. ok is false if the token is unknown, already
	// consumed, or expired as of now — the caller cannot distinguish these
	// cases, which is intentional: they all mean "this link doesn't work
	// anymore."
	ConsumeLoginToken(ctx context.Context, tokenHash string, now time.Time) (email string, ok bool, err error)

	// CreateSession persists a freshly issued session, identified by
	// tokenHash (the caller's hash of the raw session cookie value).
	CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error

	// GetSession returns the session behind a hashed token if it exists and
	// has not expired as of now. Returns (nil, nil) for a missing or
	// expired session rather than an error — an expired cookie is routine,
	// not exceptional.
	GetSession(ctx context.Context, tokenHash string, now time.Time) (*data.Session, error)

	// DeleteSession removes a session (used for logout). Deleting an
	// already-absent session is not an error.
	DeleteSession(ctx context.Context, tokenHash string) error

	// PruneExpiredAuth deletes login tokens that are expired or already
	// consumed, and sessions that are expired, returning the total rows
	// removed across both. Neither table is otherwise ever cleaned up —
	// every successful login leaves one spent login_tokens row and every
	// session outlives its own usefulness once past its expiry — so
	// without a periodic sweep both grow without bound on a long-running
	// deployment. See cmd/server's background prune loop.
	PruneExpiredAuth(ctx context.Context, now time.Time) (int, error)
}

// Store encapsulates repository behavior with lifecycle management.
type Store interface {
	ArtistRepository
	AlbumRepository
	EmbeddingRepository
	CollectionRepository
	AuthRepository
	Close(ctx context.Context) error
}

// memoryLoginToken is a login token record keyed by its hash.
type memoryLoginToken struct {
	email      string
	expiresAt  time.Time
	consumedAt time.Time // zero value means "not yet consumed"
}

// memorySession is a session record keyed by its hashed token.
type memorySession struct {
	userID    string
	expiresAt time.Time
}

// MemoryStore is an in-memory persistence layer backing the application during early development.
type MemoryStore struct {
	mu           sync.RWMutex
	artists      map[string]*data.Artist
	albums       map[string]*data.Album
	embeddings   map[string]map[string][]float32 // [model][mbid] -> vec
	usersByID    map[string]*data.User
	usersByEmail map[string]*data.User
	loginTokens  map[string]*memoryLoginToken // tokenHash -> record
	sessions     map[string]*memorySession    // tokenHash -> record
}

// NewMemoryStore constructs an in-memory store instance.
func NewMemoryStore(ctx context.Context) (*MemoryStore, error) {
	_ = ctx
	return &MemoryStore{
		artists:      make(map[string]*data.Artist),
		albums:       make(map[string]*data.Album),
		embeddings:   make(map[string]map[string][]float32),
		usersByID:    make(map[string]*data.User),
		usersByEmail: make(map[string]*data.User),
		loginTokens:  make(map[string]*memoryLoginToken),
		sessions:     make(map[string]*memorySession),
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

// AddAlbumToCollection adds an album to a user's collection (MemoryStore dummy implementation).
func (s *MemoryStore) AddAlbumToCollection(ctx context.Context, userID, albumID, format string) error {
	return nil
}

// UpdateCollectionItem updates a collection item (MemoryStore dummy implementation).
func (s *MemoryStore) UpdateCollectionItem(ctx context.Context, userID, albumID, format, customArtistName, customTitle string, customYear int) error {
	return nil
}

// RemoveAlbumFromCollection removes an album from a user's collection (MemoryStore dummy implementation).
func (s *MemoryStore) RemoveAlbumFromCollection(ctx context.Context, userID, albumID string) error {
	return nil
}

// GetUserCollection retrieves the user's collection (MemoryStore dummy implementation).
func (s *MemoryStore) GetUserCollection(ctx context.Context, userID string) ([]data.CollectionItem, error) {
	return nil, nil
}

// GetOrCreateUserByEmail returns the user for a verified email address,
// creating the account on first login. Email matching is case-insensitive.
func (s *MemoryStore) GetOrCreateUserByEmail(ctx context.Context, email string) (*data.User, error) {
	_ = ctx
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("db: email required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.usersByEmail[email]; ok {
		copyUser := *existing
		return &copyUser, nil
	}

	id, err := newRandomID()
	if err != nil {
		return nil, err
	}
	user := &data.User{ID: id, Email: email, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.usersByEmail[email] = user
	s.usersByID[id] = user

	copyUser := *user
	return &copyUser, nil
}

// SaveLoginToken stores a freshly issued magic-link token with its expiry.
func (s *MemoryStore) SaveLoginToken(ctx context.Context, tokenHash, email string, expiresAt time.Time) error {
	_ = ctx
	if strings.TrimSpace(tokenHash) == "" || strings.TrimSpace(email) == "" {
		return errors.New("db: token hash and email required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.loginTokens[tokenHash] = &memoryLoginToken{
		email:     strings.ToLower(strings.TrimSpace(email)),
		expiresAt: expiresAt,
	}
	return nil
}

// ConsumeLoginToken atomically marks a login token used and returns the
// email it was issued for.
func (s *MemoryStore) ConsumeLoginToken(ctx context.Context, tokenHash string, now time.Time) (string, bool, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.loginTokens[tokenHash]
	if !ok {
		return "", false, nil
	}
	if !token.consumedAt.IsZero() || now.After(token.expiresAt) {
		return "", false, nil
	}
	token.consumedAt = now
	return token.email, true, nil
}

// CreateSession persists a freshly issued session.
func (s *MemoryStore) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_ = ctx
	if strings.TrimSpace(tokenHash) == "" || strings.TrimSpace(userID) == "" {
		return errors.New("db: token hash and user id required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[tokenHash] = &memorySession{userID: userID, expiresAt: expiresAt}
	return nil
}

// GetSession returns the session behind a hashed token if it exists and has
// not expired as of now.
func (s *MemoryStore) GetSession(ctx context.Context, tokenHash string, now time.Time) (*data.Session, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[tokenHash]
	if !ok || now.After(session.expiresAt) {
		return nil, nil
	}
	return &data.Session{UserID: session.userID, ExpiresAt: session.expiresAt.UTC().Format(time.RFC3339)}, nil
}

// DeleteSession removes a session.
func (s *MemoryStore) DeleteSession(ctx context.Context, tokenHash string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, tokenHash)
	return nil
}

// PruneExpiredAuth deletes expired/consumed login tokens and expired
// sessions, returning the total rows removed across both.
func (s *MemoryStore) PruneExpiredAuth(ctx context.Context, now time.Time) (int, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0
	for hash, token := range s.loginTokens {
		if now.After(token.expiresAt) || !token.consumedAt.IsZero() {
			delete(s.loginTokens, hash)
			deleted++
		}
	}
	for hash, session := range s.sessions {
		if now.After(session.expiresAt) {
			delete(s.sessions, hash)
			deleted++
		}
	}
	return deleted, nil
}

// newRandomID generates an opaque hex-encoded random identifier for a new
// user record. MemoryStore has no auto-increment/UUID dependency of its
// own, so it rolls its own rather than pulling in google/uuid just for this
// dev-only path.
func newRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
