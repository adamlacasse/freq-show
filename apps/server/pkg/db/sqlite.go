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
        added_at TIMESTAMP NOT NULL,
        UNIQUE(user_id, album_id)
    )`

	if _, err := s.db.ExecContext(ctx, createCollectionItems); err != nil {
		return fmt.Errorf("db: migrate collection_items: %w", err)
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
	query := `SELECT c.id, c.user_id, c.album_id, c.format, c.added_at, a.payload
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
		var albumPayload sql.NullString
		
		if err := rows.Scan(&item.ID, &item.UserID, &item.AlbumID, &item.Format, &addedAt, &albumPayload); err != nil {
			return nil, fmt.Errorf("db: scan collection item: %w", err)
		}
		item.AddedAt = addedAt.Format(time.RFC3339)

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
