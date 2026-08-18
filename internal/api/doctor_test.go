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

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

// healthyThothDir builds a fully healthy installation in a temp dir: a wiki
// with one note, a synced index, a settings row pointing at the wiki, and a
// fake claude on PATH. Returns the dir; d.DataDir must point at it.
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
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := stg.SetSetting(settings.KeyWikiPath, wikiRoot); err != nil {
		t.Fatal(err)
	}
	if err := stg.Close(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDoctorEndpointHealthy(t *testing.T) {
	d := testDeps(t)
	d.DataDir = healthyThothDir(t)

	// Point the api/websocket probes at a port this test controls: CI runners
	// occasionally have something squatting on 127.0.0.1:8333, which flips
	// the api check from the expected "not running" (OK) to "occupied by a
	// non-thoth process" (not OK). A just-closed free port makes the check
	// deterministic without depending on the fixed port being free.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d.DoctorAddr = ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	e := New(d)

	// Nothing listens on the probed port: the api check reports "not running"
	// (OK) and the websocket check is skipped (OK) — zero fixed-port dependency.
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
	want := []string{"wiki", "claude", "claude login", "database", "index", "api", "websocket"}
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
