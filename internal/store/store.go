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

func (s *Store) CreateConversation(title string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	// created_at is stored as RFC3339 text and compared lexically by
	// ORDER BY, so it must be UTC: local offsets would misorder rows
	// written under different offsets.
	_, err = s.db.Exec(
		`INSERT INTO conversations(id, title, created_at, claude_session_id) VALUES (?, ?, ?, ?)`,
		id, title, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return "", fmt.Errorf("create conversation: %w", err)
	}
	return id, nil
}

// ConversationSessionID returns the Claude CLI session id stored for the
// conversation ("" when the conversation does not exist).
func (s *Store) ConversationSessionID(convID string) (string, error) {
	var sid string
	err := s.db.QueryRow(`SELECT claude_session_id FROM conversations WHERE id = ?`, convID).Scan(&sid)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read session id: %w", err)
	}
	return sid, nil
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

// SetClaudeSessionID stores the Claude CLI session id for the conversation
// (rotation writes a fresh id after forking away from a stale-locked one).
func (s *Store) SetClaudeSessionID(convID, sessionID string) error {
	if _, err := s.db.Exec(`UPDATE conversations SET claude_session_id = ? WHERE id = ?`, sessionID, convID); err != nil {
		return fmt.Errorf("set session id: %w", err)
	}
	return nil
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
