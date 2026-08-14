package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestSearchEndpoint(t *testing.T) {
	d := testDeps(t)
	root := d.Wiki.Root
	if err := wiki.Scaffold(root); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "knowledge", "go.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: Go\ntype: knowledge\n---\nchannels and goroutines\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.Index.Rebuild(root, d.Log); err != nil {
		t.Fatal(err)
	}

	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=goroutines", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Results []index.Result `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Results) != 1 || body.Results[0].Path != "knowledge/go.md" {
		t.Fatalf("unexpected results: %+v", body.Results)
	}
}

func TestNoteEndpoint(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: Standup\ntype: meeting\n---\nnotes\n"
	p := filepath.Join(d.Wiki.Root, "meetings", "s.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/notes?path=meetings/s.md", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Content != content {
		t.Fatalf("content mismatch: %q", body.Content)
	}

	// escaping paths are rejected
	req = httptest.NewRequest(http.MethodGet, "/api/notes?path=../secret", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for escaping path, got %d", rec.Code)
	}
}

func TestTreeEndpoint(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/wiki/tree", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Nodes []wiki.Node `json:"nodes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Nodes) == 0 {
		t.Fatal("expected tree nodes")
	}
}

func TestSearchEndpointRequiresQuery(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without q, got %d", rec.Code)
	}
}

func TestSearchEndpointDefaultsBadLimit(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	// A non-numeric limit falls back to the default and still searches.
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=go&limit=abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with defaulted limit, got %d: %s", rec.Code, rec.Body.String())
	}
	// Out-of-range limits are clamped to the default as well.
	req = httptest.NewRequest(http.MethodGet, "/api/search?q=go&limit=9999", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with clamped limit, got %d", rec.Code)
	}
}

func TestSearchEndpointIndexError(t *testing.T) {
	d := testDeps(t)
	if err := d.Index.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=go", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on closed index, got %d", rec.Code)
	}
}

func TestNoteEndpointRequiresPath(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without path, got %d", rec.Code)
	}
}

func TestNoteEndpointMissingNote(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	// The path is safe but the file does not exist: 404.
	req := httptest.NewRequest(http.MethodGet, "/api/notes?path=meetings/nope.md", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing note, got %d", rec.Code)
	}
}

func TestTreeEndpointMissingRoot(t *testing.T) {
	d := testDeps(t)
	d.Wiki = wiki.Open(filepath.Join(t.TempDir(), "missing"))
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/wiki/tree", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on missing root, got %d", rec.Code)
	}
}
