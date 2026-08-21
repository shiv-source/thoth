package cli

import (
	"database/sql"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

// healthyEnv builds a fully healthy installation in a temp thoth dir: a wiki
// with one valid note, a synced index, the selected model seeded into the
// registry, and a configured api key. Returns the thoth dir.
func healthyEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	wikiRoot := filepath.Join(dir, "wiki")
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiRoot, "inbox", "hello.md"),
		[]byte("---\ntitle: Hello\n---\n\nHello body\n"), 0o644); err != nil {
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
	ix, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Sync(wikiRoot, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateModel("claude-sonnet-5", "Claude Sonnet 5", "balanced", "Anthropic"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyWikiPath, wikiRoot); err != nil {
		t.Fatal(err)
	}
	// The api key + model setup checks pass in a healthy env.
	if err := stg.SetSetting(settings.KeyAPIKey, "sk-healthy"); err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyModel, "claude-sonnet-5"); err != nil {
		t.Fatal(err)
	}
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func executeDoctor(t *testing.T, dir string, fix bool) (string, error) {
	t.Helper()
	// Point the provider check at a stub answering 200, so the CLI never
	// probes the live network in tests.
	pv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer pv.Close()
	root := newRootCmd()
	args := []string{"doctor"}
	if fix {
		args = append(args, "--fix")
	}
	args = append(args, "--dir", dir)
	args = append(args, "--provider-base-url", pv.URL)
	var err error
	out := captureStdout(t, func() {
		root.SetArgs(args)
		err = root.Execute()
	})
	return out, err
}

// serveThothOnFixedPort binds the fixed 127.0.0.1:8333 address and serves a
// minimal Thoth (health + websocket upgrade) so healthy doctor runs are
// deterministic; skips when a real server occupies the port.
func serveThothOnFixedPort(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:8333")
	if err != nil {
		t.Skip("port 8333 is occupied — cannot run the healthy doctor deterministically")
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			_, _ = w.Write([]byte(`{"status":"ok","wiki":{"path":"/fake/wiki","exists":true}}`))
		case "/ws":
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			_ = conn.Close()
		default:
			http.NotFound(w, r)
		}
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
}

func TestDoctorHealthy(t *testing.T) {
	dir := healthyEnv(t)
	// The api/websocket checks probe the fixed port; serve a stub Thoth on it
	// so the healthy run is deterministic (skips if 8333 is busy).
	serveThothOnFixedPort(t)
	out, err := executeDoctor(t, dir, false)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}
	if strings.Contains(out, "✗") {
		t.Fatalf("unexpected failing checks:\n%s", out)
	}
	for _, want := range []string{"wiki:", "provider:", "api key:", "model:", "database:", "index:", "api:", "websocket:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "in sync: 1 notes") {
		t.Fatalf("index sync count not reported:\n%s", out)
	}
}

func TestDoctorMissingWikiAndFix(t *testing.T) {
	// The post-fix run asserts a fully green suite incl. the fixed-port api
	// check — serve a stub Thoth there (skips if 8333 is busy).
	serveThothOnFixedPort(t)
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	st, err := store.Open(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateModel("claude-sonnet-5", "Claude Sonnet 5", "balanced", "Anthropic"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	stg, err := settings.OpenRepo(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyWikiPath, wikiRoot); err != nil {
		t.Fatal(err)
	}
	// The api key + model setup checks pass in a healthy env; --fix repairs
	// wiki/index only, setup is not its job.
	if err := stg.SetSetting(settings.KeyAPIKey, "sk-healthy"); err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyModel, "claude-sonnet-5"); err != nil {
		t.Fatal(err)
	}
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}

	// Without --fix the wiki check fails and nothing is mutated.
	out, err := executeDoctor(t, dir, false)
	if err == nil {
		t.Fatalf("expected doctor to fail on a missing wiki:\n%s", out)
	}
	if !strings.Contains(out, "✗ wiki") {
		t.Fatalf("expected failing wiki check:\n%s", out)
	}
	if _, err := os.Stat(wikiRoot); !os.IsNotExist(err) {
		t.Fatalf("doctor without --fix must not scaffold the wiki")
	}

	// With --fix the wiki is scaffolded and every check passes.
	out, err = executeDoctor(t, dir, true)
	if err != nil {
		t.Fatalf("Execute --fix: %v\n%s", err, out)
	}
	if strings.Contains(out, "✗") {
		t.Fatalf("unexpected failing checks after --fix:\n%s", out)
	}
	if !strings.Contains(out, "fixed:") {
		t.Fatalf("expected fix lines:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(wikiRoot, "CLAUDE.md")); err != nil {
		t.Fatalf("--fix did not scaffold the wiki: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "thoth.db")); err != nil {
		t.Fatalf("--fix did not create the database: %v", err)
	}
}

func TestDoctorFixesOutOfSyncIndex(t *testing.T) {
	// The post-fix run asserts a fully green suite incl. the fixed-port api
	// check — serve a stub Thoth there (skips if 8333 is busy).
	serveThothOnFixedPort(t)
	dir := healthyEnv(t)
	// Add a second note the index does not know about.
	if err := os.WriteFile(filepath.Join(dir, "wiki", "inbox", "later.md"),
		[]byte("---\ntitle: Later\n---\n\nLater body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeDoctor(t, dir, false)
	if err == nil {
		t.Fatalf("expected doctor to fail on a stale index:\n%s", out)
	}
	if !strings.Contains(out, "✗ index") {
		t.Fatalf("expected failing index check:\n%s", out)
	}

	out, err = executeDoctor(t, dir, true)
	if err != nil {
		t.Fatalf("Execute --fix: %v\n%s", err, out)
	}
	if strings.Contains(out, "✗") {
		t.Fatalf("unexpected failing checks after --fix:\n%s", out)
	}
	if !strings.Contains(out, "in sync: 2 notes") {
		t.Fatalf("index not rebuilt:\n%s", out)
	}
}

func TestDoctorReportsProviderAuthFailure(t *testing.T) {
	// A provider endpoint that rejects the API key surfaces as a failing
	// provider check with the status called out.
	dir := healthyEnv(t)
	root := newRootCmd()
	pv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer pv.Close()
	var err error
	out := captureStdout(t, func() {
		root.SetArgs([]string{"doctor", "--dir", dir, "--provider-base-url", pv.URL})
		err = root.Execute()
	})
	if err == nil {
		t.Fatalf("expected doctor to fail on a provider auth failure:\n%s", out)
	}
	if !strings.Contains(out, "✗ provider") || !strings.Contains(out, "401") {
		t.Fatalf("expected failing provider check:\n%s", out)
	}
}

func TestDoctorDetectsBusyPort(t *testing.T) {
	dir := healthyEnv(t)
	// The api check probes the fixed 127.0.0.1:8333 address; skip when a real
	// server occupies it.
	ln, err := net.Listen("tcp", "127.0.0.1:8333")
	if err != nil {
		t.Skip("port 8333 is occupied by a real server — cannot run the busy-port check deterministically")
	}
	defer func() { _ = ln.Close() }()

	out, err := executeDoctor(t, dir, false)
	if err == nil {
		t.Fatalf("expected doctor to fail on a busy port:\n%s", out)
	}
	if !strings.Contains(out, "✗ api") || !strings.Contains(out, "non-thoth") {
		t.Fatalf("expected failing api check:\n%s", out)
	}
}

func TestDoctorFixesMissingDefaultWiki(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	out, err := executeDoctor(t, dir, true)
	// The default port (8333) may or may not be free on this machine, so the
	// exit status is not asserted — only that the default wiki was scaffolded
	// under the temp HOME.
	if err != nil {
		t.Logf("doctor --fix reported problems (expected if port 8333 is busy):\n%s", out)
	}
	if !strings.Contains(out, "scaffolded") {
		t.Fatalf("expected scaffold fix line:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".thoth", "wiki", "CLAUDE.md")); err != nil {
		t.Fatalf("--fix did not scaffold the default wiki under HOME: %v", err)
	}
}

func TestDoctorDetectsMissingIndexTables(t *testing.T) {
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	// store.Open runs the migrations (the single schema source). Simulate a
	// database that lost its index tables by dropping them afterwards.
	st, err := store.Open(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE notes_fts; DROP TABLE notes;`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	stg, err := settings.OpenRepo(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyWikiPath, wikiRoot); err != nil {
		t.Fatal(err)
	}
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := executeDoctor(t, dir, false)
	if err == nil {
		t.Fatalf("expected doctor to fail without index tables:\n%s", out)
	}
	if !strings.Contains(out, "✗ database") || !strings.Contains(out, "notes/notes_fts") {
		t.Fatalf("expected failing database check:\n%s", out)
	}
}

func TestDoctorDetectsNonWALDatabase(t *testing.T) {
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	// A valid sqlite database in the default (delete) journal mode (the
	// settings table is absent — the doctor falls back to the default wiki
	// path, which this test does not assert on).
	db, err := sql.Open("sqlite", filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=DELETE; CREATE TABLE t (x TEXT);`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := executeDoctor(t, dir, false)
	if err == nil {
		t.Fatalf("expected doctor to fail on a non-WAL database:\n%s", out)
	}
	if !strings.Contains(out, "✗ database") || !strings.Contains(out, "journal mode") {
		t.Fatalf("expected failing database check:\n%s", out)
	}
}
