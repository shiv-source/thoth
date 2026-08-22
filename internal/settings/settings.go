package settings

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Key constants for the settings table — one place for the wire keys.
// GitHub/sync keys carry the github_ prefix; per-provider keys are built
// from the provider slug via ProviderAPIKeyKey/ProviderBaseURLKey (flat
// keys, not a JSON blob — the point of the key/value shape).
const (
	KeyWikiPath     = "wiki_path"
	KeyWikiFolders  = "wiki_folders"
	KeyModel        = "model"
	KeyRepoURL      = "github_sync_repo"
	KeySyncEnabled  = "github_sync_enabled"
	KeyLastSyncedAt = "github_last_synced_at"
	KeySyncError    = "github_sync_error"
)

// providerSlug reduces a provider name to the slug used in its settings
// keys: lowercased with non-alphanumeric characters removed ("Z.AI" → "zai",
// "xAI" → "xai", "Anthropic" → "anthropic").
func providerSlug(provider string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(provider) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ProviderAPIKeyKey returns the settings key holding a provider's own API
// key, e.g. provider_deepseek_api_key for the "DeepSeek" provider label.
func ProviderAPIKeyKey(provider string) string {
	return "provider_" + providerSlug(provider) + "_api_key"
}

// ProviderBaseURLKey returns the settings key holding a provider's API base
// URL override, e.g. provider_deepseek_base_url for "DeepSeek". Empty means
// the provider's default endpoint.
func ProviderBaseURLKey(provider string) string {
	return "provider_" + providerSlug(provider) + "_base_url"
}

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

// ProviderConfig resolves the credentials for a provider name (the
// llm_models row's provider label): its per-provider API key and base URL
// override. Keys live in thoth.db only — there is no shared fallback and
// nothing is read from the environment. An empty provider name (no registry
// row) resolves to an empty key and the provider's default endpoint.
func (r *Repo) ProviderConfig(provider string) (apiKey, baseURL string, err error) {
	if provider == "" {
		return "", "", nil
	}
	apiKey, _, err = r.Setting(ProviderAPIKeyKey(provider))
	if err != nil {
		return "", "", err
	}
	baseURL, _, err = r.Setting(ProviderBaseURLKey(provider))
	if err != nil {
		return "", "", err
	}
	return apiKey, baseURL, nil
}

// Folders returns the configured scaffold folder names, split on commas and
// trimmed; nil when the key is absent or empty, so callers fall back to the
// default set.
func (r *Repo) Folders() ([]string, error) {
	value, _, err := r.Setting(KeyWikiFolders)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
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
