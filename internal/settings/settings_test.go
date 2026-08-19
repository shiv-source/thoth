package settings

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/store"
)

// openTestRepo builds a temp db via the store's migrations (which seed the
// settings defaults) and opens the settings repo on it.
func openTestRepo(t *testing.T) *Repo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(path)
	if err != nil {
		t.Fatalf("OpenRepo: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestOpenSeedsDefaults(t *testing.T) {
	r := openTestRepo(t)
	for key, want := range map[string]string{
		KeyWikiPath:     "~/.thoth/wiki",
		KeyRepoURL:      "",
		KeySyncEnabled:  "false",
		KeyLastSyncedAt: "",
		KeySyncError:    "",
	} {
		got, found, err := r.Setting(key)
		if err != nil || !found || got != want {
			t.Fatalf("Setting(%q) = %q/%v/%v, want %q/true/nil", key, got, found, err, want)
		}
	}
}

func TestModelSettingRoundTrip(t *testing.T) {
	r := openTestRepo(t)
	if _, found, err := r.Setting(KeyModel); err != nil || found {
		t.Fatalf("model must default to absent, found=%v err=%v", found, err)
	}
	if err := r.SetSetting(KeyModel, "claude-haiku-4-5-20251001"); err != nil {
		t.Fatal(err)
	}
	got, found, err := r.Setting(KeyModel)
	if err != nil || !found || got != "claude-haiku-4-5-20251001" {
		t.Fatalf("model round trip = %q/%v/%v", got, found, err)
	}
}

func TestSettingRoundTrip(t *testing.T) {
	r := openTestRepo(t)
	if err := r.SetSetting(KeyRepoURL, "https://github.com/octo/wiki.git"); err != nil {
		t.Fatal(err)
	}
	got, found, err := r.Setting(KeyRepoURL)
	if err != nil || !found || got != "https://github.com/octo/wiki.git" {
		t.Fatalf("after set: %q/%v/%v", got, found, err)
	}
	// Upsert semantics: a second write replaces the value.
	if err := r.SetSetting(KeyRepoURL, ""); err != nil {
		t.Fatal(err)
	}
	if got, found, err = r.Setting(KeyRepoURL); err != nil || !found || got != "" {
		t.Fatalf("after clear: %q/%v/%v", got, found, err)
	}
}

func TestSettingAbsent(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.db.Exec(`DELETE FROM settings WHERE key = ?`, "no-such-key"); err != nil {
		t.Fatal(err)
	}
	got, found, err := r.Setting("no-such-key")
	if err != nil || found || got != "" {
		t.Fatalf("absent key: %q/%v/%v, want empty/false/nil", got, found, err)
	}
}

func TestSyncEnabledParsing(t *testing.T) {
	r := openTestRepo(t)
	// Seeded default is false.
	if on, err := r.SyncEnabled(); err != nil || on {
		t.Fatalf("seeded sync_enabled = %v/%v, want false/nil", on, err)
	}
	for _, in := range []string{"true"} {
		if err := r.SetSetting(KeySyncEnabled, in); err != nil {
			t.Fatal(err)
		}
		if on, err := r.SyncEnabled(); err != nil || !on {
			t.Fatalf("sync_enabled %q = %v/%v, want true/nil", in, on, err)
		}
	}
	for _, in := range []string{"false", "garbage", "TRUE"} {
		if err := r.SetSetting(KeySyncEnabled, in); err != nil {
			t.Fatal(err)
		}
		if on, err := r.SyncEnabled(); err != nil || on {
			t.Fatalf("sync_enabled %q = %v/%v, want false/nil", in, on, err)
		}
	}
}

func TestFoldersParsing(t *testing.T) {
	r := openTestRepo(t)
	// Absent key falls back to nil.
	folders, err := r.Folders()
	if err != nil || folders != nil {
		t.Fatalf("absent wiki_folders = %v/%v, want nil/nil", folders, err)
	}
	// Comma-separated with whitespace is parsed and trimmed.
	if err := r.SetSetting(KeyWikiFolders, "journal, recipes , notes"); err != nil {
		t.Fatal(err)
	}
	folders, err = r.Folders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 3 || folders[0] != "journal" || folders[1] != "recipes" || folders[2] != "notes" {
		t.Fatalf("parsed folders = %v, want [journal recipes notes]", folders)
	}
	// An empty value means the defaults again.
	if err := r.SetSetting(KeyWikiFolders, ""); err != nil {
		t.Fatal(err)
	}
	if folders, err = r.Folders(); err != nil || folders != nil {
		t.Fatalf("empty wiki_folders = %v/%v, want nil/nil", folders, err)
	}
	// Stray commas produce no empty entries.
	if err := r.SetSetting(KeyWikiFolders, ",,,"); err != nil {
		t.Fatal(err)
	}
	if folders, err = r.Folders(); err != nil || len(folders) != 0 {
		t.Fatalf("comma-only wiki_folders = %v/%v, want []/nil", folders, err)
	}
}

func TestRepoClosedErrors(t *testing.T) {
	r := openTestRepo(t)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Setting(KeyWikiPath); err == nil {
		t.Fatal("Setting on closed repo must error")
	}
	if err := r.SetSetting(KeyWikiPath, "x"); err == nil {
		t.Fatal("SetSetting on closed repo must error")
	}
}

func TestSyncStateRoundTrip(t *testing.T) {
	r := openTestRepo(t)
	// Seeded state is empty.
	last, syncErr, err := r.SyncState()
	if err != nil || last != "" || syncErr != "" {
		t.Fatalf("seeded sync state = %q/%q/%v", last, syncErr, err)
	}

	// Failure records the error and keeps last_synced_at empty.
	if err := r.SetSyncResult(false, "push rejected"); err != nil {
		t.Fatal(err)
	}
	if last, syncErr, err = r.SyncState(); err != nil || last != "" || syncErr != "push rejected" {
		t.Fatalf("after failure: %q/%q/%v", last, syncErr, err)
	}

	// Success stamps last_synced_at and clears the error.
	if err := r.SetSyncResult(true, ""); err != nil {
		t.Fatal(err)
	}
	last, syncErr, err = r.SyncState()
	if err != nil || last == "" || !strings.HasSuffix(last, "Z") || syncErr != "" {
		t.Fatalf("after success: %q/%q/%v", last, syncErr, err)
	}

	// A later failure keeps the last successful timestamp.
	first := last
	if err := r.SetSyncResult(false, "offline"); err != nil {
		t.Fatal(err)
	}
	if last, syncErr, err = r.SyncState(); err != nil || last != first || syncErr != "offline" {
		t.Fatalf("after later failure: %q/%q/%v", last, syncErr, err)
	}
}
