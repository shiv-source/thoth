package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	// The schema lives entirely in migrations/*.sql (see migrations.go) —
	// Open never issues DDL of its own.
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate store: %w", err)
	}
	return &Store{db: db}, nil
}

func newID() (string, error) {
	// Valid RFC 4122 v4: the claude CLI rejects --session-id values that are
	// not valid UUIDs.
	u, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return u.String(), nil
}

// EnsureMetadata seeds the single app_metadata row on first boot — a v4
// installation_id and the UTC created_at — and is a no-op afterwards, so
// every boot may call it. The id defaults to 1 and the CHECK (id = 1)
// constraint keeps the table to one row, so INSERT OR IGNORE is the atomic
// "create if absent".
func (s *Store) EnsureMetadata() error {
	id, err := newID()
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO app_metadata(installation_id, created_at) VALUES (?, ?)`,
		id, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("seed metadata: %w", err)
	}
	return nil
}

// SyncState returns the recorded git sync outcome: the last successful sync
// (empty when never) and the last error (empty when none).
func (s *Store) SyncState() (lastSyncedAt, syncError string, err error) {
	var l, e sql.NullString
	err = s.db.QueryRow(`SELECT last_synced_at, sync_error FROM app_metadata`).Scan(&l, &e)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("read sync state: %w", err)
	}
	return l.String, e.String, nil
}

// SetSyncResult records the outcome of a git sync: success stamps
// last_synced_at and clears sync_error; failure records the error and keeps
// last_synced_at at the last successful sync.
func (s *Store) SetSyncResult(ok bool, detail string) error {
	var err error
	if ok {
		_, err = s.db.Exec(`UPDATE app_metadata SET last_synced_at = ?, sync_error = NULL WHERE id = 1`,
			time.Now().UTC().Format(time.RFC3339))
	} else {
		_, err = s.db.Exec(`UPDATE app_metadata SET sync_error = ? WHERE id = 1`, detail)
	}
	if err != nil {
		return fmt.Errorf("set sync result: %w", err)
	}
	return nil
}

func (s *Store) CreateConversation(title string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	// created_at is stored as RFC3339 text and compared lexically by
	// ORDER BY, so it must be UTC: local offsets would misorder rows
	// written under different offsets.
	_, err = s.db.Exec(
		`INSERT INTO conversations(id, title, created_at) VALUES (?, ?, ?)`,
		id, title, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("create conversation: %w", err)
	}
	return id, nil
}

func (s *Store) AddMessage(convID, role, content string) error {
	_, err := s.db.Exec(
		`INSERT INTO messages(conversation_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		convID, role, content, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

func (s *Store) ListConversations() ([]Conversation, error) {
	rows, err := s.db.Query(`SELECT id, title, created_at FROM conversations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Conversation
	for rows.Next() {
		var c Conversation
		var created string
		if err := rows.Scan(&c.ID, &c.Title, &created); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			c.CreatedAt = t
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Messages(convID string) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, created_at FROM messages WHERE conversation_id = ? ORDER BY id ASC`,
		convID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Message
	for rows.Next() {
		var m Message
		var created string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &created); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			m.CreatedAt = t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) Close() error { return s.db.Close() }
