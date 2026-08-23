package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestConnectionRoundTrip(t *testing.T) {
	s := openStore(t)
	p, err := s.SyncProviderBySlug("github")
	if err != nil {
		t.Fatalf("github provider missing: %v", err)
	}

	c, err := s.CreateConnection(p.ID, "work", `{"token":"t","repo_url":"https://github.com/x/w.git"}`, `{"username":"octo"}`, true)
	if err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if c.ProviderSlug != "github" || c.ProviderName != "GitHub" || c.ProviderDriver != "github" {
		t.Fatalf("connection provider join wrong: %+v", c)
	}
	if !c.Enabled || c.Protected {
		t.Fatalf("connection flags wrong: %+v", c)
	}

	// Update keeps the provider join; sync result stamps/preserves state.
	if err := s.UpdateConnection(c.ID, "renamed", `{"token":"t2","repo_url":"https://github.com/x/w.git"}`, false); err != nil {
		t.Fatalf("UpdateConnection: %v", err)
	}
	if err := s.SetConnectionSyncResult(c.ID, true, ""); err != nil {
		t.Fatalf("SetConnectionSyncResult(ok): %v", err)
	}
	got, err := s.Connection(c.ID)
	if err != nil {
		t.Fatalf("Connection: %v", err)
	}
	if got.Name != "renamed" || got.Enabled || got.LastSyncedAt == "" || got.LastError != "" {
		t.Fatalf("connection after update+sync: %+v", got)
	}
	lastSynced := got.LastSyncedAt

	// A failure preserves the last success and records the error.
	if err := s.SetConnectionSyncResult(c.ID, false, "upstream rejected"); err != nil {
		t.Fatalf("SetConnectionSyncResult(fail): %v", err)
	}
	got, _ = s.Connection(c.ID)
	if got.LastSyncedAt != lastSynced || got.LastError != "upstream rejected" {
		t.Fatalf("sync failure state wrong: %+v", got)
	}

	// Every outcome lands in the push history, newest first.
	history, err := s.ListPushHistory(c.ID)
	if err != nil {
		t.Fatalf("ListPushHistory: %v", err)
	}
	if len(history) != 2 || history[0].OK || history[0].Error != "upstream rejected" || !history[1].OK {
		t.Fatalf("push history wrong: %+v", history)
	}

	// List returns every connection with its provider joined.
	all, err := s.ListConnections()
	if err != nil || len(all) != 1 {
		t.Fatalf("ListConnections: %v %+v", err, all)
	}

	if err := s.DeleteConnection(c.ID); err != nil {
		t.Fatalf("DeleteConnection: %v", err)
	}
	if _, err := s.Connection(c.ID); !errors.Is(err, ErrConnectionNotFound) {
		t.Fatalf("deleted connection still readable: %v", err)
	}
}

func TestEnsureLocalBackupCreatesOnce(t *testing.T) {
	s := openStore(t)

	// Fresh database: the protected local backup connection is created with
	// no folder configured yet.
	c, err := s.EnsureLocalBackup()
	if err != nil {
		t.Fatalf("EnsureLocalBackup: %v", err)
	}
	if !c.Protected || c.Name != "Local backup" || !c.Enabled {
		t.Fatalf("local backup wrong: %+v", c)
	}
	if c.ProviderSlug != "local" {
		t.Fatalf("local backup provider = %q", c.ProviderSlug)
	}
	var cfg map[string]string
	if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
		t.Fatalf("parse local backup config: %v", err)
	}
	if _, hasPath := cfg["path"]; hasPath || len(cfg) != 0 {
		t.Fatalf("local backup must start with an empty config: %+v", cfg)
	}

	// A second call returns the same row — never a duplicate.
	again, err := s.EnsureLocalBackup()
	if err != nil || again.ID != c.ID {
		t.Fatalf("EnsureLocalBackup duplicated: %+v / %v", again, err)
	}

	// The local backup cannot be deleted, but it is editable (folder change).
	if err := s.DeleteConnection(c.ID); !errors.Is(err, ErrConnectionProtected) {
		t.Fatalf("delete protected connection = %v, want ErrConnectionProtected", err)
	}
	if err := s.UpdateConnection(c.ID, "Local backup", `{"path":"/Volumes/Backup"}`, true); err != nil {
		t.Fatalf("editing the local backup folder must be allowed: %v", err)
	}
	got, _ := s.Connection(c.ID)
	if !strings.Contains(got.Config, "/Volumes/Backup") {
		t.Fatalf("local backup folder not persisted: %+v", got)
	}
}

func TestConnectionMissingProviderFails(t *testing.T) {
	s := openStore(t)
	if _, err := s.CreateConnection(999999, "ghost", "{}", "", true); !errors.Is(err, ErrSyncProviderNotFound) {
		t.Fatalf("create connection with missing provider = %v, want ErrSyncProviderNotFound", err)
	}
}

func TestDeleteConnectionMissing(t *testing.T) {
	s := openStore(t)
	if err := s.DeleteConnection(999999); !errors.Is(err, ErrConnectionNotFound) {
		t.Fatalf("delete missing connection = %v, want ErrConnectionNotFound", err)
	}
}

func TestEnsureLocalBackupMissingProvider(t *testing.T) {
	s := openStore(t)
	// A wiped catalog (raw delete — the local row is protected through the
	// API) makes the backup guard fail loudly instead of attaching nowhere.
	if _, err := s.db.Exec(`DELETE FROM sync_providers`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnsureLocalBackup(); err == nil {
		t.Fatal("EnsureLocalBackup without the local provider must error")
	}
}
