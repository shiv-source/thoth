package store

import (
	"database/sql"
	"encoding/json"
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
	// Usage is the assistant turn's token breakdown as raw JSON
	// ({"input_tokens":N,"output_tokens":N,"cache_read_tokens":N,
	// "cache_write_tokens":N}); nil on user messages and rows written before
	// usage was tracked.
	Usage json.RawMessage `json:"usage,omitempty"`
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := OpenDB(path)
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
	// 0011 moved the per-provider credentials from the legacy settings keys
	// into the providers table; the copy runs after the migrations so the
	// table exists, and is a no-op once the keys are gone.
	if err := backfillProviderCredentials(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("backfill provider credentials: %w", err)
	}
	return &Store{db: db}, nil
}

func newID() (string, error) {
	// Valid RFC 4122 v4, so ids never collide and parse everywhere.
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

func (s *Store) CreateConversation(title string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	// created_at is stored as RFC3339 text and compared lexically by
	// ORDER BY, so it must be UTC: local offsets would misorder rows
	// written under different offsets. The legacy claude_session_id column
	// is retained but unused — the T12 decision kept it (no migration) to
	// avoid rewriting the conversations table for a column nothing reads.
	_, err = s.db.Exec(
		`INSERT INTO conversations(id, title, created_at) VALUES (?, ?, ?)`,
		id, title, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("create conversation: %w", err)
	}
	return id, nil
}

// DeleteConversation removes the conversation and its messages; deleting a
// missing conversation is not an error. Messages go first so a future
// PRAGMA foreign_keys=ON stays satisfied.
func (s *Store) DeleteConversation(convID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete conversation: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit
	if _, err := tx.Exec(`DELETE FROM messages WHERE conversation_id = ?`, convID); err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM conversations WHERE id = ?`, convID); err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete conversation: %w", err)
	}
	return nil
}

// AddMessage inserts a message. The optional usage is a raw JSON token
// breakdown carried by the turn's assistant message (see Message.Usage); it is
// stored as NULL when absent.
func (s *Store) AddMessage(convID, role, content string, usage ...string) error {
	var usageVal any
	if len(usage) > 0 && usage[0] != "" {
		usageVal = usage[0]
	}
	_, err := s.db.Exec(
		`INSERT INTO messages(conversation_id, role, content, usage, created_at) VALUES (?, ?, ?, ?, ?)`,
		convID, role, content, usageVal, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

func (s *Store) ListConversations() ([]Conversation, error) {
	// rowid breaks the tie between conversations created in the same second
	// (created_at is second-precision RFC3339), so "most recent" is fully
	// deterministic — the prewarm and the UI list agree.
	rows, err := s.db.Query(`SELECT id, title, created_at FROM conversations ORDER BY created_at DESC, rowid DESC`)
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

// Messages returns the messages of a conversation in insertion order.
func (s *Store) Messages(convID string) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, created_at, usage FROM messages WHERE conversation_id = ? ORDER BY id ASC`,
		convID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Message
	for rows.Next() {
		var m Message
		var created string
		var usage sql.NullString
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &created, &usage); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			m.CreatedAt = t
		}
		if usage.Valid && usage.String != "" {
			m.Usage = json.RawMessage(usage.String)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) Close() error { return s.db.Close() }
