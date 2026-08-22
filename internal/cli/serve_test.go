package cli

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentgit "github.com/shiv-source/thoth/agent/git"
	"github.com/shiv-source/thoth/internal/assets"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/github"
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
		repo, err := agentgit.Init(dir)
		if err != nil {
			t.Fatalf("Init: %v", err)
		}
		id := agentgit.Identity{Name: "test", Email: "test@example.com"}
		if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		committed, err := repo.CommitAll("c", id)
		if err != nil {
			t.Fatalf("CommitAll: %v", err)
		}
		if !committed {
			t.Fatal("CommitAll: nothing committed")
		}
		want, err := repo.Head()
		if err != nil {
			t.Fatalf("Head: %v", err)
		}
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

	w := wiki.New(oldRoot)
	var watched []string
	startWatcher := func(r string) { watched = append(watched, r) }

	cb := onSettingsSaved(log, stg, w, ix, startWatcher)
	if err := cb(newRoot); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if w.Root() != newRoot {
		t.Fatalf("wiki root = %q, want %q", w.Root(), newRoot)
	}
	if len(watched) != 1 || watched[0] != newRoot {
		t.Fatalf("watcher restarted with %v, want [%q]", watched, newRoot)
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

	w := wiki.New(oldRoot)
	var watched []string
	startWatcher := func(r string) { watched = append(watched, r) }

	cb := onSettingsSaved(log, stg, w, ix, startWatcher)
	if err := cb(newRoot); err == nil {
		t.Fatal("expected error when scaffold fails")
	}
	if w.Root() != oldRoot {
		t.Fatalf("wiki root mutated after failure: %q", w.Root())
	}
	if len(watched) != 0 {
		t.Fatalf("watcher restarted after failure: %v", watched)
	}
}

// TestEnsureWikiRecreatesReservedAttachments covers the reserved-dir
// guarantee: a wiki that predates attachments/ (or whose attachments/ was
// deleted) gets the directory back on every startup.
func TestEnsureWikiRecreatesReservedAttachments(t *testing.T) {
	_, stg := openTestRepos(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Scaffolded with a custom folder set that predates attachments/.
	root := t.TempDir()
	if err := wiki.ScaffoldWithOptions(root, wiki.ScaffoldOptions{Folders: []string{"notes"}, GitInit: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "attachments")); !os.IsNotExist(err) {
		t.Fatal("precondition: attachments must not exist yet")
	}

	w, err := ensureWiki(root, stg, log)
	if err != nil {
		t.Fatalf("ensureWiki: %v", err)
	}
	if w.Root() != root {
		t.Fatalf("wiki root = %q, want %q", w.Root(), root)
	}
	if _, err := os.Stat(filepath.Join(root, "attachments")); err != nil {
		t.Fatalf("ensureWiki must create the reserved attachments dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "attachments", ".gitkeep")); err != nil {
		t.Fatalf("reserved attachments dir must carry a .gitkeep: %v", err)
	}
}

// TestGitToolOptions asserts the agent git tools are wired to the live wiki
// root, the sync switch (Guard), and the stored GitHub connection (Auth and
// Identity), all evaluated lazily so a connection change applies without
// restart.
func TestGitToolOptions(t *testing.T) {
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
	gh, err := github.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gh.Close() })

	root := t.TempDir()
	w := wiki.New(root)
	opts := gitToolOptions(w, stg, gh)

	// RepoPath follows the live wiki root (it can move).
	if opts.RepoPath == nil {
		t.Fatal("RepoPath must be set")
	}
	if got := opts.RepoPath(); got != root {
		t.Fatalf("RepoPath() = %q, want %q", got, root)
	}

	// Guard: disabled blocks, enabled passes.
	if err := opts.Guard(); err == nil || !strings.Contains(err.Error(), "sync is disabled") {
		t.Fatalf("Guard (disabled) = %v, want sync-disabled error", err)
	}
	if err := stg.SetSetting(settings.KeySyncEnabled, "true"); err != nil {
		t.Fatal(err)
	}
	if err := opts.Guard(); err != nil {
		t.Fatalf("Guard (enabled) = %v, want nil", err)
	}

	// Auth and identity error cleanly before a connection exists.
	if _, err := opts.Auth(); err == nil || !strings.Contains(err.Error(), "no GitHub connection") {
		t.Fatalf("Auth (no connection) = %v, want connection error", err)
	}
	if _, err := opts.Identity(); err == nil || !strings.Contains(err.Error(), "no GitHub connection") {
		t.Fatalf("Identity (no connection) = %v, want connection error", err)
	}

	// The stored connection supplies push credentials and the identity; the
	// display name overrides the username as the committer name.
	if err := gh.Save(github.Auth{Token: "tok", Username: "user", Email: "u@e.com"}); err != nil {
		t.Fatal(err)
	}
	a, err := opts.Auth()
	if err != nil {
		t.Fatal(err)
	}
	if a.Token != "tok" || a.Username != "user" {
		t.Fatalf("Auth = %+v, want stored credentials", a)
	}
	id, err := opts.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if id.Name != "user" || id.Email != "u@e.com" {
		t.Fatalf("Identity = %+v, want username fallback", id)
	}
	if err := gh.Save(github.Auth{Token: "t2", Username: "user", DisplayName: "Real Name", Email: "u@e.com"}); err != nil {
		t.Fatal(err)
	}
	id, err = opts.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if id.Name != "Real Name" || id.Email != "u@e.com" {
		t.Fatalf("Identity = %+v, want display name", id)
	}
}

// TestEnsureWikiGitInitsPreExistingWiki covers the EnsureGitRepo step: a
// wiki that exists but was never versioned is git-inited on startup, so the
// agent's git tools always have a repository to act on.
func TestEnsureWikiGitInitsPreExistingWiki(t *testing.T) {
	_, stg := openTestRepos(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	root := t.TempDir()
	if err := wiki.ScaffoldWithOptions(root, wiki.ScaffoldOptions{Folders: []string{"notes"}, GitInit: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatal("precondition: wiki must not be a git repo")
	}

	w, err := ensureWiki(root, stg, log)
	if err != nil {
		t.Fatalf("ensureWiki: %v", err)
	}
	if w.Root() != root {
		t.Fatalf("wiki root = %q, want %q", w.Root(), root)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatalf("ensureWiki must git-init a pre-existing wiki: %v", err)
	}
}

// TestEnsureWikiErrorWhenAttachmentsBlocked covers the fail-fast side of the
// reserved-dir guarantee: a file squatting on the attachments name must abort
// boot with an actionable error instead of a bare "not a directory".
func TestEnsureWikiErrorWhenAttachmentsBlocked(t *testing.T) {
	_, stg := openTestRepos(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	root := t.TempDir()
	if err := wiki.ScaffoldWithOptions(root, wiki.ScaffoldOptions{Folders: []string{"notes"}, GitInit: false}); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(root, "attachments")
	if err := os.WriteFile(blocker, []byte("oops"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ensureWiki(root, stg, log); err == nil {
		t.Fatal("expected error when a file blocks the reserved attachments dir")
	} else if !strings.Contains(err.Error(), "blocked by a file") || !strings.Contains(err.Error(), blocker) {
		t.Fatalf("error must name the conflict and the blocking path, got: %v", err)
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

// TestModelProvider resolves the llm_models provider label for a model value.
func TestModelProvider(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := ensureModels(st); err != nil {
		t.Fatalf("ensureModels: %v", err)
	}
	tests := []struct{ value, want string }{
		{"deepseek-v4-flash", "DeepSeek"},
		{"claude-opus-4-8", "Anthropic"},
		{"", ""},
		{"unknown-model", ""},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := modelProvider(st, tt.value)
			if err != nil {
				t.Fatalf("modelProvider(%q): %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("modelProvider(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// TestServeProviderConfigResolution covers the boot chain model → provider →
// config: the model's registry row names the provider, whose per-provider
// settings resolve from it (own key/base URL; no shared fallback).
func TestServeProviderConfigResolution(t *testing.T) {
	st, stg := openTestRepos(t)
	if err := ensureModels(st); err != nil {
		t.Fatalf("ensureModels: %v", err)
	}

	// A model with a configured provider: per-provider key + base URL win.
	if err := stg.SetSetting(settings.ProviderAPIKeyKey("DeepSeek"), "ds-key"); err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.ProviderBaseURLKey("DeepSeek"), "https://api.deepseek.com"); err != nil {
		t.Fatal(err)
	}
	provider, err := modelProvider(st, "deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	apiKey, baseURL, err := stg.ProviderConfig(provider)
	if err != nil {
		t.Fatal(err)
	}
	if provider != "DeepSeek" || apiKey != "ds-key" || baseURL != "https://api.deepseek.com" {
		t.Fatalf("configured: provider=%q key=%q base=%q", provider, apiKey, baseURL)
	}

	// A provider without per-provider config resolves to an empty key and
	// the default endpoint (empty base URL).
	provider, err = modelProvider(st, "claude-opus-4-8")
	if err != nil {
		t.Fatal(err)
	}
	apiKey, baseURL, err = stg.ProviderConfig(provider)
	if err != nil {
		t.Fatal(err)
	}
	if provider != "Anthropic" || apiKey != "" || baseURL != "" {
		t.Fatalf("unconfigured: provider=%q key=%q base=%q", provider, apiKey, baseURL)
	}
}

// TestDefaultModel picks the first seeded claude model for a fresh install
// whose settings model key is unset.
func TestDefaultModel(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := ensureModels(st); err != nil {
		t.Fatalf("ensureModels: %v", err)
	}
	if got := defaultModel(st); got != "claude-opus-4-8" {
		t.Fatalf("defaultModel = %q, want claude-opus-4-8 (first seeded claude model)", got)
	}

	// A user who removed every claude model gets no default; agent.New then
	// surfaces its own required-model error rather than booting on a guess.
	models, err := st.ListModels()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range models {
		if strings.HasPrefix(m.Value, "claude-") {
			if err := st.DeleteModel(m.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := defaultModel(st); got != "" {
		t.Fatalf("defaultModel after deleting claude models = %q, want empty", got)
	}
}

// TestDefaultModelStoreErrorFallsBackToEmpty covers the defensive branch: a
// closed store must not panic, just report no default.
func TestDefaultModelStoreErrorFallsBackToEmpty(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if got := defaultModel(st); got != "" {
		t.Fatalf("defaultModel on closed store = %q, want empty", got)
	}
}
