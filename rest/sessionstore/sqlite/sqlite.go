// Package sqlite implements rest.SessionStore on top of SQLite.
// It shares the *sql.DB connection with the Memory backend.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/yusheng-g/openagent-go/rest"
)

// Store persists session metadata in a SQLite database.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by the given *sql.DB.
func New(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("sqlite sessionstore: migrate: %w", err)
	}
	return s, nil
}

// ── rest.SessionStore ──

func (s *Store) Save(ctx context.Context, info rest.SessionInfo) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO sessions (id, kind, title, model_id, provider, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		info.ID, info.Kind, info.Title, info.ModelID, info.Provider,
		info.CreatedAt.Format(time.RFC3339), info.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) Get(ctx context.Context, id string) (*rest.SessionInfo, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, kind, title, model_id, provider, created_at, updated_at FROM sessions WHERE id = ?`, id)
	return scanInfo(row)
}

func (s *Store) List(ctx context.Context, kind string) ([]rest.SessionInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, title, model_id, provider, created_at, updated_at
		 FROM sessions WHERE kind = ? ORDER BY updated_at DESC`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []rest.SessionInfo
	for rows.Next() {
		info, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *info)
	}
	if list == nil {
		list = []rest.SessionInfo{}
	}
	return list, rows.Err()
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) Close() error { return nil }

// ── migration ──

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT PRIMARY KEY,
			kind       TEXT NOT NULL DEFAULT 'single',
			title      TEXT NOT NULL DEFAULT '',
			model_id   TEXT NOT NULL DEFAULT '',
			provider   TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_kind ON sessions(kind);
	`)
	return err
}

// ── helpers ──

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInfo(row rowScanner) (*rest.SessionInfo, error) {
	var (
		id, kind, title, modelID, provider string
		createdRaw, updatedRaw             string
	)
	if err := row.Scan(&id, &kind, &title, &modelID, &provider, &createdRaw, &updatedRaw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	created, err := time.Parse(time.RFC3339, createdRaw)
	if err != nil {
		log.Printf("sqlite sessionstore: parse created_at %q: %v", createdRaw, err)
	}
	updated, err := time.Parse(time.RFC3339, updatedRaw)
	if err != nil {
		log.Printf("sqlite sessionstore: parse updated_at %q: %v", updatedRaw, err)
	}
	return &rest.SessionInfo{
		ID:        id,
		Kind:      kind,
		Title:     title,
		ModelID:   modelID,
		Provider:  provider,
		CreatedAt: created,
		UpdatedAt: updated,
	}, nil
}

func scanRows(rows *sql.Rows) (*rest.SessionInfo, error) {
	var (
		id, kind, title, modelID, provider string
		createdRaw, updatedRaw             string
	)
	if err := rows.Scan(&id, &kind, &title, &modelID, &provider, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339, createdRaw)
	if err != nil {
		log.Printf("sqlite sessionstore: parse created_at %q: %v", createdRaw, err)
	}
	updated, err := time.Parse(time.RFC3339, updatedRaw)
	if err != nil {
		log.Printf("sqlite sessionstore: parse updated_at %q: %v", updatedRaw, err)
	}
	return &rest.SessionInfo{
		ID:        id,
		Kind:      kind,
		Title:     title,
		ModelID:   modelID,
		Provider:  provider,
		CreatedAt: created,
		UpdatedAt: updated,
	}, nil
}
