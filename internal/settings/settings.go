package settings

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Key constants for the settings table — one place for the wire keys.
// GitHub/sync keys carry the github_ prefix.
const (
	KeyWikiPath     = "wiki_path"
	KeyModel        = "model"
	KeyAPIKey       = "api_key"
	KeyRepoURL      = "github_sync_repo"
	KeySyncEnabled  = "github_sync_enabled"
	KeyLastSyncedAt = "github_last_synced_at"
	KeySyncError    = "github_sync_error"
)

// DefaultWikiPath is the fallback wiki location; the value is mirrored by
// the 0007_settings.sql seed so the row always exists in practice.
const DefaultWikiPath = "~/.thoth/wiki"

// Repo owns the key/value settings table on its own connection to thoth.db.
type Repo struct {
	db *sql.DB
}

// OpenRepo opens a read/write handle. Deliberately no migrations, no WAL
// pragma, and no store.OpenDB: the schema comes from the store's migrations,
// and pragmas (including OpenDB's busy_timeout) force an early connection,
// which would create the file on a missing path during an otherwise
// read-only doctor check. The file is WAL by construction (store.Open always
// runs first in every real flow).
func OpenRepo(path string) (*Repo, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open settings: %w", err)
	}
	return &Repo{db: db}, nil
}

func (r *Repo) Close() error { return r.db.Close() }

// Setting returns the value for key; found is false when the key is absent.
func (r *Repo) Setting(key string) (string, bool, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read setting %s: %w", key, err)
	}
	return value, true, nil
}

// SetSetting upserts the value for key.
func (r *Repo) SetSetting(key, value string) error {
	if _, err := r.db.Exec(
		`INSERT INTO settings(key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value); err != nil {
		return fmt.Errorf("set setting %s: %w", key, err)
	}
	return nil
}

// SyncEnabled reports the auto-sync switch; anything other than the literal
// "true" (including an absent key) is false.
func (r *Repo) SyncEnabled() (bool, error) {
	value, _, err := r.Setting(KeySyncEnabled)
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

// SyncState returns the recorded git sync outcome: the last successful sync
// (empty when never) and the last error (empty when none).
func (r *Repo) SyncState() (lastSyncedAt, syncError string, err error) {
	lastSyncedAt, _, err = r.Setting(KeyLastSyncedAt)
	if err != nil {
		return "", "", err
	}
	syncError, _, err = r.Setting(KeySyncError)
	if err != nil {
		return "", "", err
	}
	return lastSyncedAt, syncError, nil
}

// SetSyncResult records the outcome of a git sync: success stamps
// last_synced_at and clears sync_error; failure records the error and keeps
// last_synced_at at the last successful sync.
func (r *Repo) SetSyncResult(ok bool, detail string) error {
	if ok {
		if err := r.SetSetting(KeyLastSyncedAt, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		return r.SetSetting(KeySyncError, "")
	}
	return r.SetSetting(KeySyncError, detail)
}
