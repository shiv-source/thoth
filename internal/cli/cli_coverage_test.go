package cli

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

func TestVersion(t *testing.T) {
	if Version() != version {
		t.Fatalf("Version() = %q, want %q", Version(), version)
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(os.Stderr) {
		t.Fatal("isTerminal must be false for a non-TTY stderr in tests")
	}
	f, err := os.CreateTemp(t.TempDir(), "term")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if isTerminal(f) {
		t.Fatal("isTerminal must be false for a regular file")
	}
}

func TestOpenIndex(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbPath := filepath.Join(t.TempDir(), "idx.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	wikiRoot := t.TempDir()
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	ix, err := openIndex(dbPath, wikiRoot, log)
	if err != nil {
		t.Fatalf("openIndex: %v", err)
	}
	defer func() { _ = ix.Close() }()

	// A db path whose parent does not exist cannot be opened.
	if _, err := openIndex(filepath.Join(t.TempDir(), "missing", "x.db"), wikiRoot, log); err == nil {
		t.Fatal("expected an error for an unopenable db path")
	}
}

func TestServeUntilShutdownStartError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	if err := serveUntilShutdown(e, ln.Addr().String(), context.Background()); err == nil {
		t.Fatal("expected an error when the address is already bound")
	}
}

func TestServeUntilShutdownOnCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	if err := serveUntilShutdown(e, "127.0.0.1:"+strconv.Itoa(port), ctx); err != nil {
		t.Fatalf("serveUntilShutdown on cancel = %v, want nil", err)
	}
}

func TestRunServeBootsAndShutsDown(t *testing.T) {
	// Refuse to run over a live server: the boot test binds the fixed prod
	// port and must be deterministic.
	ln, err := net.Listen("tcp", "127.0.0.1:8333")
	if err != nil {
		t.Skip("port 8333 is occupied — cannot run the serve boot test")
	}
	_ = ln.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cmd := &cobra.Command{}
	ctx, cancel := context.WithCancel(context.Background())
	cmd.SetContext(ctx)
	done := make(chan error, 1)
	go func() { done <- runServe(cmd, false) }()

	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := http.Get("http://127.0.0.1:8333/api/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server did not become healthy: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runServe returned %v after shutdown, want nil", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("runServe did not return after the context was cancelled")
	}

	// The boot seeded the wiki under the isolated home.
	if _, err := os.Stat(filepath.Join(home, ".thoth", "wiki", "CLAUDE.md")); err != nil {
		t.Fatalf("boot did not scaffold the wiki: %v", err)
	}
}

func TestRunServeSwitchesWikiPathLive(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:8333")
	if err != nil {
		t.Skip("port 8333 is occupied — cannot run the serve boot test")
	}
	_ = ln.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cmd := &cobra.Command{}
	ctx, cancel := context.WithCancel(context.Background())
	cmd.SetContext(ctx)
	done := make(chan error, 1)
	go func() { done <- runServe(cmd, false) }()

	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := http.Get("http://127.0.0.1:8333/api/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server did not become healthy: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// A live wiki-path change swaps the root and restarts the watcher,
	// exercising the second startWatcher call (which cancels the first).
	newRoot := filepath.Join(home, ".thoth", "wiki2")
	body := `{"wiki_path":"` + newRoot + `","repo_url":"","sync_enabled":false}`
	req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:8333/api/settings", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("PUT settings: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("PUT settings status %d, want 200", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runServe returned %v after shutdown, want nil", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("runServe did not return after the context was cancelled")
	}
	if _, err := os.Stat(filepath.Join(newRoot, "CLAUDE.md")); err != nil {
		t.Fatalf("live wiki-path change did not scaffold the new root: %v", err)
	}
}

func TestRunServeFailsWhenNoClaudeModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Pre-create the db with only a non-claude model so ensureModels does not
	// seed the claude family, defaultModel comes back empty, and agent.New
	// fails the boot.
	dbPath := filepath.Join(home, ".thoth", "thoth.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateModel("custom-model", "Custom", "", "Vendor"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runServe(cmd, false); err == nil {
		t.Fatal("expected runServe to fail when no claude model is available")
	}
}

func TestRunServeDevBootsAndShutsDown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:8334")
	if err != nil {
		t.Skip("port 8334 is occupied — cannot run the dev serve boot test")
	}
	_ = ln.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cmd := &cobra.Command{}
	ctx, cancel := context.WithCancel(context.Background())
	cmd.SetContext(ctx)
	done := make(chan error, 1)
	go func() { done <- runServe(cmd, true) }()

	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := http.Get("http://127.0.0.1:8334/api/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("dev server did not become healthy: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runServe dev returned %v after shutdown, want nil", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("runServe dev did not return after the context was cancelled")
	}
	// Dev isolates its data under ~/.thoth/dev.
	if _, err := os.Stat(filepath.Join(home, ".thoth", "dev", "wiki", "CLAUDE.md")); err != nil {
		t.Fatalf("dev boot did not scaffold the isolated wiki: %v", err)
	}
}

func TestRunServeFailsWhenHomeIsFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", blocker)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runServe(cmd, false); err == nil {
		t.Fatal("expected runServe to fail when the thoth dir cannot be created")
	}
}

func TestRunServeFailsWhenStoreUnopenable(t *testing.T) {
	home := t.TempDir()
	// A directory squatting on thoth.db makes store.Open fail after the
	// thoth dir itself was created.
	if err := os.MkdirAll(filepath.Join(home, ".thoth", "thoth.db"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runServe(cmd, false); err == nil {
		t.Fatal("expected runServe to fail when thoth.db is unopenable")
	}
}

func TestRunServeFailsWhenWikiBlocked(t *testing.T) {
	home := t.TempDir()
	// A file squatting on the default wiki path makes the scaffold fail.
	if err := os.MkdirAll(filepath.Join(home, ".thoth"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".thoth", "wiki"), []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runServe(cmd, false); err == nil {
		t.Fatal("expected runServe to fail when the wiki path is blocked by a file")
	}
}

func TestResolveThothDirEmptyUsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := resolveThothDir("")
	if err != nil {
		t.Fatalf("resolveThothDir: %v", err)
	}
	if want := filepath.Join(home, ".thoth"); got != want {
		t.Fatalf("resolveThothDir(\"\") = %q, want %q", got, want)
	}
	if got, err := resolveThothDir("/custom/dir"); err != nil || got != "/custom/dir" {
		t.Fatalf("resolveThothDir(explicit) = %q/%v", got, err)
	}
}

func TestOnSettingsSavedSwitchesRootTwice(t *testing.T) {
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

	rootA := filepath.Join(t.TempDir(), "a")
	rootB := filepath.Join(t.TempDir(), "b")
	rootC := filepath.Join(t.TempDir(), "c")
	for _, r := range []string{rootA, rootB, rootC} {
		if err := wiki.Scaffold(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := ix.Sync(rootA, log); err != nil {
		t.Fatal(err)
	}

	w := wiki.New(rootA)
	var watches int
	startWatcher := func(r string) { watches++ }
	cb := onSettingsSaved(log, stg, w, ix, startWatcher)
	if err := cb(rootB); err != nil {
		t.Fatalf("first swap: %v", err)
	}
	// The second swap must cancel the first watcher (its cancel path) and
	// restart the watcher under the new root.
	if err := cb(rootC); err != nil {
		t.Fatalf("second swap: %v", err)
	}
	if w.Root() != rootC {
		t.Fatalf("root = %q, want %q", w.Root(), rootC)
	}
	if watches != 2 {
		t.Fatalf("watcher restarted %d times, want 2", watches)
	}
}

func TestOnSettingsSavedSyncError(t *testing.T) {
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

	root := filepath.Join(t.TempDir(), "wiki")
	if err := wiki.Scaffold(root); err != nil {
		t.Fatal(err)
	}
	w := wiki.New(root)
	cb := onSettingsSaved(log, stg, w, ix, func(string) {})

	// A closed index makes the rebuild fail after the new root scaffolded;
	// the root must stay untouched.
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}
	newRoot := filepath.Join(t.TempDir(), "new")
	if err := cb(newRoot); err == nil {
		t.Fatal("expected an error when the index sync fails")
	}
	if w.Root() != root {
		t.Fatalf("root mutated after a failed sync: %q", w.Root())
	}
}

func TestThothDirErrorWhenHomeIsFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", blocker)
	if _, err := thothDir(false); err == nil {
		t.Fatal("expected an error when HOME is a file")
	}
}

func TestModelProviderStoreError(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := modelProvider(st, "claude-sonnet-5"); err == nil {
		t.Fatal("expected an error for a closed store")
	}
}

func TestSettleWikiPathError(t *testing.T) {
	st, stg := openTestRepos(t)
	_ = st
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := settleWikiPath(stg, false, t.TempDir()); err == nil {
		t.Fatal("expected an error for a closed settings repo")
	}
}

func TestEnsureModelsStoreError(t *testing.T) {
	st, _ := openTestRepos(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ensureModels(st); err == nil {
		t.Fatal("expected an error for a closed store")
	}
}

func TestScaffoldFoldersError(t *testing.T) {
	st, stg := openTestRepos(t)
	_ = st
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}
	if got := scaffoldFolders(stg, slog.New(slog.NewTextHandler(io.Discard, nil))); got != nil {
		t.Fatalf("scaffoldFolders = %v, want nil on error", got)
	}
}

func TestDoctorRepairScaffoldFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	// A file blocks the configured wiki path so the repair's scaffold fails.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "thoth.db")
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
	if err := stg.SetSetting(settings.KeyWikiPath, filepath.Join(blocker, "wiki")); err != nil {
		t.Fatal(err)
	}
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := executeDoctor(t, dir, true)
	if err == nil {
		t.Fatal("expected an unhealthy doctor result")
	}
	if !strings.Contains(out, "wiki: could not scaffold") {
		t.Fatalf("expected a scaffold-failure fix message, got:\n%s", out)
	}
}

func TestDoctorRepairIndexSyncFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// The thoth dir sits under a regular file, so the index repair's open of
	// <dir>/thoth.db must fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(blocker, "sub")
	out, err := executeDoctor(t, dir, true)
	if err == nil {
		t.Fatal("expected an unhealthy doctor result")
	}
	if !strings.Contains(out, "index: sync failed") {
		t.Fatalf("expected an index sync-failure fix message, got:\n%s", out)
	}
}
