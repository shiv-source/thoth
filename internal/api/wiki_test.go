package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-warehouse/events"
	"github.com/shiv-source/thoth/internal/wiki"
)

// exportReq hits GET /api/v1/wiki/export.
func exportReq(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wiki/export"+query, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// importReq posts a wiki zip (built from srcRoot) to POST /api/v1/wiki/import.
func importReq(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, zipBytes []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "wiki.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(zipBytes); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wiki/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// zipWiki builds a zip from the given root (all files, wiki-relative paths).
func zipWiki(t *testing.T, root string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := wiki.New(root).ExportTo(&buf, wiki.ExportOptions{}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExportWikiStreamsZip(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wiki")
	if err := wiki.Scaffold(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inbox", "hello.md"), []byte("---\ntitle: Hello\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.Wiki = wiki.New(root)
	e := New(d)

	rec := exportReq(t, e, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("export status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("content-type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, `attachment; filename="thoth-wiki-`) {
		t.Fatalf("content-disposition = %q", cd)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("response is not a valid zip: %v", err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "inbox/hello.md" {
			found = true
		}
	}
	if !found {
		t.Fatal("export zip missing inbox/hello.md")
	}
}

func TestExportWikiMissingRoot(t *testing.T) {
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "missing"))
	e := New(d)
	if rec := exportReq(t, e, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("export of a missing wiki = %d, want 404", rec.Code)
	}
}

func TestImportWikiHappyPathReflectsInIndex(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if err := wiki.Scaffold(src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "inbox", "hello.md"), []byte("---\ntitle: Hello\n---\nImported body"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "wiki"))
	e := New(d)

	rec := importReq(t, e, zipWiki(t, src))
	if rec.Code != http.StatusOK {
		t.Fatalf("import status %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Files  int    `json:"files"`
		Backup string `json:"backup"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Files == 0 {
		t.Fatal("import reported 0 files")
	}
	// The index reflects the imported tree: a search finds the note.
	results, err := d.Index.Search(`"hello"`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].Path != "inbox/hello.md" {
		t.Fatalf("search after import = %+v", results)
	}
}

func TestImportWikiRejectsMissingFile(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wiki/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing file = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestImportWikiRejectsMalformedZip(t *testing.T) {
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "wiki"))
	e := New(d)
	if rec := importReq(t, e, []byte("not a zip")); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed zip = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestImportWikiRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create("../evil.txt"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "wiki"))
	e := New(d)
	rec := importReq(t, e, buf.Bytes())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "escapes") {
		t.Fatalf("traversal = %d %s, want 400 with an escape error", rec.Code, rec.Body.String())
	}
}

func TestExportWikiStreamError(t *testing.T) {
	// An unreadable wiki root fails the export before any zip bytes are
	// written, so the 500 lands instead of a truncated stream.
	root := filepath.Join(t.TempDir(), "wiki")
	if err := wiki.Scaffold(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

	d := testDeps(t)
	d.Wiki = wiki.New(root)
	e := New(d)
	// The streamed zip commits 200 before the walk error surfaces, so the
	// failure is visible as the internal-error body rather than the status.
	rec := exportReq(t, e, "")
	if !strings.Contains(rec.Body.String(), "internal error") {
		t.Fatalf("export of an unreadable wiki: want an internal-error body, got status %d %q", rec.Code, rec.Body.String())
	}
}

func TestImportWikiReindexError(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if err := wiki.Scaffold(src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "inbox", "hello.md"), []byte("---\ntitle: Hello\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "wiki"))
	if err := d.Index.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	if rec := importReq(t, e, zipWiki(t, src)); rec.Code != http.StatusInternalServerError {
		t.Fatalf("import with a closed index = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestImportWikiClosedBusPublish(t *testing.T) {
	// The wiki_changed publish is best-effort: a closed bus must not turn a
	// successful import into an error.
	src := filepath.Join(t.TempDir(), "src")
	if err := wiki.Scaffold(src); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "wiki"))
	bus := events.New()
	bus.Close()
	d.Events = bus
	e := New(d)
	if rec := importReq(t, e, zipWiki(t, src)); rec.Code != http.StatusOK {
		t.Fatalf("import with a closed bus = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestImportWikiPublishesChange(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if err := wiki.Scaffold(src); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "wiki"))
	bus := events.New()
	d.Events = bus
	e := New(d)

	// Subscribe before importing so the published wiki_changed batch is
	// observed (the UI refetches the tree on it).
	got := make(chan wiki.Changed, 1)
	if err := events.Subscribe(bus, context.Background(), func(ev events.Event[wiki.Changed]) {
		got <- ev.Data
	}); err != nil {
		t.Fatal(err)
	}

	rec := importReq(t, e, zipWiki(t, src))
	if rec.Code != http.StatusOK {
		t.Fatalf("import status %d: %s", rec.Code, rec.Body.String())
	}
	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("no wiki_changed event published after import")
	}
}
