package store

import (
	"database/sql"
	"fmt"
	"time"
)

// PushEntry is one completed sync attempt, newest first. The Settings page
// renders the recent run history from these rows — beyond the single
// last_synced_at / last_error columns on the connection.
type PushEntry struct {
	At    string `json:"at"` // UTC RFC3339 of the attempt
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// maxPushHistory caps the retained history per connection: the table is
// pruned to this many rows after each append, so it cannot grow unbounded.
const maxPushHistory = 20

// AppendPushHistory records one sync attempt and prunes the connection's
// history to maxPushHistory rows (oldest dropped first). It is called by
// SetConnectionSyncResult, so every sync outcome lands here without callers
// duplicating the write.
func (s *Store) AppendPushHistory(connectionID int64, ok bool, detail string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(
		`INSERT INTO sync_push_history(connection_id, at, ok, error, created_at) VALUES (?, ?, ?, ?, ?)`,
		connectionID, now, ok, nullString(detail), now); err != nil {
		return fmt.Errorf("append push history: %w", err)
	}
	if _, err := s.db.Exec(
		`DELETE FROM sync_push_history WHERE id IN (
			SELECT id FROM sync_push_history WHERE connection_id = ?
			ORDER BY id DESC LIMIT -1 OFFSET ?)`,
		connectionID, maxPushHistory); err != nil {
		return fmt.Errorf("prune push history: %w", err)
	}
	return nil
}

// ListPushHistory returns the most recent push attempts for a connection,
// newest first, capped at maxPushHistory.
func (s *Store) ListPushHistory(connectionID int64) ([]PushEntry, error) {
	rows, err := s.db.Query(
		`SELECT at, ok, error FROM sync_push_history WHERE connection_id = ? ORDER BY id DESC LIMIT ?`,
		connectionID, maxPushHistory)
	if err != nil {
		return nil, fmt.Errorf("list push history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	// Non-nil so an empty history serializes as [] on the connection DTO,
	// never null — the client types it as an array.
	out := make([]PushEntry, 0)
	for rows.Next() {
		var at string
		var ok int
		var errText sql.NullString
		if err := rows.Scan(&at, &ok, &errText); err != nil {
			return nil, fmt.Errorf("scan push history: %w", err)
		}
		out = append(out, PushEntry{At: at, OK: ok != 0, Error: errText.String})
	}
	return out, rows.Err()
}
