package cli

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/assets"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

// openTestRepos opens a temp store and settings repo on the same db file.
func openTestRepos(t *testing.T) (*store.Store, *settings.Repo) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stg.Close() })
	return st, stg
}

func TestEnsureModelsSeedsFirstBoot(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := ensureModels(st); err != nil {
		t.Fatalf("ensureModels: %v", err)
	}
	models, err := st.ListModels()
	if err != nil {
		t.Fatal(err)
	}
	opts, err := assets.ModelOptions()
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != len(opts) {
		t.Fatalf("seeded %d models, want %d", len(models), len(opts))
	}
}

func TestEnsureModelsReseedsEmptyTable(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := ensureModels(st); err != nil {
		t.Fatalf("first ensureModels: %v", err)
	}
	// The user deletes every model: the next startup seeds the table again
	// (empty means "not there" — the seed runs on every boot).
	models, err := st.ListModels()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range models {
		if err := st.DeleteModel(m.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := ensureModels(st); err != nil {
		t.Fatalf("second ensureModels: %v", err)
	}
	opts, err := assets.ModelOptions()
	if err != nil {
		t.Fatal(err)
	}
	if models, err = st.ListModels(); err != nil || len(models) != len(opts) {
		t.Fatalf("table not reseeded after delete-all: %v (%d models)", err, len(models))
	}
}

func TestEnsureModelsKeepsUserRows(t *testing.T) {
	st, _ := openTestRepos(t)
	if _, err := st.CreateModel("custom", "Custom", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := ensureModels(st); err != nil {
		t.Fatalf("ensureModels: %v", err)
	}
	models, err := st.ListModels()
	if err != nil || len(models) != 1 || models[0].Value != "custom" {
		t.Fatalf("user row not preserved: %v %+v", err, models)
	}
}

func TestServePort(t *testing.T) {
	tests := []struct {
		name string
		dev  bool
		want int
	}{
		{"default", false, config.DefaultPort},
		{"dev", true, config.DevPort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := servePort(tt.dev); got != tt.want {
				t.Fatalf("servePort(%v) = %d, want %d", tt.dev, got, tt.want)
			}
		})
	}
}

func TestOnSettingsSavedSwitchesRootAndRestartsWatcher(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbPath := filepath.Join(t.TempDir(), "test.db")
	// The schema lives in the store's migrations; index.Open issues no DDL.
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	ix, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	oldRoot := filepath.Join(t.TempDir(), "old")
	newRoot := filepath.Join(t.TempDir(), "new")
	if err := wiki.Scaffold(oldRoot); err != nil {
		t.Fatal(err)
	}
	if err := ix.Sync(oldRoot, log); err != nil {
		t.Fatal(err)
	}

	root := newRootHolder(oldRoot)
	w := wiki.New(oldRoot)
	var watched []string
	startWatcher := func(r string) { watched = append(watched, r) }
	flushed := 0
	flush := func() { flushed++ }

	cb := onSettingsSaved(log, root, w, ix, startWatcher, flush)
	if err := cb(newRoot); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if got := root.get(); got != newRoot {
		t.Fatalf("rootHolder = %q, want %q", got, newRoot)
	}
	if w.Root != newRoot {
		t.Fatalf("wiki root = %q, want %q", w.Root, newRoot)
	}
	if len(watched) != 1 || watched[0] != newRoot {
		t.Fatalf("watcher restarted with %v, want [%q]", watched, newRoot)
	}
	if flushed != 1 {
		t.Fatalf("pooled CLI flush ran %d times, want 1", flushed)
	}
	// The new root was scaffolded by the callback.
	if !w.Exists() {
		t.Fatal("new wiki root was not scaffolded")
	}
	// A no-op call (path already current) must not restart the watcher again.
	if err := cb(newRoot); err != nil {
		t.Fatalf("no-op callback: %v", err)
	}
	if len(watched) != 1 {
		t.Fatalf("no-op callback restarted the watcher: %v", watched)
	}
	if flushed != 1 {
		t.Fatalf("no-op callback flushed again: %d", flushed)
	}
}

func TestOnSettingsSavedFailureLeavesRootUntouched(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	ix, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	oldRoot := filepath.Join(t.TempDir(), "old")
	if err := wiki.Scaffold(oldRoot); err != nil {
		t.Fatal(err)
	}
	// newRoot sits under a regular file: Scaffold must fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	newRoot := filepath.Join(blocker, "wiki")

	root := newRootHolder(oldRoot)
	w := wiki.New(oldRoot)
	var watched []string
	startWatcher := func(r string) { watched = append(watched, r) }
	flushed := 0
	flush := func() { flushed++ }

	cb := onSettingsSaved(log, root, w, ix, startWatcher, flush)
	if err := cb(newRoot); err == nil {
		t.Fatal("expected error when scaffold fails")
	}
	if got := root.get(); got != oldRoot {
		t.Fatalf("rootHolder mutated after failure: %q", got)
	}
	if w.Root != oldRoot {
		t.Fatalf("wiki root mutated after failure: %q", w.Root)
	}
	if len(watched) != 0 {
		t.Fatalf("watcher restarted after failure: %v", watched)
	}
	if flushed != 0 {
		t.Fatalf("pooled CLI flushed after failure: %d", flushed)
	}
}

func TestServeErrorWhenWikiScaffoldFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	thothDir := filepath.Join(home, ".thoth")
	if err := os.MkdirAll(thothDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The settings table's wiki_path points under a regular file: MkdirAll
	// in Scaffold fails and serve must abort.
	dbPath := filepath.Join(thothDir, "thoth.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(home, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyWikiPath, filepath.Join(blocker, "wiki")); err != nil {
		t.Fatal(err)
	}
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"serve"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when wiki scaffold fails")
	}
}
