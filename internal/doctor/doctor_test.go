package doctor

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

func testLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// freePort returns a port that was free at the moment of the call.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port
}

// fakeClaude writes an executable "claude" script to a fresh dir, puts that
// dir on PATH, and returns the path. The script matches "$1" against the given
// version and auth exit codes.
func fakeClaude(t *testing.T, versionExit, authExit int) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "claude")
	script := "#!/bin/sh\ncase \"$1\" in\n  --version) exit " + strconv.Itoa(versionExit) + " ;;\n  auth) exit " + strconv.Itoa(authExit) + " ;;\n  *) exit 1 ;;\nesac\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin))
	return bin
}

// healthyThothDir builds a fully healthy installation in a temp dir: a wiki
// with one note, a synced index, a config with a free port, and a fake claude
// on PATH. Returns the thoth dir.
func healthyThothDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fakeClaude(t, 0, 0)

	wikiRoot := filepath.Join(dir, "wiki")
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiRoot, "inbox", "hello.md"),
		[]byte("---\ntitle: Hello\n---\n\nHello body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "thoth.db")
	ix, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Rebuild(wikiRoot, testLog()); err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.WikiPath = wikiRoot
	cfg.Port = freePort(t)
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	return dir
}

func byName(t *testing.T, checks []Check, name string) Check {
	t.Helper()
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %q check among %+v", name, checks)
	return Check{}
}

func TestRunHealthy(t *testing.T) {
	checks := Run(context.Background(), healthyThothDir(t), testLog())
	want := []string{"config", "wiki", "claude", "claude login", "database", "index", "api", "websocket"}
	if len(checks) != len(want) {
		t.Fatalf("got %d checks, want %d: %+v", len(checks), len(want), checks)
	}
	for i, name := range want {
		if checks[i].Name != name {
			t.Fatalf("check %d: got %q, want %q", i, checks[i].Name, name)
		}
		if !checks[i].OK {
			t.Fatalf("check %q failed in a healthy env: %s", name, checks[i].Message)
		}
	}
}

// TestRunEmptyDir exercises the default-resolution path (dir == "") against a
// HOME that has nothing in it: every probe that can fail must fail, and the
// claude check must not attempt a login probe when the binary is missing.
func TestRunEmptyDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	checks := Run(context.Background(), "", testLog())
	for _, name := range []string{"config", "wiki", "claude", "database", "index"} {
		if byName(t, checks, name).OK {
			t.Fatalf("check %q must fail in an empty installation", name)
		}
	}
	for _, c := range checks {
		if c.Name == "claude login" {
			t.Fatalf("claude login must be skipped when the binary is missing: %+v", checks)
		}
	}
}

func TestRunMissingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	checks := Run(context.Background(), dir, testLog())
	c := byName(t, checks, "config")
	if c.OK || !strings.Contains(c.Message, "is missing") {
		t.Fatalf("config: %s", c.Message)
	}
	// With the config missing, later checks run against the defaults, so the
	// default wiki path must be reported as missing too.
	if byName(t, checks, "wiki").OK {
		t.Fatalf("wiki must fail when the default wiki path is missing")
	}
}

func TestRunUnparsableConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[[[ not toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "config")
	if c.OK || !strings.Contains(c.Message, "cannot be parsed") {
		t.Fatalf("config: %s", c.Message)
	}
}

func TestRunWikiMissingFolders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(filepath.Join(wikiRoot, "inbox"), 0o755); err != nil { // partial scaffold
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.WikiPath = wikiRoot
	cfg.Port = freePort(t)
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "wiki")
	if c.OK || !strings.Contains(c.Message, "is missing") {
		t.Fatalf("wiki: %s", c.Message)
	}
}

func TestRunClaudeVersionFailure(t *testing.T) {
	dir := healthyThothDir(t)
	// Override the PATH claude with one that fails --version.
	fakeClaude(t, 1, 1)
	checks := Run(context.Background(), dir, testLog())
	c := byName(t, checks, "claude")
	if c.OK || !strings.Contains(c.Message, "--version failed") {
		t.Fatalf("claude: %s", c.Message)
	}
	for _, chk := range checks {
		if chk.Name == "claude login" {
			t.Fatalf("claude login must be skipped when the binary is broken: %+v", checks)
		}
	}
}

func TestRunClaudeLoginUnknown(t *testing.T) {
	dir := healthyThothDir(t)
	fakeClaude(t, 0, 1)
	checks := Run(context.Background(), dir, testLog())
	if !byName(t, checks, "claude").OK {
		t.Fatalf("claude check must pass when --version works: %+v", checks)
	}
	login := byName(t, checks, "claude login")
	if login.OK || !strings.Contains(login.Message, "login status unknown") {
		t.Fatalf("claude login: %s", login.Message)
	}
}

func TestRunConfiguredClaudeBinMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	cfg := config.Default()
	cfg.ClaudeBin = filepath.Join(t.TempDir(), "nope")
	cfg.Port = freePort(t)
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "claude")
	if c.OK || !strings.Contains(c.Message, "configured claude_bin") {
		t.Fatalf("claude: %s", c.Message)
	}
}

func TestRunNonThothProcessOnPort(t *testing.T) {
	dir := healthyThothDir(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	port := ln.Addr().(*net.TCPAddr).Port

	cfg := configLoad(t, filepath.Join(dir, "config.toml"))
	cfg.Port = port
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "api")
	if c.OK || !strings.Contains(c.Message, "non-thoth") {
		t.Fatalf("api: %s", c.Message)
	}
	// The websocket check is skipped: the port is not a Thoth server.
	ws := byName(t, Run(context.Background(), dir, testLog()), "websocket")
	if !ws.OK || !strings.Contains(ws.Message, "skipped") {
		t.Fatalf("websocket: %s", ws.Message)
	}
}

// serveThothAPI serves the REST health and chat websocket endpoints on ln the
// way the real server does, so the api check can probe a running Thoth.
func serveThothAPI(t *testing.T, ln net.Listener, health string) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			_, _ = w.Write([]byte(health))
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

func TestRunAPIHealthy(t *testing.T) {
	dir := healthyThothDir(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	cfg := configLoad(t, filepath.Join(dir, "config.toml"))
	cfg.Port = ln.Addr().(*net.TCPAddr).Port
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	serveThothAPI(t, ln, `{"status":"ok","claude":{"found":true,"path":"/fake/claude"},"wiki":{"path":"/fake/wiki","exists":true}}`)

	checks := Run(context.Background(), dir, testLog())
	c := byName(t, checks, "api")
	if !c.OK || !strings.Contains(c.Message, "REST") {
		t.Fatalf("api: %s", c.Message)
	}
	ws := byName(t, checks, "websocket")
	if !ws.OK || !strings.Contains(ws.Message, "connects") {
		t.Fatalf("websocket: %s", ws.Message)
	}
}

func TestRunAPIPartialFailures(t *testing.T) {
	for name, health := range map[string]string{
		"claude missing": `{"status":"ok","claude":{"found":false,"path":""},"wiki":{"exists":true}}`,
		"wiki missing":   `{"status":"ok","claude":{"found":true,"path":"/f"},"wiki":{"exists":false}}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := healthyThothDir(t)
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = ln.Close() }()
			cfg := configLoad(t, filepath.Join(dir, "config.toml"))
			cfg.Port = ln.Addr().(*net.TCPAddr).Port
			if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
				t.Fatal(err)
			}
			serveThothAPI(t, ln, health)

			c := byName(t, Run(context.Background(), dir, testLog()), "api")
			if c.OK || !strings.Contains(c.Message, strings.Split(name, " ")[0]) {
				t.Fatalf("api: %s", c.Message)
			}
		})
	}
}

func TestRunAPIWebsocketFails(t *testing.T) {
	dir := healthyThothDir(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	cfg := configLoad(t, filepath.Join(dir, "config.toml"))
	cfg.Port = ln.Addr().(*net.TCPAddr).Port
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	// REST health works but the /ws upgrade does not.
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			_, _ = w.Write([]byte(`{"status":"ok","claude":{"found":true,"path":"/f"},"wiki":{"exists":true}}`))
			return
		}
		http.NotFound(w, r)
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	checks := Run(context.Background(), dir, testLog())
	if c := byName(t, checks, "api"); !c.OK {
		t.Fatalf("api: %s", c.Message)
	}
	if c := byName(t, checks, "websocket"); c.OK || !strings.Contains(c.Message, "did not connect") {
		t.Fatalf("websocket: %s", c.Message)
	}
}

func TestRunNonWALDatabase(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=DELETE`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	cfg := config.Default()
	cfg.WikiPath = filepath.Join(dir, "wiki")
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "database")
	if c.OK || !strings.Contains(c.Message, `journal mode is "delete"`) {
		t.Fatalf("database: %s", c.Message)
	}
}

func TestRunDatabaseMissingTables(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	cfg := config.Default()
	cfg.WikiPath = filepath.Join(dir, "wiki")
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "database")
	if c.OK || !strings.Contains(c.Message, "missing the notes/notes_fts tables") {
		t.Fatalf("database: %s", c.Message)
	}
}

func TestRunCorruptDatabase(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "thoth.db"), []byte("definitely not a sqlite file"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.WikiPath = filepath.Join(dir, "wiki")
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "database")
	if c.OK || !strings.Contains(c.Message, "not a usable sqlite database") {
		t.Fatalf("database: %s", c.Message)
	}
}

func TestRunMissingDatabaseIndex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.WikiPath = wikiRoot
	cfg.Port = freePort(t)
	if err := config.Save(filepath.Join(dir, "config.toml"), cfg); err != nil {
		t.Fatal(err)
	}
	// Wiki exists, but thoth.db was never created: both the database and the
	// index checks must fail, each with its own message.
	checks := Run(context.Background(), dir, testLog())
	if db := byName(t, checks, "database"); db.OK || !strings.Contains(db.Message, "does not exist") {
		t.Fatalf("database: %s", db.Message)
	}
	if ix := byName(t, checks, "index"); ix.OK || !strings.Contains(ix.Message, "does not exist") {
		t.Fatalf("index: %s", ix.Message)
	}
}

func TestRunIndexOutOfSync(t *testing.T) {
	dir := healthyThothDir(t)
	// Add a note on disk after the index was built: the index is now stale.
	if err := os.WriteFile(filepath.Join(dir, "wiki", "inbox", "later.md"),
		[]byte("---\ntitle: Later\n---\n\nAdded after indexing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := byName(t, Run(context.Background(), dir, testLog()), "index")
	if c.OK || !strings.Contains(c.Message, "notes on disk") {
		t.Fatalf("index: %s", c.Message)
	}
}

func TestRunWikiMissingNoteIsSkipped(t *testing.T) {
	// Notes without frontmatter are not indexable; they must not fail the
	// index sync check.
	dir := healthyThothDir(t)
	if err := os.WriteFile(filepath.Join(dir, "wiki", "inbox", "plain.md"),
		[]byte("just some text, no frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !byName(t, Run(context.Background(), dir, testLog()), "index").OK {
		t.Fatalf("index must ignore non-indexable notes")
	}
}

// config.LoadOrPanicForTest exists to keep the helper list local; it loads the
// config or fails the test.
func configLoad(t *testing.T, path string) config.Config {
	t.Helper()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
