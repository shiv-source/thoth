package store

import (
	"database/sql"
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

// EnsureMetadata seeds one-time app metadata — installation_id (a v4 UUID
// identifying this installation) and created_at — and is a no-op once they
// exist, so every boot may call it.
func (s *Store) EnsureMetadata() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin metadata: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM app_metadata WHERE key = 'installation_id'`).Scan(&n); err != nil {
		return fmt.Errorf("check installation_id: %w", err)
	}
	if n == 0 {
		id, err := newID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO app_metadata(key, value) VALUES ('installation_id', ?)`, id); err != nil {
			return fmt.Errorf("store installation_id: %w", err)
		}
		if _, err := tx.Exec(`INSERT INTO app_metadata(key, value) VALUES ('created_at', ?)`,
			time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("store created_at: %w", err)
		}
	}
	return tx.Commit()
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
