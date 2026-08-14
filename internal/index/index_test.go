package index

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUpsertSearchDelete(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { ix.Close() })

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

func TestDeletePrefixRemovesSubtree(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
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
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
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
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
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
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
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
	if err := ix.Rebuild(t.TempDir(), discardLog()); err == nil {
		t.Fatal("Rebuild on closed index must error")
	}
}

func TestSearchRejectsInvalidQuery(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	// An unterminated phrase is a syntax error in FTS5's MATCH query.
	if _, err := ix.Search(`"unterminated`, 10); err == nil {
		t.Fatal("expected error for malformed FTS5 query")
	}
}
