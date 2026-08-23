package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func openStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSyncProviderCRUD(t *testing.T) {
	s := openStore(t)

	// The migration seeds the four built-ins; user rows are added on top.
	builtins, err := s.ListSyncProviders()
	if err != nil {
		t.Fatalf("ListSyncProviders: %v", err)
	}
	if len(builtins) != 4 {
		t.Fatalf("expected 4 built-ins, got %+v", builtins)
	}

	p, err := s.CreateSyncProvider("gh-enterprise", "GitHub Enterprise", "github", "https://ghe.example.com")
	if err != nil {
		t.Fatalf("CreateSyncProvider: %v", err)
	}
	if p.Protected || p.Slug != "gh-enterprise" || p.Driver != "github" {
		t.Fatalf("created provider wrong: %+v", p)
	}

	got, err := s.SyncProvider(p.ID)
	if err != nil {
		t.Fatalf("SyncProvider: %v", err)
	}
	if got.BaseURL != "https://ghe.example.com" {
		t.Fatalf("provider base_url = %q", got.BaseURL)
	}
	bySlug, err := s.SyncProviderBySlug("gh-enterprise")
	if err != nil || bySlug.ID != p.ID {
		t.Fatalf("SyncProviderBySlug: %v %+v", err, bySlug)
	}

	if err := s.UpdateSyncProvider(p.ID, "GH Enterprise", "https://ghe2.example.com"); err != nil {
		t.Fatalf("UpdateSyncProvider: %v", err)
	}
	got, _ = s.SyncProvider(p.ID)
	if got.Name != "GH Enterprise" || got.BaseURL != "https://ghe2.example.com" {
		t.Fatalf("provider not updated: %+v", got)
	}

	if err := s.DeleteSyncProvider(p.ID); err != nil {
		t.Fatalf("DeleteSyncProvider: %v", err)
	}
	if _, err := s.SyncProvider(p.ID); !errors.Is(err, ErrSyncProviderNotFound) {
		t.Fatalf("deleted provider still readable: %v", err)
	}
}

func TestSyncProviderDuplicateSlug(t *testing.T) {
	s := openStore(t)
	if _, err := s.CreateSyncProvider("dup", "First", "github", ""); err != nil {
		t.Fatalf("CreateSyncProvider: %v", err)
	}
	_, err := s.CreateSyncProvider("dup", "Second", "github", "")
	if !errors.Is(err, ErrSyncProviderExists) {
		t.Fatalf("duplicate slug = %v, want ErrSyncProviderExists", err)
	}
}

// TestEnsureSyncProvider pins the idempotent seed helper: creating an absent
// row and returning the existing one on a repeat call.
func TestEnsureSyncProvider(t *testing.T) {
	s := openStore(t)
	p, err := s.EnsureSyncProvider("custom", "Custom", "github", "", false)
	if err != nil {
		t.Fatalf("EnsureSyncProvider: %v", err)
	}
	if p.Slug != "custom" || p.Driver != "github" || p.Protected {
		t.Fatalf("ensured provider wrong: %+v", p)
	}
	again, err := s.EnsureSyncProvider("custom", "Custom", "github", "", false)
	if err != nil || again.ID != p.ID {
		t.Fatalf("EnsureSyncProvider not idempotent: %+v / %v", again, err)
	}
	// The seed's protected flag lands on the row.
	local, err := s.EnsureSyncProvider("local", "Local backup", "local", "", true)
	if err != nil {
		t.Fatalf("EnsureSyncProvider(local): %v", err)
	}
	if !local.Protected {
		t.Fatalf("protected seed flag not stored: %+v", local)
	}
}

func TestSyncProviderProtectedLocked(t *testing.T) {
	s := openStore(t)
	local, err := s.SyncProviderBySlug("local")
	if err != nil {
		t.Fatalf("local provider missing: %v", err)
	}
	if err := s.UpdateSyncProvider(local.ID, "renamed", ""); !errors.Is(err, ErrSyncProviderProtected) {
		t.Fatalf("update protected = %v, want ErrSyncProviderProtected", err)
	}
	if err := s.DeleteSyncProvider(local.ID); !errors.Is(err, ErrSyncProviderProtected) {
		t.Fatalf("delete protected = %v, want ErrSyncProviderProtected", err)
	}
}

func TestSyncProviderInUseCannotDelete(t *testing.T) {
	s := openStore(t)
	p, err := s.CreateSyncProvider("inuse", "In Use", "github", "")
	if err != nil {
		t.Fatalf("CreateSyncProvider: %v", err)
	}
	if _, err := s.CreateConnection(p.ID, "mine", `{"token":"t"}`, `{"username":"u"}`, true); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.DeleteSyncProvider(p.ID); !errors.Is(err, ErrSyncProviderInUse) {
		t.Fatalf("delete provider with connections = %v, want ErrSyncProviderInUse", err)
	}
}

// TestSyncProviderSeedMatchesMigration pins the two built-in sources in sync:
// the migration's inline seed and assets/sync-providers.json must agree, so
// editing the JSON without the migration (or vice versa) fails loudly.
func TestSyncProviderSeedMatchesMigration(t *testing.T) {
	s := openStore(t)
	want := map[string]bool{"github": true, "gitlab": true, "aws_s3": true, "local": true}
	providers, err := s.ListSyncProviders()
	if err != nil {
		t.Fatalf("ListSyncProviders: %v", err)
	}
	if len(providers) != len(want) {
		t.Fatalf("seeded providers = %d, want %d", len(providers), len(want))
	}
	for _, p := range providers {
		if !want[p.Slug] {
			t.Fatalf("unexpected seeded provider %q", p.Slug)
		}
		if p.Driver == "" {
			t.Fatalf("provider %s has no driver", p.Slug)
		}
	}
}
