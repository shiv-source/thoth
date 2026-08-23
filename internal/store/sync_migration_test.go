package store

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"
)

// TestMigration0014BackfillsAttemptAt upgrades a v13 database (a user on the
// previous schema) and verifies migration 0014 adds last_attempt_at and
// backfills it from last_synced_at — a synced connection keeps its cooldown
// instead of re-firing right after the upgrade, while a never-synced one stays
// due (empty attempt).
func TestMigration0014BackfillsAttemptAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	// Apply everything through 0013 — the pre-0014 state.
	for i, name := range names[:13] {
		raw, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(db, i+1, splitStatements(string(raw))); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	var localID int64
	if err := db.QueryRow(`SELECT id FROM sync_providers WHERE slug = 'local'`).Scan(&localID); err != nil {
		t.Fatal(err)
	}
	// One synced connection (last_synced_at set) and one never-synced.
	lastSynced := "2026-01-02T03:04:05Z"
	if _, err := db.Exec(
		`INSERT INTO sync_connections(provider_id, name, config, identity, enabled, protected, last_synced_at, last_error, created_at, updated_at)
		 VALUES (?, 'synced', '{}', NULL, 1, 0, ?, '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		localID, lastSynced); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO sync_connections(provider_id, name, config, identity, enabled, protected, created_at, updated_at)
		 VALUES (?, 'never', '{}', NULL, 1, 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		localID); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path) // runs migration 0014
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()
	conns, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	for _, c := range conns {
		switch c.Name {
		case "synced":
			if c.LastAttemptAt != lastSynced {
				t.Fatalf("synced connection attempt not backfilled: %+v", c)
			}
		case "never":
			if c.LastAttemptAt != "" {
				t.Fatalf("never-synced connection must keep an empty attempt: %+v", c)
			}
		}
	}
}

// TestSyncMigration0012 reproduces a pre-0012 database — the single github_auth
// row plus the four github_sync_* settings keys — and verifies migration 0012
// promotes it: a connection under the github provider carrying the token, sync
// target, enabled flag, and identity; the built-in provider catalog seeded
// (local protected); and the legacy table and keys gone.
func TestSyncMigration0012(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	// Apply everything through 0011 — the pre-sync state.
	for i, name := range names[:11] {
		raw, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(db, i+1, splitStatements(string(raw))); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	// A connected GitHub account and its sync state.
	if _, err := db.Exec(`
		INSERT INTO github_auth
			(id, token, username, display_name, email, avatar_url, scopes, expires_at, profile_url, account_created_at, account_updated_at, created_at, updated_at)
		VALUES
			(1, 'ghp_secret', 'octo', 'Octo Cat', 'octo@example.com', 'https://avatars/octo', 'user:email repo', '', 'https://github.com/octo', '2020-01-01T00:00:00Z', '2021-01-01T00:00:00Z', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"github_sync_repo":      "https://github.com/octo/wiki.git",
		"github_sync_enabled":   "true",
		"github_last_synced_at": "2024-01-02T00:00:00Z",
		"github_sync_error":     "",
	} {
		if _, err := db.Exec(`UPDATE settings SET value = ? WHERE key = ?`, value, key); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	// The account became a connection under the github provider, carrying the
	// token and the sync state that used to live in the settings keys.
	conns, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %+v", conns)
	}
	c := conns[0]
	if c.ProviderSlug != "github" || c.ProviderDriver != "github" || !c.Enabled || c.Protected {
		t.Fatalf("migrated connection wrong: %+v", c)
	}
	if c.LastSyncedAt != "2024-01-02T00:00:00Z" || c.LastError != "" {
		t.Fatalf("migrated sync state wrong: last=%q err=%q", c.LastSyncedAt, c.LastError)
	}
	var cfg, ident struct {
		Token    string `json:"token"`
		RepoURL  string `json:"repo_url"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
		t.Fatalf("parse migrated config: %v", err)
	}
	if cfg.Token != "ghp_secret" || cfg.RepoURL != "https://github.com/octo/wiki.git" {
		t.Fatalf("migrated config wrong: %+v", cfg)
	}
	if err := json.Unmarshal([]byte(c.Identity), &ident); err != nil {
		t.Fatalf("parse migrated identity: %v", err)
	}
	if ident.Username != "octo" || ident.Email != "octo@example.com" {
		t.Fatalf("migrated identity wrong: %+v", ident)
	}

	// The built-in catalog is seeded; local is protected.
	providers, err := s.ListSyncProviders()
	if err != nil {
		t.Fatalf("ListSyncProviders: %v", err)
	}
	if len(providers) != 4 {
		t.Fatalf("expected 4 built-in providers, got %+v", providers)
	}
	local, err := s.SyncProviderBySlug("local")
	if err != nil {
		t.Fatalf("local provider missing: %v", err)
	}
	if !local.Protected {
		t.Fatalf("local provider must be protected: %+v", local)
	}

	// The legacy table and keys are gone — the cutover is complete.
	var settingsCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM settings WHERE key IN ('github_sync_repo','github_sync_enabled','github_last_synced_at','github_sync_error')`).Scan(&settingsCount); err != nil {
		t.Fatal(err)
	}
	if settingsCount != 0 {
		t.Fatalf("legacy github_sync_* settings keys survived: %d", settingsCount)
	}
	var tableCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'github_auth'`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Fatalf("github_auth table survived migration 0012")
	}

	// Reopening is stable — the connection and its state persist.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s.Close() }()
	again, err := s.Connection(c.ID)
	if err != nil {
		t.Fatalf("connection lost on reopen: %v", err)
	}
	if again.Config != c.Config || again.LastSyncedAt != c.LastSyncedAt {
		t.Fatalf("connection changed on reopen: %+v", again)
	}
}
