package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListDirsEndpoint(t *testing.T) {
	e := New(testDeps(t))
	root := t.TempDir()
	for _, d := range []string{"beta", "Alpha", "gamma"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Files are not directories and must be excluded.
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+root, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Dirs []string `json:"dirs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Only subdirectories, sorted lexically (Alpha before beta).
	want := []string{filepath.Join(root, "Alpha"), filepath.Join(root, "beta"), filepath.Join(root, "gamma")}
	if !reflect.DeepEqual(body.Dirs, want) {
		t.Fatalf("dirs = %v, want %v", body.Dirs, want)
	}
}

func TestListDirsEndpointErrors(t *testing.T) {
	e := New(testDeps(t))
	// Missing path parameter.
	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing path: status %d, want 400", rec.Code)
	}
	// Path that does not exist.
	req = httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+filepath.Join(t.TempDir(), "missing"), nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing dir: status %d, want 400", rec.Code)
	}
	// Path pointing at a file.
	f := filepath.Join(t.TempDir(), "file.md")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+f, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("file path: status %d, want 400", rec.Code)
	}
}
