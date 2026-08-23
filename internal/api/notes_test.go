package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestSearchEndpoint(t *testing.T) {
	d := testDeps(t)
	root := d.Wiki.Root()
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
	if err := d.Index.Sync(root, d.Log); err != nil {
		t.Fatal(err)
	}

	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=goroutines", nil)
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

// TestSearchEndpointEmptyIsArray verifies a search with no matches serializes
// results as [] — never null — so the client's array schema accepts it.
func TestSearchEndpointEmptyIsArray(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	if err := d.Index.Sync(d.Wiki.Root(), d.Log); err != nil {
		t.Fatal(err)
	}

	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=zzzznomatch", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body.Results) != "[]" {
		t.Fatalf("results = %s, want []", body.Results)
	}
}

func TestNoteEndpoint(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: Standup\ntype: meeting\n---\nnotes\n"
	p := filepath.Join(d.Wiki.Root(), "meetings", "s.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes?path=meetings/s.md", nil)
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
	req = httptest.NewRequest(http.MethodGet, "/api/v1/notes?path=../secret", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for escaping path, got %d", rec.Code)
	}
}

func TestNoteEndpointServesAttachmentsAsRawBytes(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	attachments := filepath.Join(d.Wiki.Root(), "attachments")
	if err := os.MkdirAll(attachments, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attachments, "logo.png"), []byte("png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attachments, "install.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(d)

	// Images are served inline as raw bytes so an <img> can preview them.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes?path=attachments/logo.png", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for image attachment, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != "" {
		t.Fatalf("image must not be an attachment, got Content-Disposition %q", got)
	}
	if got := rec.Body.String(); got != "png-bytes" {
		t.Fatalf("body = %q, want png-bytes", got)
	}

	// Other attachments are downloads carrying their basename.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/notes?path=attachments/install.sh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for script attachment, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="install.sh"` {
		t.Fatalf("Content-Disposition = %q, want attachment with basename", got)
	}
	if got := rec.Body.String(); got != "#!/bin/sh\n" {
		t.Fatalf("body = %q, want the file's raw bytes", got)
	}
}

func TestTreeEndpoint(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wiki/tree", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=go&limit=abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with defaulted limit, got %d: %s", rec.Code, rec.Body.String())
	}
	// Out-of-range limits are clamped to the default as well.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/search?q=go&limit=9999", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=go", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on closed index, got %d", rec.Code)
	}
}

func TestNoteEndpointRequiresPath(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes?path=meetings/nope.md", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing note, got %d", rec.Code)
	}
}

func TestTreeEndpointMissingRoot(t *testing.T) {
	d := testDeps(t)
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "missing"))
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wiki/tree", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on missing root, got %d", rec.Code)
	}
}

func TestCreateNoteEndpoint(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"# The Answer\n\nbody","folder":"knowledge"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Path  string `json:"path"`
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Path != "knowledge/the-answer.md" {
		t.Fatalf("path = %q, want knowledge/the-answer.md", body.Path)
	}
	if body.Title != "The Answer" || body.Type != "knowledge" {
		t.Fatalf("title/type = %q/%q", body.Title, body.Type)
	}

	// The file landed as a valid, frontmattered note under the wiki root.
	content, err := os.ReadFile(filepath.Join(d.Wiki.Root(), "knowledge", "the-answer.md"))
	if err != nil {
		t.Fatal(err)
	}
	meta, _, perr := wiki.ParseNote(content)
	if perr != nil {
		t.Fatal(perr)
	}
	if meta.Title != "The Answer" || meta.Kind != "knowledge" {
		t.Fatalf("frontmatter = %+v", meta)
	}
	if problems := wiki.Validate("knowledge/the-answer.md", content); len(problems) > 0 {
		t.Fatalf("note not valid: %+v", problems)
	}

	// The promoted note is searchable after an index sync, matching the
	// "appears in the tree within ~200 ms" expectation (the watcher syncs).
	if err := d.Index.Sync(d.Wiki.Root(), d.Log); err != nil {
		t.Fatal(err)
	}
	results, err := d.Index.Search(`"The Answer"`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != "knowledge/the-answer.md" {
		t.Fatalf("search results = %+v", results)
	}
}

func TestCreateNoteEndpointDefaultsFolder(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"# Solo\n\nbody"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Path != "inbox/solo.md" {
		t.Fatalf("default folder path = %q, want inbox/solo.md", body.Path)
	}
}

func TestCreateNoteEndpointUsesConfiguredFolders(t *testing.T) {
	d := testDeps(t)
	if err := d.Settings.SetSetting(settings.KeyWikiFolders, "journal, recipes"); err != nil {
		t.Fatal(err)
	}
	if err := wiki.ScaffoldWithOptions(d.Wiki.Root(), wiki.ScaffoldOptions{Folders: []string{"journal", "recipes"}, GitInit: false}); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"# J\n\nbody"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Path != "journal/j.md" {
		t.Fatalf("default folder path = %q, want journal/j.md", body.Path)
	}
}

func TestCreateNoteEndpointRequiresContent(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"  "}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty content, got %d", rec.Code)
	}
}

func TestCreateNoteEndpointRejectsUnsafeFolder(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	for _, folder := range []string{"../evil", "a/b", ".hidden", "attachments"} {
		body := `{"content":"# X\n\nbody","folder":"` + folder + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("folder %q: expected 400, got %d", folder, rec.Code)
		}
	}
}

func TestCreateNoteEndpointEscapingFolderDoesNotWrite(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"# X\n\nbody","folder":"../secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for escaping folder, got %d", rec.Code)
	}
	// Nothing escaped the wiki root: no secret dir outside it.
	if _, err := os.Stat(filepath.Join(d.Wiki.Root(), "..", "secret")); !os.IsNotExist(err) {
		t.Fatal("escaping folder wrote outside the wiki root")
	}
}
