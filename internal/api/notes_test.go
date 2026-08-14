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
