package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	// The schema lives in the store's migrations; the index and the settings
	// repo open the same file afterwards.
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ix, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	gh, err := github.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gh.Close() })
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stg.Close() })
	return Deps{
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Store:    st,
		Claude:   &FakeClient{},
		GitHub:   &github.Service{Repo: gh},
		Settings: stg,
		DataDir:  t.TempDir(),
		Version:  "test-version",
		Wiki:     wiki.New(t.TempDir()),
		Index:    ix,
	}
}

func TestHealth(t *testing.T) {
	e := New(testDeps(t))
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Status != "ok" {
		t.Fatalf("body: %v %s", err, rec.Body.String())
	}
	if body.Version != "test-version" {
		t.Fatalf("version = %q, want test-version", body.Version)
	}
}

func TestHealthDev(t *testing.T) {
	tests := []struct {
		name string
		dev  bool
	}{
		{"prod", false},
		{"dev", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := testDeps(t)
			deps.Dev = tt.dev
			e := New(deps)
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d", rec.Code)
			}
			var body struct {
				Dev bool `json:"dev"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body: %v %s", err, rec.Body.String())
			}
			if body.Dev != tt.dev {
				t.Fatalf("dev = %v, want %v", body.Dev, tt.dev)
			}
		})
	}
}

func TestHealthDefaultWikiPath(t *testing.T) {
	deps := testDeps(t)
	deps.DefaultWikiPath = "~/.thoth/dev/wiki"
	e := New(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		DefaultWikiPath string `json:"default_wiki_path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v %s", err, rec.Body.String())
	}
	if body.DefaultWikiPath != "~/.thoth/dev/wiki" {
		t.Fatalf("default_wiki_path = %q, want ~/.thoth/dev/wiki", body.DefaultWikiPath)
	}
}

func TestHealthCommit(t *testing.T) {
	deps := testDeps(t)
	deps.Commit = "abc1234"
	e := New(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Commit string `json:"commit"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v %s", err, rec.Body.String())
	}
	if body.Commit != "abc1234" {
		t.Fatalf("commit = %q, want abc1234", body.Commit)
	}
}
