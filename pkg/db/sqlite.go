package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adamlacasse/freq-show/pkg/data"

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
	return nil
}
