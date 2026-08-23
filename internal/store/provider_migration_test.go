package store

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"

	"github.com/shiv-source/thoth/internal/settings"
)

// TestProviderMigration0011 reproduces a pre-0011 database — llm_models rows
// carrying free-text provider labels, credentials in the legacy
// provider_<slug>_* settings keys — and verifies migration 0011 + the
// credential backfill promote it: providers rows exist, models point at them,
// credentials copy over, and the legacy keys and provider column are gone.
func TestProviderMigration0011(t *testing.T) {
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
	// Apply everything through 0010 — the pre-providers-table state.
	for i, name := range names[:10] {
		raw, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(db, i+1, splitStatements(string(raw))); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	// A registry with two providers (one with a key + base url, one bare).
	if _, err := db.Exec(
		`INSERT INTO llm_models(value, name, tag, provider) VALUES
		 ('deepseek-v4-flash', 'V4 Flash', 'fastest', 'DeepSeek'),
		 ('deepseek-v4-pro[1m]', 'V4 Pro', 'strongest', 'DeepSeek'),
		 ('claude-sonnet-5', 'Sonnet 5', 'balanced', 'Anthropic'),
		 ('orphan', 'Orphan', '', '')`); err != nil {
		t.Fatal(err)
	}
	// Legacy per-provider credentials, keyed by the provider slug.
	keys := map[string]string{
		settings.ProviderAPIKeyKey("DeepSeek"):   "ds-secret",
		settings.ProviderBaseURLKey("DeepSeek"):  "https://api.deepseek.com",
		settings.ProviderAPIKeyKey("Anthropic"):  "",
		settings.ProviderBaseURLKey("Anthropic"): "",
	}
	for k, v := range keys {
		if _, err := db.Exec(`INSERT INTO settings(key, value) VALUES (?, ?)`, k, v); err != nil {
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

	// Providers were created from the distinct labels, with credentials
	// copied from the legacy keys.
	providers, err := s.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %+v", providers)
	}
	ds, err := s.ProviderByName("DeepSeek")
	if err != nil {
		t.Fatalf("DeepSeek provider missing: %v", err)
	}
	if ds.APIKey != "ds-secret" || ds.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("DeepSeek credentials not migrated: %+v", ds)
	}
	an, err := s.ProviderByName("Anthropic")
	if err != nil {
		t.Fatalf("Anthropic provider missing: %v", err)
	}
	if an.APIKey != "" || an.BaseURL != "" {
		t.Fatalf("Anthropic should have empty credentials: %+v", an)
	}
	for _, p := range providers {
		switch p.Name {
		case "DeepSeek":
			if p.ModelCount != 2 {
				t.Fatalf("DeepSeek model count = %d, want 2", p.ModelCount)
			}
		case "Anthropic":
			if p.ModelCount != 1 {
				t.Fatalf("Anthropic model count = %d, want 1", p.ModelCount)
			}
		}
	}

	// Models point at their provider rows; the orphan stays unassigned.
	models, err := s.ListModels()
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 4 {
		t.Fatalf("expected 4 models, got %+v", models)
	}
	for _, m := range models {
		switch m.Value {
		case "deepseek-v4-flash", "deepseek-v4-pro[1m]":
			if m.Provider != "DeepSeek" || m.ProviderID != ds.ID {
				t.Fatalf("model %q not pointed at DeepSeek: %+v", m.Value, m)
			}
		case "claude-sonnet-5":
			if m.Provider != "Anthropic" || m.ProviderID != an.ID {
				t.Fatalf("model %q not pointed at Anthropic: %+v", m.Value, m)
			}
		case "orphan":
			if m.Provider != "" || m.ProviderID != 0 {
				t.Fatalf("orphan must stay unassigned: %+v", m)
			}
		}
	}

	// The legacy keys and the provider column are gone.
	var keyCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key LIKE 'provider\_%' ESCAPE '\'`).Scan(&keyCount); err != nil {
		t.Fatal(err)
	}
	if keyCount != 0 {
		t.Fatalf("legacy provider settings keys survived: %d", keyCount)
	}
	var providerCol int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('llm_models') WHERE name = 'provider'`).Scan(&providerCol); err != nil {
		t.Fatal(err)
	}
	if providerCol != 0 {
		t.Fatalf("llm_models.provider column survived migration 0011")
	}

	// The backfill is idempotent: reopening changes nothing.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s.Close() }()
	again, err := s.ProviderByName("DeepSeek")
	if err != nil || again.APIKey != "ds-secret" {
		t.Fatalf("credentials changed on reopen: %+v / %v", again, err)
	}
}
