package settings

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// Key constants for the settings table — one place for the wire keys.
// The per-provider keys below are retired: migration 0011 moved credentials
// into the providers table, so ProviderAPIKeyKey/ProviderBaseURLKey survive
// only as the read keys of the one-time credential backfill in store.Open.
// Sync state (target, enabled, last sync, errors) lives on sync_connections
// rows now (migration 0012), not here.
const (
	KeyWikiPath         = "wiki_path"
	KeyWikiFolders      = "wiki_folders"
	KeyModel            = "model"
	KeyActiveConnection = "sync_active_connection"
	// KeyContextInjection gates pre-searching the wiki into each chat turn's
	// first prompt (off by default — it changes answer semantics, so users
	// opt in).
	KeyContextInjection = "context_injection"
	// KeyConversationRetentionDays is the chat-history auto-delete window in
	// days; absent/unparseable falls back to DefaultRetentionDays and a stored
	// zero disables auto-delete.
	KeyConversationRetentionDays = "conversation_retention_days"
)

// DefaultRetentionDays is the conversation auto-delete window a fresh install
// runs with when the settings key is absent.
const DefaultRetentionDays = 7

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

// ProviderAPIKeyKey returns the legacy settings key that held a provider's
// own API key, e.g. provider_deepseek_api_key for the "DeepSeek" provider
// label. Retained for the 0011 credential backfill — no new code writes it.
func ProviderAPIKeyKey(provider string) string {
	return "provider_" + providerSlug(provider) + "_api_key"
}

// ProviderBaseURLKey returns the legacy settings key that held a provider's
// API base URL override, e.g. provider_deepseek_base_url for "DeepSeek".
// Retained for the 0011 credential backfill — no new code writes it.
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

// ProviderConfig resolves the credentials for a provider name from the
// providers table (migration 0011): its own API key and base URL override.
// Keys live in thoth.db only — there is no shared fallback and nothing is
// read from the environment. An empty provider name (no registry row)
// resolves to an empty key and the provider's default endpoint.
func (r *Repo) ProviderConfig(provider string) (apiKey, baseURL string, err error) {
	if provider == "" {
		return "", "", nil
	}
	err = r.db.QueryRow(`SELECT api_key, base_url FROM providers WHERE name = ?`, provider).
		Scan(&apiKey, &baseURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("read provider config %q: %w", provider, err)
	}
	return apiKey, baseURL, nil
}

// ProviderHeaders resolves the custom request headers for a provider name from
// the provider_headers table (migration 0015). An empty provider name (or one
// with no rows) returns an empty map.
func (r *Repo) ProviderHeaders(provider string) (map[string]string, error) {
	if provider == "" {
		return nil, nil
	}
	rows, err := r.db.Query(
		`SELECT ph.name, ph.value
		 FROM provider_headers ph
		 JOIN providers p ON p.id = ph.provider_id
		 WHERE p.name = ?`, provider)
	if err != nil {
		return nil, fmt.Errorf("read provider headers %q: %w", provider, err)
	}
	defer func() { _ = rows.Close() }()
	headers := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan provider header %q: %w", provider, err)
		}
		headers[name] = value
	}
	return headers, rows.Err()
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

// ContextInjection reports the pre-searched context toggle; anything other
// than the literal "true" (including an absent key) is false, so the feature
// stays off until the user opts in.
func (r *Repo) ContextInjection() (bool, error) {
	value, _, err := r.Setting(KeyContextInjection)
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

// ConversationRetentionDays reports the chat-history auto-delete window in
// days. Anything but a stored integer (absent key, unparseable) falls back to
// DefaultRetentionDays, so a fresh install keeps the documented window without
// a seed row; a stored zero disables auto-delete.
func (r *Repo) ConversationRetentionDays() (int, error) {
	value, _, err := r.Setting(KeyConversationRetentionDays)
	if err != nil {
		return DefaultRetentionDays, err
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return DefaultRetentionDays, nil
	}
	return n, nil
}
