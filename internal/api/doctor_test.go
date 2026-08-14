package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

// freePort returns a port that was free at the moment of the call.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// healthyThothDir builds a fully healthy installation in a temp dir: a wiki
// with one note, a synced index, a config with a free port, and a fake claude
// on PATH. Returns the dir; d.ConfigPath must point at dir/config.toml.
func healthyThothDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	claude := filepath.Join(t.TempDir(), "claude")
	script := "#!/bin/sh\ncase \"$1\" in\n  --version) echo \"9.9.9\"; exit 0 ;;\n  auth) exit 0 ;;\n  *) exit 1 ;;\nesac\n"
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(claude))

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
	if err := ix.Rebuild(wikiRoot, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
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

func TestDoctorEndpointHealthy(t *testing.T) {
	d := testDeps(t)
	d.ConfigPath = filepath.Join(healthyThothDir(t), "config.toml")
	e := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/doctor", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Checks []struct {
			Name    string `json:"name"`
			OK      bool   `json:"ok"`
			Message string `json:"message"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := []string{"config", "wiki", "claude", "claude login", "database", "index", "port"}
	if len(body.Checks) != len(want) {
		t.Fatalf("got %d checks, want %d: %+v", len(body.Checks), len(want), body.Checks)
	}
	for i, name := range want {
		c := body.Checks[i]
		if c.Name != name {
			t.Fatalf("check %d: got %q, want %q", i, c.Name, name)
		}
		if !c.OK {
			t.Fatalf("check %q failed in a healthy env: %s", name, c.Message)
		}
	}
}
