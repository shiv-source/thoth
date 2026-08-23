package index

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/store"
)

// openTest opens an index on a temp db whose schema was created by the
// store's SQL migrations — index.Open issues no DDL of its own.
func openTest(t *testing.T) *Index {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	ix, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	return ix
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUpsertSearchDelete(t *testing.T) {
	ix := openTest(t)

	n := Note{
		Path: "meetings/2026-08-14-standup.md", Title: "Standup", Kind: "meeting",
		Tags: []string{"standup"}, Body: "Decided to ship the search index.",
		UpdatedAt: time.Now(),
	}
	if err := ix.Upsert(n); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := ix.Search("search index", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].Path != n.Path {
		t.Fatalf("unexpected results: %+v", got)
	}
	if !strings.Contains(got[0].Snippet, "<mark>") {
		t.Fatalf("snippet should mark matches: %q", got[0].Snippet)
	}

	if err := ix.Delete(n.Path); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = ix.Search("search index", 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("expected empty results after delete: %v %+v", err, got)
	}
}

func TestSearchSnippetEscapesHTML(t *testing.T) {
	ix := openTest(t)
	// A note body containing markup must come back escaped in the snippet so
	// the frontend's dangerouslySetInnerHTML cannot execute it.
	n := Note{Path: "knowledge/xss.md", Title: "XSS",
		Kind: "knowledge", Body: `payload <img src=x onerror=alert(1)> and <mark>fake</mark> injection`,
		UpdatedAt: time.Now()}
	if err := ix.Upsert(n); err != nil {
		t.Fatal(err)
	}
	got, err := ix.Search("payload", 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("Search: %v %+v", err, got)
	}
	s := got[0].Snippet
	if strings.Contains(s, "<img") {
		t.Fatalf("snippet contains unescaped markup: %q", s)
	}
	if !strings.Contains(s, "&lt;img") {
		t.Fatalf("snippet did not escape <img: %q", s)
	}
	if strings.Contains(s, "<mark>fake</mark>") {
		t.Fatalf("note's own <mark> must be escaped, not rendered: %q", s)
	}
	// The match markers are still converted to real tags.
	if !strings.Contains(s, "<mark>payload</mark>") && !strings.Contains(s, "<mark>") {
		t.Fatalf("snippet should mark matches: %q", s)
	}
}

func TestDeletePrefixEscapesLIKEWildcards(t *testing.T) {
	ix := openTest(t)
	now := time.Now()
	notes := []Note{
		{Path: "dirs/50%/inside.md", Title: "Pct", Kind: "note", Body: "percent dir", UpdatedAt: now},
		{Path: "dirs/50X/inside.md", Title: "Wild", Kind: "note", Body: "wild dir", UpdatedAt: now},
		{Path: "dirs/a_b/inside.md", Title: "Underscore", Kind: "note", Body: "underscore dir", UpdatedAt: now},
	}
	for _, n := range notes {
		if err := ix.Upsert(n); err != nil {
			t.Fatal(err)
		}
	}
	// "50%" with a literal % must remove only the "50%" directory; without
	// escaping, LIKE "50%/%" would also match "50X".
	if err := ix.DeletePrefix("dirs/50%"); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}
	if got, err := ix.Search("percent", 10); err != nil || len(got) != 0 {
		t.Fatalf("expected literal-percent dir removed: %v %+v", err, got)
	}
	if got, err := ix.Search("wild", 10); err != nil || len(got) != 1 {
		t.Fatalf("wildcard dir must survive: %v %+v", err, got)
	}
	// "_" is likewise literal.
	if err := ix.DeletePrefix("dirs/a_b"); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}
	if got, err := ix.Search("underscore", 10); err != nil || len(got) != 0 {
		t.Fatalf("expected underscore dir removed: %v %+v", err, got)
	}
	if got, err := ix.Search("wild", 10); err != nil || len(got) != 1 {
		t.Fatalf("wildcard dir must still survive: %v %+v", err, got)
	}
}

func TestDeletePrefixRemovesSubtree(t *testing.T) {
	ix := openTest(t)
	now := time.Now()
	notes := []Note{
		{Path: "projects/doomed/a.md", Title: "A", Kind: "project", Body: "doomed alpha", UpdatedAt: now},
		{Path: "projects/doomed/sub/b.md", Title: "B", Kind: "project", Body: "doomed beta", UpdatedAt: now},
		{Path: "projects/kept.md", Title: "Kept", Kind: "project", Body: "stays", UpdatedAt: now},
	}
	for _, n := range notes {
		if err := ix.Upsert(n); err != nil {
			t.Fatal(err)
		}
	}

	if err := ix.DeletePrefix("projects/doomed"); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}

	// The whole subtree is gone, including nested directories...
	got, err := ix.Search("doomed", 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("expected no results after DeletePrefix: %v %+v", err, got)
	}
	// ...while the sibling path and the prefix path itself survive.
	got, err = ix.Search("stays", 10)
	if err != nil || len(got) != 1 || got[0].Path != "projects/kept.md" {
		t.Fatalf("expected kept note to survive: %v %+v", err, got)
	}
	// DeletePrefix on a file path removes exactly that file (prefix match
	// only), not the whole tree.
	if err := ix.DeletePrefix("projects/kept.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Search("stays", 10); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertOverwrites(t *testing.T) {
	ix := openTest(t)
	n := Note{Path: "knowledge/go.md", Title: "Go", Kind: "knowledge", Body: "v1", UpdatedAt: time.Now()}
	if err := ix.Upsert(n); err != nil {
		t.Fatal(err)
	}
	n.Body = "v2 updated"
	if err := ix.Upsert(n); err != nil {
		t.Fatal(err)
	}
	got, err := ix.Search("updated", 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("expected one result: %v %+v", err, got)
	}
}

func TestSearchLimitZeroReturnsNothing(t *testing.T) {
	ix := openTest(t)
	n := Note{Path: "knowledge/go.md", Title: "Go", Kind: "knowledge", Body: "goroutines are cheap", UpdatedAt: time.Now()}
	if err := ix.Upsert(n); err != nil {
		t.Fatal(err)
	}
	got, err := ix.Search("goroutines", 0)
	if err != nil {
		t.Fatalf("Search with limit 0: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected zero results, got %+v", got)
	}
}

func TestSearchMatchesTitleOnly(t *testing.T) {
	ix := openTest(t)
	// The query term appears only in the title, never the body: FTS5 must
	// still match it (title carries the higher bm25 weight).
	n := Note{Path: "knowledge/golang-patterns.md", Title: "Golang Patterns",
		Kind: "knowledge", Body: "completely unrelated body text", UpdatedAt: time.Now()}
	if err := ix.Upsert(n); err != nil {
		t.Fatal(err)
	}
	got, err := ix.Search("golang", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].Path != n.Path {
		t.Fatalf("expected title-only match, got %+v", got)
	}
}

func TestOpenErrorWhenPathIsDirectory(t *testing.T) {
	// sql.Open is lazy, so the failure surfaces on the first Exec: opening a
	// sqlite DB at an existing directory path must error.
	if _, err := Open(t.TempDir()); err == nil {
		t.Fatal("expected error opening index at a directory path")
	}
}

func TestClosedIndexErrors(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}
	n := Note{Path: "x.md", Title: "X", Body: "body", UpdatedAt: time.Now()}
	if err := ix.Upsert(n); err == nil {
		t.Fatal("Upsert on closed index must error")
	}
	if err := ix.Delete("x.md"); err == nil {
		t.Fatal("Delete on closed index must error")
	}
	if err := ix.DeletePrefix("x"); err == nil {
		t.Fatal("DeletePrefix on closed index must error")
	}
	if _, err := ix.Search("x", 10); err == nil {
		t.Fatal("Search on closed index must error")
	}
	if err := ix.Sync(t.TempDir(), discardLog()); err == nil {
		t.Fatal("Sync on closed index must error")
	}
}

func TestSearchRejectsInvalidQuery(t *testing.T) {
	ix := openTest(t)
	// An unterminated phrase is a syntax error in FTS5's MATCH query.
	if _, err := ix.Search(`"unterminated`, 10); err == nil {
		t.Fatal("expected error for malformed FTS5 query")
	}
}

func TestSearchCleanStripsMarkers(t *testing.T) {
	ix := openTest(t)
	// The body is deliberately messy: markup the escaping path must render
	// as HTML entities, and a phrase spanning a match so the snippet carries
	// the FTS5 markers around it.
	n := Note{Path: "knowledge/context.md", Title: "Context injection",
		Kind: "knowledge", Body: "the quick brown fox jumps over the lazy dog <b>bold</b>",
		UpdatedAt: time.Now()}
	if err := ix.Upsert(n); err != nil {
		t.Fatal(err)
	}
	got, err := ix.SearchClean("quick fox", 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("SearchClean: %v %+v", err, got)
	}
	s := got[0].Snippet
	if strings.Contains(s, "<mark>") || strings.Contains(s, "</mark>") {
		t.Fatalf("clean snippet carries mark tags: %q", s)
	}
	if strings.Contains(s, "\x01") || strings.Contains(s, "\x02") {
		t.Fatalf("clean snippet carries control markers: %q", s)
	}
	if strings.Contains(s, "&lt;b&gt;") {
		t.Fatalf("clean snippet is HTML-escaped: %q", s)
	}
	if !strings.Contains(s, "fox") {
		t.Fatalf("clean snippet lost the match text: %q", s)
	}
	// The rendering path still marks matches and escapes markup.
	marked, err := ix.Search("quick fox", 10)
	if err != nil || len(marked) != 1 {
		t.Fatalf("Search: %v %+v", err, marked)
	}
	if !strings.Contains(marked[0].Snippet, "<mark>") {
		t.Fatalf("rendered snippet should mark matches: %q", marked[0].Snippet)
	}
	if strings.Contains(marked[0].Snippet, "<b>") {
		t.Fatalf("rendered snippet should escape markup: %q", marked[0].Snippet)
	}
}
