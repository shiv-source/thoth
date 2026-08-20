package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-warehouse/events"
	"github.com/shiv-source/thoth/internal/assets"
	"github.com/shiv-source/thoth/internal/claude"
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

// TestWatchWikiFlushesPoolOnRulebookEdit covers the serve-side wiring: the
// watcher's rulebook hook is bound to the pooled-CLI flush, so editing the
// wiki-root CLAUDE.md drops every idle pooled process (the next turn of each
// conversation respawns under the new rules). A plain note edit must not
// flush.
func TestWatchWikiFlushesPoolOnRulebookEdit(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()
	ix, err := index.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	bus := events.New(events.WithLogger(log))
	t.Cleanup(func() { bus.Close(); _ = bus.Wait(context.Background()) })

	var mu sync.Mutex
	flushed := 0
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watchWiki(ctx, root, ix, log, bus, func() {
			mu.Lock()
			flushed++
			mu.Unlock()
		})
	}()
	t.Cleanup(func() { cancel(); <-done })

	// A plain note edit must not flush the pool.
	note := filepath.Join(root, "daily", "note.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("---\ntitle: Watched\ntype: daily\n---\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond) // let the watcher debounce and flush
	mu.Lock()
	if flushed != 0 {
		mu.Unlock()
		t.Fatalf("note edit flushed the pool %d times, want 0", flushed)
	}
	mu.Unlock()

	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("new rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForCond(t, 5*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return flushed >= 1
	})
	mu.Lock()
	defer mu.Unlock()
	if flushed != 1 {
		t.Fatalf("one rulebook edit flushed the pool %d times, want 1", flushed)
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
	if w.Root != root {
		t.Fatalf("wiki root = %q, want %q", w.Root, root)
	}
	if _, err := os.Stat(filepath.Join(root, "attachments")); err != nil {
		t.Fatalf("ensureWiki must create the reserved attachments dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "attachments", ".gitkeep")); err != nil {
		t.Fatalf("reserved attachments dir must carry a .gitkeep: %v", err)
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

// writePrewarmFake writes a fake claude that logs each invocation's argv to
// argv.txt and, per stream-json user control line on stdin, replies with one
// turn — the contract a pooled process must satisfy for a boot pre-warm: a
// warm process answers a later turn without a respawn.
func writePrewarmFake(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	script := `#!/bin/sh
echo "$@" >> "$(dirname "$0")/argv.txt"
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// cliSpawnCount returns how many pooled processes the fake spawned (0 when
// argv.txt is not there yet — prewarm spawns asynchronously).
func cliSpawnCount(t *testing.T, bin string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		return 0
	}
	return strings.Count(string(raw), "--input-format")
}

func waitForCond(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

func TestPrewarmPoolExistingConversation(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, _ := openTestRepos(t)
	bin := writePrewarmFake(t)
	pc := claude.NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	id, err := st.CreateConversation("warm me")
	if err != nil {
		t.Fatal(err)
	}
	prewarmPool(log, st, pc)

	// The warm spawn must be the persistent-mode invocation for the
	// conversation's session, with no prompt (it rides stdin later).
	waitForCond(t, 5*time.Second, func() bool { return cliSpawnCount(t, bin) == 1 })
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--session-id", id} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("prewarm argv %q missing %q", raw, want)
		}
	}

	// The warmed process is alive and pooled: a turn on the same session
	// produces zero new spawns.
	var got []claude.Event
	if err := pc.Start(context.Background(), id, "hello", claude.WriterFunc(func(e claude.Event) error {
		got = append(got, e)
		return nil
	})); err != nil {
		t.Fatalf("Start on warmed session: %v", err)
	}
	if len(got) == 0 || got[0].Text != "turn-1" {
		t.Fatalf("warmed-session deltas: %+v", got)
	}
	if n := cliSpawnCount(t, bin); n != 1 {
		t.Fatalf("turn after prewarm respawned: %d spawns, want 1", n)
	}
}

func TestPrewarmPoolUsesRotatedSessionID(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, _ := openTestRepos(t)
	bin := writePrewarmFake(t)
	pc := claude.NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	id, err := st.CreateConversation("warm me")
	if err != nil {
		t.Fatal(err)
	}
	// A forked session is the one a turn uses, so it — not the conversation
	// id — must be warmed.
	if err := st.SetClaudeSessionID(id, "rotated-session-uuid"); err != nil {
		t.Fatal(err)
	}
	prewarmPool(log, st, pc)
	waitForCond(t, 5*time.Second, func() bool { return cliSpawnCount(t, bin) == 1 })
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "--session-id rotated-session-uuid") {
		t.Fatalf("prewarm argv %q must carry the rotated session id", raw)
	}
}

func TestPrewarmPoolEmptyDBLeavesPoolEmpty(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, _ := openTestRepos(t)
	bin := writePrewarmFake(t)
	pc := claude.NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	prewarmPool(log, st, pc)
	if n := cliSpawnCount(t, bin); n != 0 {
		t.Fatalf("empty database must not prewarm, got %d spawns", n)
	}
}

func TestPrewarmPoolSpawnFailureNonFatal(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, _ := openTestRepos(t)
	if _, err := st.CreateConversation("warm me"); err != nil {
		t.Fatal(err)
	}
	pc := claude.NewPersistent(filepath.Join(t.TempDir(), "missing-claude"), t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	prewarmPool(log, st, pc) // must not panic; boot continues
	if n := cliSpawnCount(t, filepath.Dir(pc.Bin)); n != 0 {
		t.Fatalf("failed prewarm spawned: %d", n)
	}
}

func TestPrewarmPoolStoreErrorNonFatal(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bin := writePrewarmFake(t)
	pc := claude.NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })
	st, _ := openTestRepos(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	prewarmPool(log, st, pc) // closed store: warn and continue
	if n := cliSpawnCount(t, bin); n != 0 {
		t.Fatalf("store error must not prewarm, got %d spawns", n)
	}
}

// fakePrewarmStore drives prewarmPool's defensive branches that a real store
// cannot isolate (a session-id lookup only fails after the recent-conversation
// lookup succeeded, which a closed db cannot express).
type fakePrewarmStore struct {
	recent    store.Conversation
	recentErr error
	sid       string
	sidErr    error
}

func (f *fakePrewarmStore) RecentConversation() (store.Conversation, error) {
	return f.recent, f.recentErr
}

func (f *fakePrewarmStore) ClaudeSessionID(string) (string, error) {
	return f.sid, f.sidErr
}

func TestPrewarmPoolSessionIDErrorNonFatal(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bin := writePrewarmFake(t)
	pc := claude.NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })
	st := &fakePrewarmStore{recent: store.Conversation{ID: "conv-1"}, sidErr: errors.New("boom")}

	prewarmPool(log, st, pc) // session-id lookup error: warn and continue
	if n := cliSpawnCount(t, bin); n != 0 {
		t.Fatalf("session-id error must not prewarm, got %d spawns", n)
	}
}
