package db

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

// SQLiteStore persists artists in a SQLite database using JSON payloads for flexibility.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at the provided DSN and applies lightweight migrations.
func NewSQLiteStore(ctx context.Context, dsn string) (*SQLiteStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("db: database url required")
	}

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open sqlite: %w", err)
	}

	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("db: ping sqlite: %w", err)
	}

	store := &SQLiteStore{db: database}
	if err := store.migrate(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}

	return store, nil
}

// Close releases database resources.
func (s *SQLiteStore) Close(ctx context.Context) error {
	_ = ctx
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// GetArtist retrieves an artist by ID if present.
func (s *SQLiteStore) GetArtist(ctx context.Context, id string) (*data.Artist, error) {
	row := s.db.QueryRowContext(ctx, `SELECT payload FROM artists WHERE id = ?`, id)

	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: query artist: %w", err)
	}

	var artist data.Artist
	if err := json.Unmarshal([]byte(payload), &artist); err != nil {
		return nil, fmt.Errorf("db: decode artist: %w", err)
	}

	return &artist, nil
}

// SaveArtist upserts an artist record in the database.
func (s *SQLiteStore) SaveArtist(ctx context.Context, artist *data.Artist) error {
	if artist == nil {
		return errors.New("db: artist cannot be nil")
	}
	if strings.TrimSpace(artist.ID) == "" {
		return errors.New("db: artist id required")
	}

	payload, err := json.Marshal(artist)
	if err != nil {
		return fmt.Errorf("db: encode artist: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO artists (id, payload, updated_at)
         VALUES (?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET payload = excluded.payload, updated_at = excluded.updated_at`,
		artist.ID,
		string(payload),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("db: upsert artist: %w", err)
	}
	return nil
}

// GetAlbum retrieves an album by ID if present.
func (s *SQLiteStore) GetAlbum(ctx context.Context, id string) (*data.Album, error) {
	row := s.db.QueryRowContext(ctx, `SELECT payload FROM albums WHERE id = ?`, id)

	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: query album: %w", err)
	}

	var album data.Album
	if err := json.Unmarshal([]byte(payload), &album); err != nil {
		return nil, fmt.Errorf("db: decode album: %w", err)
	}

	return &album, nil
}

// SaveAlbum upserts an album record in the database.
func (s *SQLiteStore) SaveAlbum(ctx context.Context, album *data.Album) error {
	if album == nil {
		return errors.New("db: album cannot be nil")
	}
	if strings.TrimSpace(album.ID) == "" {
		return errors.New("db: album id required")
	}

	payload, err := json.Marshal(album)
	if err != nil {
		return fmt.Errorf("db: encode album: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO albums (id, payload, updated_at)
         VALUES (?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET payload = excluded.payload, updated_at = excluded.updated_at`,
		album.ID,
		string(payload),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("db: upsert album: %w", err)
	}
	return nil
}

// ListAlbumsMissingEmbedding returns cached albums without an embedding row
// for the supplied model. A non-positive limit means no limit.
func (s *SQLiteStore) ListAlbumsMissingEmbedding(ctx context.Context, model string, limit int) ([]data.Album, error) {
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("db: embedding model required")
	}

	query := `SELECT albums.payload
        FROM albums
        LEFT JOIN album_embeddings
          ON album_embeddings.mbid = albums.id
         AND album_embeddings.model = ?
        WHERE album_embeddings.mbid IS NULL
        ORDER BY albums.updated_at ASC`
	args := []any{model}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: query albums missing embeddings: %w", err)
	}
	defer rows.Close()

	var albums []data.Album
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("db: scan album payload: %w", err)
		}
		var album data.Album
		if err := json.Unmarshal([]byte(payload), &album); err != nil {
			return nil, fmt.Errorf("db: decode album payload: %w", err)
		}
		albums = append(albums, album)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate albums missing embeddings: %w", err)
	}
	return albums, nil
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	const createArtists = `CREATE TABLE IF NOT EXISTS artists (
        id TEXT PRIMARY KEY,
        payload TEXT NOT NULL,
        updated_at TIMESTAMP NOT NULL
    )`

	if _, err := s.db.ExecContext(ctx, createArtists); err != nil {
		return fmt.Errorf("db: migrate artists: %w", err)
	}

	const createAlbums = `CREATE TABLE IF NOT EXISTS albums (
        id TEXT PRIMARY KEY,
        payload TEXT NOT NULL,
        updated_at TIMESTAMP NOT NULL
    )`

	if _, err := s.db.ExecContext(ctx, createAlbums); err != nil {
		return fmt.Errorf("db: migrate albums: %w", err)
	}

	const createEmbeddings = `CREATE TABLE IF NOT EXISTS album_embeddings (
        mbid       TEXT NOT NULL,
        model      TEXT NOT NULL,
        dim        INTEGER NOT NULL,
        vec        BLOB NOT NULL,
        updated_at TIMESTAMP NOT NULL,
        PRIMARY KEY (mbid, model)
    )`

	if _, err := s.db.ExecContext(ctx, createEmbeddings); err != nil {
		return fmt.Errorf("db: migrate album_embeddings: %w", err)
	}

	const createEmbeddingsModelIdx = `CREATE INDEX IF NOT EXISTS album_embeddings_model_idx
        ON album_embeddings (model)`

	if _, err := s.db.ExecContext(ctx, createEmbeddingsModelIdx); err != nil {
		return fmt.Errorf("db: migrate album_embeddings index: %w", err)
	}

	const createCollectionItems = `CREATE TABLE IF NOT EXISTS collection_items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id TEXT NOT NULL,
        album_id TEXT NOT NULL,
        format TEXT,
        custom_artist_name TEXT,
        custom_title TEXT,
        custom_year INTEGER,
        added_at TIMESTAMP NOT NULL,
        UNIQUE(user_id, album_id)
    )`

	if _, err := s.db.ExecContext(ctx, createCollectionItems); err != nil {
		return fmt.Errorf("db: migrate collection_items: %w", err)
	}

	// Gracefully ensure columns exist for pre-existing tables
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE collection_items ADD COLUMN custom_artist_name TEXT`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE collection_items ADD COLUMN custom_title TEXT`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE collection_items ADD COLUMN custom_year INTEGER`)

	// Magic-link auth (see BACKLOG.md "Magic link authentication").
	const createUsers = `CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        created_at TIMESTAMP NOT NULL,
        last_login_at TIMESTAMP
    )`

	if _, err := s.db.ExecContext(ctx, createUsers); err != nil {
		return fmt.Errorf("db: migrate users: %w", err)
	}

	// login_tokens holds hashed, one-time magic-link tokens. The raw token
	// is only ever seen by the user's browser (as the ?token= query
	// parameter) and is never persisted — only its SHA-256 hash is, so a
	// database dump can't be replayed as a working login link.
	const createLoginTokens = `CREATE TABLE IF NOT EXISTS login_tokens (
        token_hash TEXT PRIMARY KEY,
        email TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        expires_at TIMESTAMP NOT NULL,
        consumed_at TIMESTAMP
    )`

	if _, err := s.db.ExecContext(ctx, createLoginTokens); err != nil {
		return fmt.Errorf("db: migrate login_tokens: %w", err)
	}

	// No index on login_tokens.email: every query against this table
	// (ConsumeLoginToken) looks up by token_hash, the primary key. email is
	// stored only to hand back to GetOrCreateUserByEmail after a token is
	// consumed, never filtered on.

	// sessions holds hashed session-cookie values, mirroring login_tokens:
	// only the hash of the cookie value is stored.
	const createSessions = `CREATE TABLE IF NOT EXISTS sessions (
        token_hash TEXT PRIMARY KEY,
        user_id TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        expires_at TIMESTAMP NOT NULL
    )`

	if _, err := s.db.ExecContext(ctx, createSessions); err != nil {
		return fmt.Errorf("db: migrate sessions: %w", err)
	}

	const createSessionsUserIdx = `CREATE INDEX IF NOT EXISTS sessions_user_id_idx
        ON sessions (user_id)`

	if _, err := s.db.ExecContext(ctx, createSessionsUserIdx); err != nil {
		return fmt.Errorf("db: migrate sessions index: %w", err)
	}

	return nil
}

// GetEmbedding retrieves an embedding for (mbid, model). Returns (nil, nil)
// if no row exists.
func (s *SQLiteStore) GetEmbedding(ctx context.Context, mbid, model string) ([]float32, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT vec FROM album_embeddings WHERE mbid = ? AND model = ?`,
		mbid, model,
	)

	var blob []byte
	if err := row.Scan(&blob); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: query embedding: %w", err)
	}
	vec, err := decodeVector(blob)
	if err != nil {
		return nil, fmt.Errorf("db: decode embedding: %w", err)
	}
	return vec, nil
}

// SaveEmbedding upserts an embedding for (mbid, model). The vector's length
// is stored as the `dim` column for inspection and sanity checks.
func (s *SQLiteStore) SaveEmbedding(ctx context.Context, mbid, model string, vec []float32) error {
	if strings.TrimSpace(mbid) == "" {
		return errors.New("db: embedding mbid required")
	}
	if strings.TrimSpace(model) == "" {
		return errors.New("db: embedding model required")
	}
	if len(vec) == 0 {
		return errors.New("db: embedding vector cannot be empty")
	}

	blob := encodeVector(vec)
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO album_embeddings (mbid, model, dim, vec, updated_at)
         VALUES (?, ?, ?, ?, ?)
         ON CONFLICT(mbid, model) DO UPDATE SET dim = excluded.dim, vec = excluded.vec, updated_at = excluded.updated_at`,
		mbid, model, len(vec), blob, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("db: upsert embedding: %w", err)
	}
	return nil
}

// LoadAllForModel returns every (mbid, vec) pair currently stored for the
// given model. The caller may keep these in memory across requests; the
// discovery service does this and reloads on a TTL or after a reindex.
func (s *SQLiteStore) LoadAllForModel(ctx context.Context, model string) ([]EmbeddingRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT mbid, vec FROM album_embeddings WHERE model = ?`,
		model,
	)
	if err != nil {
		return nil, fmt.Errorf("db: query embeddings for model: %w", err)
	}
	defer rows.Close()

	var records []EmbeddingRecord
	for rows.Next() {
		var mbid string
		var blob []byte
		if err := rows.Scan(&mbid, &blob); err != nil {
			return nil, fmt.Errorf("db: scan embedding row: %w", err)
		}
		vec, err := decodeVector(blob)
		if err != nil {
			return nil, fmt.Errorf("db: decode embedding for %s: %w", mbid, err)
		}
		records = append(records, EmbeddingRecord{MBID: mbid, Vec: vec})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate embedding rows: %w", err)
	}
	return records, nil
}

// DeleteOtherModels removes every embedding whose model != keepModel.
// Returns the count of deleted records.
func (s *SQLiteStore) DeleteOtherModels(ctx context.Context, keepModel string) (int, error) {
	res, err := s.db.ExecContext(
		ctx,
		`DELETE FROM album_embeddings WHERE model != ?`,
		keepModel,
	)
	if err != nil {
		return 0, fmt.Errorf("db: delete embeddings: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("db: rows affected: %w", err)
	}
	return int(n), nil
}

// encodeVector packs a float32 slice as raw little-endian bytes (4 bytes
// per element). 4× smaller than JSON and avoids parse overhead at corpus scale.
func encodeVector(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeVector is the inverse of encodeVector.
func decodeVector(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("vector blob has non-multiple-of-4 length: %d", len(b))
	}
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v, nil
}

// AddAlbumToCollection adds an album to a user's collection.
func (s *SQLiteStore) AddAlbumToCollection(ctx context.Context, userID, albumID, format string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(albumID) == "" {
		return errors.New("db: user id and album id required")
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO collection_items (user_id, album_id, format, added_at)
         VALUES (?, ?, ?, ?)
         ON CONFLICT(user_id, album_id) DO UPDATE SET format = excluded.format, added_at = excluded.added_at`,
		userID, albumID, format, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("db: add to collection: %w", err)
	}
	return nil
}

// UpdateCollectionItem updates format, custom_artist_name, custom_title, and custom_year for a collection item.
func (s *SQLiteStore) UpdateCollectionItem(ctx context.Context, userID, albumID, format, customArtistName, customTitle string, customYear int) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(albumID) == "" {
		return errors.New("db: user id and album id required")
	}

	var titleVal sql.NullString
	if strings.TrimSpace(customTitle) != "" {
		titleVal = sql.NullString{String: strings.TrimSpace(customTitle), Valid: true}
	}

	var yearVal sql.NullInt64
	if customYear > 0 {
		yearVal = sql.NullInt64{Int64: int64(customYear), Valid: true}
	}

	var artistVal sql.NullString
	if strings.TrimSpace(customArtistName) != "" {
		artistVal = sql.NullString{String: strings.TrimSpace(customArtistName), Valid: true}
	}

	_, err := s.db.ExecContext(
		ctx,
		`UPDATE collection_items SET format = ?, custom_artist_name = ?, custom_title = ?, custom_year = ? WHERE user_id = ? AND album_id = ?`,
		format, artistVal, titleVal, yearVal, userID, albumID,
	)
	if err != nil {
		return fmt.Errorf("db: update collection item: %w", err)
	}
	return nil
}

// RemoveAlbumFromCollection removes an album from a user's collection.
func (s *SQLiteStore) RemoveAlbumFromCollection(ctx context.Context, userID, albumID string) error {
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM collection_items WHERE user_id = ? AND album_id = ?`,
		userID, albumID,
	)
	if err != nil {
		return fmt.Errorf("db: remove from collection: %w", err)
	}
	return nil
}

// GetUserCollection retrieves the user's collection.
func (s *SQLiteStore) GetUserCollection(ctx context.Context, userID string) ([]data.CollectionItem, error) {
	query := `SELECT c.id, c.user_id, c.album_id, c.format, c.custom_artist_name, c.custom_title, c.custom_year, c.added_at, a.payload
	          FROM collection_items c
			  LEFT JOIN albums a ON c.album_id = a.id
			  WHERE c.user_id = ?
			  ORDER BY c.added_at DESC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("db: query collection: %w", err)
	}
	defer rows.Close()

	var items []data.CollectionItem
	for rows.Next() {
		var item data.CollectionItem
		var addedAt time.Time
		var customArtistName sql.NullString
		var customTitle sql.NullString
		var customYear sql.NullInt64
		var albumPayload sql.NullString

		if err := rows.Scan(&item.ID, &item.UserID, &item.AlbumID, &item.Format, &customArtistName, &customTitle, &customYear, &addedAt, &albumPayload); err != nil {
			return nil, fmt.Errorf("db: scan collection item: %w", err)
		}
		item.AddedAt = addedAt.Format(time.RFC3339)
		if customArtistName.Valid {
			item.CustomArtistName = customArtistName.String
		}
		if customTitle.Valid {
			item.CustomTitle = customTitle.String
		}
		if customYear.Valid {
			item.CustomYear = int(customYear.Int64)
		}

		if albumPayload.Valid && albumPayload.String != "" {
			var album data.Album
			if err := json.Unmarshal([]byte(albumPayload.String), &album); err == nil {
				item.Album = &album
			}
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate collection: %w", err)
	}

	return items, nil
}

// GetOrCreateUserByEmail returns the user for a verified email address,
// creating the account on first login. Email matching is case-insensitive.
func (s *SQLiteStore) GetOrCreateUserByEmail(ctx context.Context, email string) (*data.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("db: email required")
	}

	if user, err := s.getUserByEmail(ctx, email); err != nil {
		return nil, err
	} else if user != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now().UTC(), user.ID)
		return user, nil
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, email, created_at, last_login_at) VALUES (?, ?, ?, ?)`,
		id, email, now, now,
	)
	if err != nil {
		if isUniqueConstraintErr(err) {
			// Another request created this same user concurrently between
			// our lookup and insert above — re-read rather than failing.
			user, readErr := s.getUserByEmail(ctx, email)
			if readErr != nil {
				return nil, readErr
			}
			if user != nil {
				return user, nil
			}
		}
		return nil, fmt.Errorf("db: insert user: %w", err)
	}

	return &data.User{ID: id, Email: email, CreatedAt: now.Format(time.RFC3339)}, nil
}

// GetUser returns the user with the given ID, or (nil, nil) if not found.
func (s *SQLiteStore) GetUser(ctx context.Context, id string) (*data.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT id, email, created_at FROM users WHERE id = ?`, id)

	var user data.User
	var createdAt time.Time
	if err := row.Scan(&user.ID, &user.Email, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: query user by id: %w", err)
	}
	user.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return &user, nil
}

func (s *SQLiteStore) getUserByEmail(ctx context.Context, email string) (*data.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, email, created_at FROM users WHERE email = ?`, email)

	var user data.User
	var createdAt time.Time
	if err := row.Scan(&user.ID, &user.Email, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: query user: %w", err)
	}
	user.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return &user, nil
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint")
}

// SaveLoginToken stores a freshly issued magic-link token with its expiry.
func (s *SQLiteStore) SaveLoginToken(ctx context.Context, tokenHash, email string, expiresAt time.Time) error {
	if strings.TrimSpace(tokenHash) == "" || strings.TrimSpace(email) == "" {
		return errors.New("db: token hash and email required")
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO login_tokens (token_hash, email, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash, strings.ToLower(strings.TrimSpace(email)), time.Now().UTC(), expiresAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("db: insert login token: %w", err)
	}
	return nil
}

// ConsumeLoginToken atomically marks a login token used and returns the
// email it was issued for. The UPDATE's own WHERE clause (consumed_at IS
// NULL AND expires_at > ?) is the atomicity: a SELECT-then-UPDATE would
// leave a window between the two statements where two near-simultaneous
// callers (e.g. an email scanner prefetching the link and the real user
// clicking it moments later) could both read "not yet consumed" and both
// proceed to update, relying on SQLite's BUSY retry behavior rather than
// the query itself to prevent a double-consume. A single conditional
// UPDATE has no such window: only one caller's statement can be the one
// that actually flips consumed_at, and RowsAffected tells us which.
func (s *SQLiteStore) ConsumeLoginToken(ctx context.Context, tokenHash string, now time.Time) (string, bool, error) {
	if strings.TrimSpace(tokenHash) == "" {
		return "", false, nil
	}

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE login_tokens SET consumed_at = ? WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?`,
		now.UTC(), tokenHash, now.UTC(),
	)
	if err != nil {
		return "", false, fmt.Errorf("db: consume login token: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("db: consume login token rows affected: %w", err)
	}
	if affected != 1 {
		// Either the token doesn't exist, or it was already consumed/expired
		// (including by a concurrent caller that won the race above).
		return "", false, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT email FROM login_tokens WHERE token_hash = ?`, tokenHash)
	var email string
	if err := row.Scan(&email); err != nil {
		return "", false, fmt.Errorf("db: query consumed login token email: %w", err)
	}
	return email, true, nil
}

// CreateSession persists a freshly issued session.
func (s *SQLiteStore) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	if strings.TrimSpace(tokenHash) == "" || strings.TrimSpace(userID) == "" {
		return errors.New("db: token hash and user id required")
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash, userID, time.Now().UTC(), expiresAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("db: insert session: %w", err)
	}
	return nil
}

// GetSession returns the session behind a hashed token if it exists and has
// not expired as of now.
func (s *SQLiteStore) GetSession(ctx context.Context, tokenHash string, now time.Time) (*data.Session, error) {
	row := s.db.QueryRowContext(ctx, `SELECT user_id, expires_at FROM sessions WHERE token_hash = ?`, tokenHash)

	var userID string
	var expiresAt time.Time
	if err := row.Scan(&userID, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: query session: %w", err)
	}
	if now.After(expiresAt) {
		return nil, nil
	}
	return &data.Session{UserID: userID, ExpiresAt: expiresAt.UTC().Format(time.RFC3339)}, nil
}

// DeleteSession removes a session.
func (s *SQLiteStore) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return fmt.Errorf("db: delete session: %w", err)
	}
	return nil
}

// PruneExpiredAuth deletes expired/consumed login tokens and expired
// sessions, returning the total rows removed across both.
func (s *SQLiteStore) PruneExpiredAuth(ctx context.Context, now time.Time) (int, error) {
	now = now.UTC()

	tokensRes, err := s.db.ExecContext(ctx, `DELETE FROM login_tokens WHERE expires_at < ? OR consumed_at IS NOT NULL`, now)
	if err != nil {
		return 0, fmt.Errorf("db: prune login_tokens: %w", err)
	}
	tokensDeleted, err := tokensRes.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("db: prune login_tokens rows affected: %w", err)
	}

	sessionsRes, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now)
	if err != nil {
		return int(tokensDeleted), fmt.Errorf("db: prune sessions: %w", err)
	}
	sessionsDeleted, err := sessionsRes.RowsAffected()
	if err != nil {
		return int(tokensDeleted), fmt.Errorf("db: prune sessions rows affected: %w", err)
	}

	return int(tokensDeleted + sessionsDeleted), nil
}
