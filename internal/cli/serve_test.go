package cli

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestThothDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tests := []struct {
		name string
		dev  bool
		want string
	}{
		{"prod", false, filepath.Join(home, ".thoth")},
		{"dev", true, filepath.Join(home, ".thoth", "dev")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := thothDir(tt.dev)
			if err != nil {
				t.Fatalf("thothDir(%v): %v", tt.dev, err)
			}
			if got != tt.want {
				t.Fatalf("thothDir(%v) = %q, want %q", tt.dev, got, tt.want)
			}
			if st, err := os.Stat(got); err != nil || !st.IsDir() {
				t.Fatalf("thothDir(%v) did not create %q: %v", tt.dev, got, err)
			}
		})
	}
}

func TestDefaultWikiPath(t *testing.T) {
	tests := []struct {
		name string
		dev  bool
		dir  string
		want string
	}{
		{"prod keeps the shared default", false, "/tmp/.thoth", settings.DefaultWikiPath},
		{"dev falls back inside the dev data dir", true, "/tmp/.thoth/dev", "/tmp/.thoth/dev/wiki"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultWikiPath(tt.dev, tt.dir); got != tt.want {
				t.Fatalf("defaultWikiPath(%v, %q) = %q, want %q", tt.dev, tt.dir, got, tt.want)
			}
		})
	}
}

func TestResolveWikiPath(t *testing.T) {
	tests := []struct {
		name   string
		dev    bool
		dir    string
		stored string
		found  bool
		want   string
	}{
		{"missing row falls back", false, "/tmp/.thoth", "", false, settings.DefaultWikiPath},
		{"empty value falls back", false, "/tmp/.thoth", "", true, settings.DefaultWikiPath},
		{"stored path wins in prod", false, "/tmp/.thoth", "~/notes", true, "~/notes"},
		{"stored path wins in dev", true, "/tmp/.thoth/dev", "~/notes", true, "~/notes"},
		{"dev overrides the seeded prod default", true, "/tmp/.thoth/dev", settings.DefaultWikiPath, true, "/tmp/.thoth/dev/wiki"},
		{"prod keeps the seeded default", false, "/tmp/.thoth", settings.DefaultWikiPath, true, settings.DefaultWikiPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveWikiPath(tt.dev, tt.dir, tt.stored, tt.found); got != tt.want {
				t.Fatalf("resolveWikiPath(%v, %q, %q, %v) = %q, want %q", tt.dev, tt.dir, tt.stored, tt.found, got, tt.want)
			}
		})
	}
}

func TestSettleWikiPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".thoth", "dev")

	t.Run("dev rewrites the seeded prod default", func(t *testing.T) {
		_, stg := openTestRepos(t)
		got, err := settleWikiPath(stg, true, dir)
		if err != nil {
			t.Fatalf("settleWikiPath: %v", err)
		}
		if got != filepath.Join(dir, "wiki") {
			t.Fatalf("settleWikiPath = %q, want %q", got, filepath.Join(dir, "wiki"))
		}
		stored, found, err := stg.Setting(settings.KeyWikiPath)
		if err != nil || !found {
			t.Fatalf("stored setting: %v found=%v", err, found)
		}
		if stored != "~/.thoth/dev/wiki" {
			t.Fatalf("stored = %q, want ~/.thoth/dev/wiki", stored)
		}
	})

	t.Run("prod keeps the seeded default", func(t *testing.T) {
		_, stg := openTestRepos(t)
		got, err := settleWikiPath(stg, false, filepath.Join(home, ".thoth"))
		if err != nil {
			t.Fatalf("settleWikiPath: %v", err)
		}
		if got != settings.DefaultWikiPath {
			t.Fatalf("settleWikiPath = %q, want %q", got, settings.DefaultWikiPath)
		}
	})

	t.Run("a custom stored path is never rewritten", func(t *testing.T) {
		_, stg := openTestRepos(t)
		if err := stg.SetSetting(settings.KeyWikiPath, "~/notes"); err != nil {
			t.Fatal(err)
		}
		got, err := settleWikiPath(stg, true, dir)
		if err != nil {
			t.Fatalf("settleWikiPath: %v", err)
		}
		if got != "~/notes" {
			t.Fatalf("settleWikiPath = %q, want ~/notes", got)
		}
		stored, _, _ := stg.Setting(settings.KeyWikiPath)
		if stored != "~/notes" {
			t.Fatalf("stored = %q, want ~/notes", stored)
		}
	})
}

func TestDevCommit(t *testing.T) {
	t.Run("returns the full commit id inside a repo", func(t *testing.T) {
		dir := t.TempDir()
		run := func(args ...string) string {
			out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
			if err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
			return strings.TrimSpace(string(out))
		}
		run("init", "-q", "-b", "main")
		run("config", "user.email", "test@example.com")
		run("config", "user.name", "test")
		if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", "f")
		run("commit", "-q", "-m", "c")
		want := run("rev-parse", "HEAD")
		got := devCommit(dir)
		if got != want {
			t.Fatalf("devCommit(%q) = %q, want %q", dir, got, want)
		}
		if len(got) != 40 {
			t.Fatalf("devCommit(%q) = %q, want 40 hex chars", dir, got)
		}
	})
	t.Run("empty outside a repo", func(t *testing.T) {
		if got := devCommit(t.TempDir()); got != "" {
			t.Fatalf("devCommit = %q, want empty", got)
		}
	})
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
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stg.Close() })
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

	cb := onSettingsSaved(log, stg, root, w, ix, startWatcher, flush)
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
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stg.Close() })
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

	cb := onSettingsSaved(log, stg, root, w, ix, startWatcher, flush)
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
