package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/wiki"
)

func postCapture(t *testing.T, e http.Handler, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capture", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var parsed map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	}
	return rec, parsed
}

func TestCaptureBookmark(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	rec, body := postCapture(t, e, `{"kind":"bookmark","url":"https://example.com/go","title":"Go Channels","category":"reference","reason":"docs"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if body["path"] != wiki.BookmarkFile || body["type"] != "bookmark" || body["title"] != "Go Channels" {
		t.Fatalf("body = %+v", body)
	}

	// The line landed in links/bookmarks.md in the rulebook format.
	content, err := d.Wiki.Read(wiki.BookmarkFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "- [Go Channels](https://example.com/go) — docs") {
		t.Fatalf("bookmark line missing:\n%s", content)
	}
	if problems := wiki.Validate(wiki.BookmarkFile, content); len(problems) > 0 {
		t.Fatalf("bookmarks file not valid: %+v", problems)
	}
}

func TestCaptureBookmarkDuplicateIs409(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	if _, err := d.Wiki.Bookmark(wiki.Bookmark{Title: "Existing", URL: "https://example.com/dup"}); err != nil {
		t.Fatal(err)
	}

	rec, body := postCapture(t, e, `{"kind":"bookmark","url":"https://example.com/dup","title":"Again"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409: %s", rec.Code, rec.Body.String())
	}
	if body["error"] != "url already saved" || body["path"] != wiki.BookmarkFile {
		t.Fatalf("body = %+v", body)
	}
}

func TestCaptureNoteWithSource(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	rec, body := postCapture(t, e, `{"kind":"selection","url":"https://example.com/article","title":"Saved selection","text":"> the quote\n\nMore context.","tags":["quotes"],"folder":"knowledge"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	rel, _ := body["path"].(string)
	if rel != "knowledge/saved-selection.md" {
		t.Fatalf("path = %q", rel)
	}
	// The response title honors the request title (a selection's page title),
	// not the blockquote the body starts with.
	if body["title"] != "Saved selection" {
		t.Fatalf("title = %v", body["title"])
	}
	content, err := d.Wiki.Read(rel)
	if err != nil {
		t.Fatal(err)
	}
	meta, _, perr := wiki.ParseNote(content)
	if perr != nil {
		t.Fatal(perr)
	}
	if meta.Source != "https://example.com/article" || meta.Kind != "knowledge" {
		t.Fatalf("frontmatter = %+v", meta)
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "quotes" {
		t.Fatalf("tags = %v", meta.Tags)
	}
	if problems := wiki.Validate(rel, content); len(problems) > 0 {
		t.Fatalf("note not valid: %+v", problems)
	}
}

func TestCaptureNoteDefaultsFolder(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec, body := postCapture(t, e, `{"kind":"note","text":"# Quick thought\n\nbody"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if body["path"] != "inbox/quick-thought.md" {
		t.Fatalf("path = %v, want inbox default", body["path"])
	}
}

func TestCaptureReadLater(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec, body := postCapture(t, e, `{"kind":"readlater","url":"https://example.com/long","title":"Long read"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if body["path"] != wiki.ReadLaterFile || body["type"] != "read-later" {
		t.Fatalf("body = %+v", body)
	}
	content, err := d.Wiki.Read(wiki.ReadLaterFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "- [Long read](https://example.com/long)") {
		t.Fatalf("read-later line missing:\n%s", content)
	}
	// Duplicate read-later URLs are 409 too.
	rec, body = postCapture(t, e, `{"kind":"readlater","url":"https://example.com/long","title":"Dup"}`)
	if rec.Code != http.StatusConflict || body["path"] != wiki.ReadLaterFile {
		t.Fatalf("duplicate read-later = %d %+v", rec.Code, body)
	}
}

func TestCaptureBadInput(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	// Unknown kind.
	rec, _ := postCapture(t, e, `{"kind":"teleport"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown kind = %d, want 400", rec.Code)
	}
	// Bookmark without a URL.
	rec, _ = postCapture(t, e, `{"kind":"bookmark","title":"No url"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bookmark without url = %d, want 400", rec.Code)
	}
	// Bookmark with a non-http URL.
	rec, _ = postCapture(t, e, `{"kind":"bookmark","url":"file:///etc/passwd"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad scheme = %d, want 400", rec.Code)
	}
	// Note without text.
	rec, _ = postCapture(t, e, `{"kind":"note"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("note without text = %d, want 400", rec.Code)
	}
	// Note with an unsafe folder.
	rec, _ = postCapture(t, e, `{"kind":"note","text":"x","folder":"../evil"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unsafe folder = %d, want 400", rec.Code)
	}
	// Note with a bad source URL.
	rec, _ = postCapture(t, e, `{"kind":"note","text":"x","url":"javascript:alert(1)"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad source = %d, want 400", rec.Code)
	}
	// Read-later without a URL.
	rec, _ = postCapture(t, e, `{"kind":"readlater"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("readlater without url = %d, want 400", rec.Code)
	}
}

func TestCaptureCheck(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Wiki.Bookmark(wiki.Bookmark{Title: "A", URL: "https://example.com/a"}); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/capture/check?url="+url.QueryEscape("https://example.com/a"), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Exists bool   `json:"exists"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Exists || body.Path != wiki.BookmarkFile {
		t.Fatalf("body = %+v", body)
	}

	// A URL that is not saved reports exists=false.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/capture/check?url="+url.QueryEscape("https://example.com/new"), nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Exists {
		t.Fatalf("unsaved url should be exists=false: %+v", body)
	}

	// Missing url → 400.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/capture/check", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing url = %d, want 400", rec.Code)
	}
}

func TestCaptureInboxCount(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	// Empty wiki → zero.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capture/inbox-count", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Count != 0 {
		t.Fatalf("count = %d, want 0", body.Count)
	}

	// Two inbox captures land as notes, then the count reflects them.
	for i, title := range []string{"one", "two"} {
		if _, err := d.Wiki.Save(wiki.SaveOptions{Folder: "inbox", Title: title, Body: "body " + title}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	if err := d.Index.Sync(d.Wiki.Root(), d.Log); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/capture/inbox-count", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Count != 2 {
		t.Fatalf("count = %d, want 2", body.Count)
	}
}

func TestCaptureSummarize(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	d.Claude = &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventDelta, Text: "A concise summary of the page."},
		{Type: agentlib.EventDone},
	}}
	e := New(d)

	rec, body := postCapture(t, e, `{"kind":"note","url":"https://example.com/page","title":"Page","text":"some content"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if body["path"] != "inbox/page.md" {
		t.Fatalf("path = %v", body["path"])
	}

	// The summarize endpoint promotes the assistant's summary to knowledge/.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capture/summarize", strings.NewReader(
		`{"url":"https://example.com/page","title":"Example","text":"a long page body"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("summarize status %d: %s", rec.Code, rec.Body.String())
	}
	var sum struct {
		Path  string `json:"path"`
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Type != "knowledge" || sum.Title != "Summary: Example" {
		t.Fatalf("summary = %+v", sum)
	}
	content, err := d.Wiki.Read(sum.Path)
	if err != nil {
		t.Fatal(err)
	}
	meta, bodyContent, perr := wiki.ParseNote(content)
	if perr != nil {
		t.Fatal(perr)
	}
	if meta.Source != "https://example.com/page" {
		t.Fatalf("summary source = %q", meta.Source)
	}
	if !strings.Contains(string(bodyContent), "A concise summary") {
		t.Fatalf("summary body = %q", bodyContent)
	}
	if problems := wiki.Validate(sum.Path, content); len(problems) > 0 {
		t.Fatalf("summary note not valid: %+v", problems)
	}
}

func TestCaptureSummarizeEmptyResult(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	d.Claude = &FakeClient{} // no scripted deltas → no summary
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capture/summarize", strings.NewReader(`{"text":"page body"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("empty summary = %d, want 502", rec.Code)
	}
}

func TestCaptureReadLaterTriage(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	for _, b := range []wiki.Bookmark{
		{Title: "Long read", URL: "https://example.com/long"},
		{Title: "Another", URL: "https://example.com/another"},
	} {
		if _, err := d.Wiki.AddReadLater(b); err != nil {
			t.Fatal(err)
		}
	}
	e := New(d)

	// Listing returns the queued items.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capture/read-later", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d", rec.Code)
	}
	var body struct {
		Items []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0].URL != "https://example.com/long" {
		t.Fatalf("items = %+v", body.Items)
	}

	// Deleting one URL removes exactly it.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/capture/read-later?url="+url.QueryEscape("https://example.com/long"), nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status %d", rec.Code)
	}
	entries, err := d.Wiki.ReadLater()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].URL != "https://example.com/another" {
		t.Fatalf("after delete = %+v", entries)
	}

	// Deleting the same URL again is idempotent.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/capture/read-later?url="+url.QueryEscape("https://example.com/long"), nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("idempotent delete status %d", rec.Code)
	}

	// Missing/invalid urls are 400.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/capture/read-later", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing url = %d, want 400", rec.Code)
	}
}

func TestCaptureInboxCountIndexError(t *testing.T) {
	d := testDeps(t)
	if err := d.Index.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capture/inbox-count", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("closed index = %d, want 500", rec.Code)
	}
}

func TestCaptureSummarizeAgentError(t *testing.T) {
	d := testDeps(t)
	if err := wiki.Scaffold(d.Wiki.Root()); err != nil {
		t.Fatal(err)
	}
	d.Claude = &FakeClient{Err: errors.New("provider down")}
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capture/summarize", strings.NewReader(`{"text":"page body"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("agent error = %d, want 500", rec.Code)
	}
}

func TestCaptureSummarizeRequiresText(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capture/summarize", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing text = %d, want 400", rec.Code)
	}
}
