package doctor

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

func testLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// freeAddr returns a 127.0.0.1:port address that was free at the moment of
// the call.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().String()
}

// seedSettingsRaw creates thoth.db in the given journal mode containing ONLY
// the settings table with the wiki_path row — simulating a partial or
// misconfigured database without running the store's migrations (which would
// repair the very breakage the test wants to probe).
func seedSettingsRaw(t *testing.T, dir, wikiPath, journalMode string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=` + journalMode); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO settings(key, value) VALUES ('wiki_path', ?)`, wikiPath); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

// providerStub serves the provider models endpoint with status for every
// request and returns the server, whose URL and Client the checks probe.
func providerStub(t *testing.T, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// runChecks runs the suite with a stub provider endpoint answering 200, so
// the provider check never touches the live network in tests.
func runChecks(t *testing.T, dir, addr string) []Check {
	t.Helper()
	srv := providerStub(t, http.StatusOK)
	return Run(context.Background(), Options{Dir: dir, Addr: addr, Log: testLog(), HTTP: srv.Client(), BaseURL: srv.URL})
}

// seedWikiPath stores a wiki path into the settings table of thoth.db under
// dir (creating the database via the store's migrations first).
func seedWikiPath(t *testing.T, dir, wikiPath string) {
	t.Helper()
	st, err := store.Open(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	r, err := settings.OpenRepo(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	if err := r.SetSetting(settings.KeyWikiPath, wikiPath); err != nil {
		t.Fatal(err)
	}
}

// healthyThothDir builds a fully healthy installation in a temp dir: a wiki
// with one note, a synced index, the settings table pointing at the wiki, the
// selected model seeded into the registry, and a configured api key. Returns
// the thoth dir.
func healthyThothDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	wikiRoot := filepath.Join(dir, "wiki")
	if err := wiki.Scaffold(wikiRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiRoot, "inbox", "hello.md"),
		[]byte("---\ntitle: Hello\ntype: inbox\n---\n\nHello body\n"), 0o644); err != nil {
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
	if err := ix.Sync(wikiRoot, testLog()); err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}
	seedWikiPath(t, dir, wikiRoot)
	// The setup checks pass in a healthy install: key + model configured,
	// and the model exists in the llm_models registry.
	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateModel("claude-sonnet-5", "Claude Sonnet 5", "balanced", "Anthropic"); err != nil {
		t.Fatal(err)
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.ProviderAPIKeyKey("Anthropic"), "sk-healthy"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.KeyModel, "claude-sonnet-5"); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	_ = st.Close()
	return dir
}

func TestRunSetupUnset(t *testing.T) {
	// A fresh install with no api key and no model: both checks fail with
	// guidance, and nothing else breaks.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	checks := runChecks(t, dir, "")
	if c := byName(t, checks, "api key"); c.OK || !strings.Contains(c.Message, "no API key") {
		t.Fatalf("api key: %s", c.Message)
	}
	if c := byName(t, checks, "model"); c.OK || !strings.Contains(c.Message, "no model") {
		t.Fatalf("model: %s", c.Message)
	}
}

func TestRunSetupConfigured(t *testing.T) {
	dir := healthyThothDir(t)
	checks := runChecks(t, dir, freeAddr(t))
	if c := byName(t, checks, "api key"); !c.OK || !strings.Contains(c.Message, "configured") {
		t.Fatalf("api key: %s", c.Message)
	}
	if c := byName(t, checks, "model"); !c.OK || !strings.Contains(c.Message, "claude-sonnet-5") {
		t.Fatalf("model: %s", c.Message)
	}
}

// runProviderChecks runs the suite with the provider stub answering status.
func runProviderChecks(t *testing.T, dir string, status int) []Check {
	t.Helper()
	srv := providerStub(t, status)
	return Run(context.Background(), Options{Dir: dir, Log: testLog(), HTTP: srv.Client(), BaseURL: srv.URL})
}

func TestRunProviderReachable(t *testing.T) {
	dir := healthyThothDir(t)
	c := byName(t, runProviderChecks(t, dir, http.StatusOK), "provider")
	if !c.OK || !strings.Contains(c.Message, "reachable") {
		t.Fatalf("provider: %s", c.Message)
	}
}

func TestRunProviderFailures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		want   string
	}{
		{"auth", http.StatusUnauthorized, "401"},
		{"rate limit", http.StatusTooManyRequests, "429"},
		{"server error", http.StatusInternalServerError, "server error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := healthyThothDir(t)
			c := byName(t, runProviderChecks(t, dir, tt.status), "provider")
			if c.OK || !strings.Contains(c.Message, tt.want) {
				t.Fatalf("provider: %s", c.Message)
			}
		})
	}
}

func TestRunProviderTimeout(t *testing.T) {
	dir := healthyThothDir(t)
	// A stub that hangs until the client gives up: the probe must time out.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	c := byName(t, Run(ctx, Options{Dir: dir, Log: testLog(), HTTP: srv.Client(), BaseURL: srv.URL}), "provider")
	if c.OK || !strings.Contains(c.Message, "timed out") {
		t.Fatalf("provider: %s", c.Message)
	}
}

func TestRunProviderNoModel(t *testing.T) {
	// Without a model the provider cannot be determined: the check fails with
	// guidance and never probes.
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	t.Cleanup(srv.Close)
	checks := Run(context.Background(), Options{Dir: dir, Log: testLog(), HTTP: srv.Client(), BaseURL: srv.URL})
	c := byName(t, checks, "provider")
	if c.OK || !strings.Contains(c.Message, "no model selected") {
		t.Fatalf("provider: %s", c.Message)
	}
	if hit {
		t.Fatal("provider probe must not be issued without a model")
	}
}

func TestRunProviderUnknownModelFamily(t *testing.T) {
	dir := healthyThothDir(t)
	// A model with no concrete provider maps to the same failure the agent
	// would report when asked to run it.
	r, err := settings.OpenRepo(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.KeyModel, "deepseek-v4-flash"); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	c := byName(t, runProviderChecks(t, dir, http.StatusOK), "provider")
	if c.OK || !strings.Contains(c.Message, "no provider for model") {
		t.Fatalf("provider: %s", c.Message)
	}
}

func TestRunProviderRoutesByRegistryProvider(t *testing.T) {
	// A DeepSeek model with its own key + base url resolves to the
	// OpenAI-compatible probe, and the per-provider key rides the probe — the
	// same model→provider→config chain serve resolves at boot.
	dir := healthyThothDir(t)
	dbPath := filepath.Join(dir, "thoth.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateModel("deepseek-v4-flash", "V4 Flash", "fastest", "DeepSeek"); err != nil {
		t.Fatal(err)
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.KeyModel, "deepseek-v4-flash"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.ProviderAPIKeyKey("DeepSeek"), "ds-secret"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.ProviderBaseURLKey("DeepSeek"), "https://api.deepseek.com"); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	_ = st.Close()

	var mu sync.Mutex
	var gotKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		gotKey = req.Header.Get("Authorization")
		gotPath = req.URL.Path
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := byName(t, Run(context.Background(), Options{Dir: dir, Log: testLog(), HTTP: srv.Client(), BaseURL: srv.URL}), "provider")
	if !c.OK || !strings.Contains(c.Message, "reachable") {
		t.Fatalf("provider: %s", c.Message)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotKey != "Bearer ds-secret" {
		t.Fatalf("probe Authorization = %q, want Bearer ds-secret", gotKey)
	}
	if gotPath != "/v1/models" {
		t.Fatalf("probe path = %q, want /v1/models", gotPath)
	}
}

func TestRunApiKeyCheckResolvesPerProviderKey(t *testing.T) {
	// Only the selected provider's own key is set — the check must pass with
	// the credential the agent will use (there is no shared fallback).
	dir := healthyThothDir(t)
	dbPath := filepath.Join(dir, "thoth.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateModel("deepseek-v4-flash", "V4 Flash", "fastest", "DeepSeek"); err != nil {
		t.Fatal(err)
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.KeyModel, "deepseek-v4-flash"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.ProviderAPIKeyKey("DeepSeek"), "ds-secret"); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	_ = st.Close()

	checks := runChecks(t, dir, freeAddr(t))
	if c := byName(t, checks, "api key"); !c.OK {
		t.Fatalf("api key: %s", c.Message)
	}
}

func TestRunUnknownModel(t *testing.T) {
	dir := healthyThothDir(t)
	// A model value absent from the llm_models registry fails the model check
	// with "unknown model", even though the settings key is set.
	r, err := settings.OpenRepo(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSetting(settings.KeyModel, "not-a-real-model"); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	c := byName(t, runProviderChecks(t, dir, http.StatusOK), "model")
	if c.OK || !strings.Contains(c.Message, "unknown model") {
		t.Fatalf("model: %s", c.Message)
	}
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
	// A free address keeps the healthy run deterministic: nothing answers
	// there, so api reports "not running" (OK) and websocket is skipped (OK)
	// regardless of what occupies the fixed port on the machine. The api key
	// and model checks pass because healthyThothDir configures them.
	checks := runChecks(t, healthyThothDir(t), freeAddr(t))
	want := []string{"wiki", "provider", "api key", "model", "database", "index", "malformed", "api", "websocket"}
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
// HOME that has nothing in it: every probe that can fail must fail.
func TestRunEmptyDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	checks := runChecks(t, "", "")
	for _, name := range []string{"wiki", "provider", "database", "index"} {
		if byName(t, checks, name).OK {
			t.Fatalf("check %q must fail in an empty installation", name)
		}
	}
}

func TestRunWikiMissingFolders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(filepath.Join(wikiRoot, "inbox"), 0o755); err != nil { // partial scaffold
		t.Fatal(err)
	}
	seedWikiPath(t, dir, wikiRoot)
	c := byName(t, runChecks(t, dir, ""), "wiki")
	if c.OK || !strings.Contains(c.Message, "is missing") {
		t.Fatalf("wiki: %s", c.Message)
	}
}

// seedWikiFolders stores a custom scaffold folder set into thoth.db under dir.
func seedWikiFolders(t *testing.T, dir, folders string) {
	t.Helper()
	r, err := settings.OpenRepo(filepath.Join(dir, "thoth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	if err := r.SetSetting(settings.KeyWikiFolders, folders); err != nil {
		t.Fatal(err)
	}
}

func TestRunWikiConfiguredFolders(t *testing.T) {
	// A wiki with a custom configured folder set: the check requires exactly
	// those folders, not the defaults.
	dir := t.TempDir()
	wikiRoot := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(filepath.Join(wikiRoot, "journal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiRoot, "CLAUDE.md"), []byte("# rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedWikiPath(t, dir, wikiRoot)
	seedWikiFolders(t, dir, "journal")

	checks := runChecks(t, dir, "")
	c := byName(t, checks, "wiki")
	if !c.OK || !strings.Contains(c.Message, "1 scaffold folders") {
		t.Fatalf("wiki with journal only: %s", c.Message)
	}

	// A missing folder from the configured set fails the check and is named.
	if err := os.RemoveAll(filepath.Join(wikiRoot, "journal")); err != nil {
		t.Fatal(err)
	}
	c = byName(t, runChecks(t, dir, ""), "wiki")
	if c.OK || !strings.Contains(c.Message, "journal") {
		t.Fatalf("wiki after removing journal: %s", c.Message)
	}
}

func TestRunNonThothProcessOnPort(t *testing.T) {
	dir := healthyThothDir(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	checks := runChecks(t, dir, ln.Addr().String())
	c := byName(t, checks, "api")
	if c.OK || !strings.Contains(c.Message, "non-thoth") {
		t.Fatalf("api: %s", c.Message)
	}
	ws := byName(t, checks, "websocket")
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
		case "/api/v1/health":
			_, _ = w.Write([]byte(health))
		case "/ws/v1":
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
	serveThothAPI(t, ln, `{"status":"ok","wiki":{"path":"/fake/wiki","exists":true}}`)

	checks := runChecks(t, dir, ln.Addr().String())
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
		"wiki missing": `{"status":"ok","wiki":{"exists":false}}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := healthyThothDir(t)
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = ln.Close() }()
			serveThothAPI(t, ln, health)

			c := byName(t, runChecks(t, dir, ln.Addr().String()), "api")
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
	// REST health works but the /ws/v1 upgrade does not.
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			_, _ = w.Write([]byte(`{"status":"ok","wiki":{"exists":true}}`))
			return
		}
		http.NotFound(w, r)
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	checks := runChecks(t, dir, ln.Addr().String())
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
	seedSettingsRaw(t, dir, filepath.Join(dir, "wiki"), "DELETE")

	c := byName(t, runChecks(t, dir, ""), "database")
	if c.OK || !strings.Contains(c.Message, `journal mode is "delete"`) {
		t.Fatalf("database: %s", c.Message)
	}
}

func TestRunDatabaseMissingTables(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	seedSettingsRaw(t, dir, filepath.Join(dir, "wiki"), "WAL")

	c := byName(t, runChecks(t, dir, ""), "database")
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
	c := byName(t, runChecks(t, dir, ""), "database")
	if c.OK || !strings.Contains(c.Message, "not a usable sqlite database") {
		t.Fatalf("database: %s", c.Message)
	}
}

func TestRunMissingDatabaseIndex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	// thoth.db was never created: both the database and the index checks
	// must fail, each with its own message.
	checks := runChecks(t, dir, "")
	if db := byName(t, checks, "database"); db.OK || !strings.Contains(db.Message, "does not exist") {
		t.Fatalf("database: %s", db.Message)
	}
	if ix := byName(t, checks, "index"); ix.OK {
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
	c := byName(t, runChecks(t, dir, ""), "index")
	if c.OK || !strings.Contains(c.Message, "notes on disk") {
		t.Fatalf("index: %s", c.Message)
	}
}

func TestRunMalformedNotes(t *testing.T) {
	dir := healthyThothDir(t)
	// A note without frontmatter is silently skipped by the index; the
	// malformed check must surface it by name.
	if err := os.WriteFile(filepath.Join(dir, "wiki", "inbox", "plain.md"),
		[]byte("just some text, no frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An advisory violation (missing type) still indexes — it must not fail
	// the malformed check.
	if err := os.WriteFile(filepath.Join(dir, "wiki", "knowledge", "topic.md"),
		[]byte("---\ntitle: Topic\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := byName(t, runChecks(t, dir, ""), "malformed")
	if c.OK || !strings.Contains(c.Message, "inbox/plain.md") {
		t.Fatalf("malformed: %s", c.Message)
	}
	if strings.Contains(c.Message, "topic.md") {
		t.Fatalf("advisory violations must not fail the malformed check: %s", c.Message)
	}
}

func TestCheckMalformed(t *testing.T) {
	write := func(root, rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A missing root is reported, not silently OK.
	c := checkMalformed(filepath.Join(t.TempDir(), "missing"))
	if c.OK || !strings.Contains(c.Message, "cannot scan") {
		t.Fatalf("missing root: %s", c.Message)
	}

	root := t.TempDir()
	// Valid and advisory-only notes parse; a non-indexable hidden file and a
	// non-markdown attachment are skipped; only parse failures are flagged.
	write(root, "inbox/ok.md", "---\ntitle: OK\ntype: inbox\n---\nbody\n")
	write(root, "knowledge/legacy.md", "---\ntitle: Legacy\ntype: note\n---\nbody\n")
	write(root, "meetings/notes.md", "---\ntitle: No Type\n---\nbody\n") // advisory only
	write(root, ".hidden.md", "no frontmatter\n")                        // not indexable
	write(root, "attachments/install.sh", "#!/bin/sh\n")                 // not markdown
	if c := checkMalformed(root); !c.OK {
		t.Fatalf("clean wiki: %s", c.Message)
	}

	// More than five parse failures are capped with a "+N more" tail.
	for i := 0; i < 7; i++ {
		write(root, fmt.Sprintf("inbox/bad-%d.md", i), "no frontmatter\n")
	}
	c = checkMalformed(root)
	if c.OK || !strings.Contains(c.Message, "and 2 more") {
		t.Fatalf("capped list: %s", c.Message)
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
	if !byName(t, runChecks(t, dir, ""), "index").OK {
		t.Fatalf("index must ignore non-indexable notes")
	}
}

func TestRunDefaultWikiPathWhenDatabaseMissing(t *testing.T) {
	// No database at all: the wiki check probes the seeded default path
	// (which does not exist under the temp HOME).
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	c := byName(t, runChecks(t, dir, ""), "wiki")
	if c.OK || !strings.Contains(c.Message, filepath.Join(home, ".thoth", "wiki")) {
		t.Fatalf("wiki: %s", c.Message)
	}
}
