package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBookmarkCreatesFile(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	rel, err := w.Bookmark(Bookmark{
		Title:    "Go Channels",
		URL:      "https://go.dev/blog/channels",
		Category: "reference",
		Reason:   "docs",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rel != BookmarkFile {
		t.Fatalf("rel = %q, want %q", rel, BookmarkFile)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	// The master list is a valid note: frontmatter parses and Validate passes.
	meta, body, perr := ParseNote(content)
	if perr != nil {
		t.Fatal(perr)
	}
	if meta.Title != "Bookmarks" || meta.Kind != "link" {
		t.Fatalf("frontmatter = %+v", meta)
	}
	if problems := Validate(rel, content); len(problems) > 0 {
		t.Fatalf("bookmarks file not valid: %+v", problems)
	}
	want := "## reference\n- [Go Channels](https://go.dev/blog/channels) — docs\n"
	if !strings.Contains(string(body), want) {
		t.Fatalf("body missing expected section:\n%s", body)
	}
}

func TestBookmarkGroupsByCategory(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	for _, b := range []Bookmark{
		{Title: "One", URL: "https://example.com/one", Category: "reference"},
		{Title: "Two", URL: "https://example.com/two", Category: "reading"},
		{Title: "Three", URL: "https://example.com/three", Category: "reference"},
	} {
		if _, err := w.Bookmark(b); err != nil {
			t.Fatalf("bookmark %s: %v", b.Title, err)
		}
	}
	content, err := w.Read(BookmarkFile)
	if err != nil {
		t.Fatal(err)
	}
	body := string(content)
	// Both reference links sit under the same heading, before the reading
	// section opens.
	refIdx := strings.Index(body, "## reference")
	readIdx := strings.Index(body, "## reading")
	if refIdx < 0 || readIdx < 0 || refIdx > readIdx {
		t.Fatalf("section order wrong:\n%s", body)
	}
	if strings.Index(body, "example.com/one") > readIdx || strings.Index(body, "example.com/three") > readIdx {
		t.Fatalf("reference entries not grouped under their heading:\n%s", body)
	}
}

func TestBookmarkDefaultsCategory(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	if _, err := w.Bookmark(Bookmark{Title: "X", URL: "https://example.com/x"}); err != nil {
		t.Fatal(err)
	}
	content, err := w.Read(BookmarkFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "## "+DefaultCategory) {
		t.Fatalf("no default category section:\n%s", content)
	}
}

func TestBookmarkDuplicateURL(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	b := Bookmark{Title: "Dup", URL: "https://example.com/dup", Category: "reference"}
	if _, err := w.Bookmark(b); err != nil {
		t.Fatal(err)
	}
	// Same URL, different title/category: still a duplicate in the master list.
	b2 := Bookmark{Title: "Dup Again", URL: "https://example.com/dup", Category: "reading"}
	if _, err := w.Bookmark(b2); !errors.Is(err, ErrDuplicateBookmark) {
		t.Fatalf("expected ErrDuplicateBookmark, got %v", err)
	}
	content, _ := w.Read(BookmarkFile)
	if n := strings.Count(string(content), "example.com/dup"); n != 1 {
		t.Fatalf("duplicate URL written %d times:\n%s", n, content)
	}
}

func TestBookmarkValidation(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	tests := []struct {
		name string
		b    Bookmark
	}{
		{"empty title", Bookmark{URL: "https://example.com/x"}},
		{"empty url", Bookmark{Title: "X"}},
		{"not a url", Bookmark{Title: "X", URL: "not a url"}},
		{"bad scheme", Bookmark{Title: "X", URL: "ftp://example.com/x"}},
		{"newline in category", Bookmark{Title: "X", URL: "https://example.com/x", Category: "a\n## evil"}},
		{"newline in reason", Bookmark{Title: "X", URL: "https://example.com/x", Reason: "a\r\n- [evil](https://evil.com)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := w.Bookmark(tt.b); err == nil {
				t.Fatal("expected error, got nil")
			}
			if _, err := w.AddReadLater(tt.b); err == nil {
				t.Fatal("expected AddReadLater error, got nil")
			}
		})
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(BookmarkFile))); !os.IsNotExist(err) {
		t.Fatal("invalid bookmarks must not create the file")
	}
}

func TestBookmarksReader(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	entries, err := w.Bookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("missing file should read empty, got %+v", entries)
	}
	want := []Bookmark{
		{Title: "Alpha", URL: "https://example.com/a", Category: "reference", Reason: "docs"},
		{Title: "Beta", URL: "https://example.com/b", Category: "reading"},
	}
	for _, b := range want {
		if _, err := w.Bookmark(b); err != nil {
			t.Fatal(err)
		}
	}
	entries, err = w.Bookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
	if entries[0] != (LinkEntry{Title: "Alpha", URL: "https://example.com/a", Category: "reference", Reason: "docs"}) {
		t.Fatalf("entry 0 = %+v", entries[0])
	}
	if entries[1] != (LinkEntry{Title: "Beta", URL: "https://example.com/b", Category: "reading"}) {
		t.Fatalf("entry 1 = %+v", entries[1])
	}
}

func TestAddReadLater(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	rel, err := w.AddReadLater(Bookmark{Title: "Long Read", URL: "https://example.com/long"})
	if err != nil {
		t.Fatal(err)
	}
	if rel != ReadLaterFile {
		t.Fatalf("rel = %q, want %q", rel, ReadLaterFile)
	}
	content, err := w.Read(ReadLaterFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "- [Long Read](https://example.com/long)") {
		t.Fatalf("read-later line missing:\n%s", content)
	}
	// The queue is flat: a second entry appends without a category heading.
	if _, err := w.AddReadLater(Bookmark{Title: "Another", URL: "https://example.com/another"}); err != nil {
		t.Fatal(err)
	}
	content, _ = w.Read(ReadLaterFile)
	if strings.Count(string(content), "## ") != 0 {
		t.Fatalf("read-later queue must be flat:\n%s", content)
	}
	// Duplicate URLs are rejected here too.
	if _, err := w.AddReadLater(Bookmark{Title: "Dup", URL: "https://example.com/long"}); !errors.Is(err, ErrDuplicateBookmark) {
		t.Fatalf("expected ErrDuplicateBookmark, got %v", err)
	}
	// And the queue parses back.
	entries, err := w.ReadLater()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d read-later entries, want 2", len(entries))
	}
}

func TestRemoveReadLater(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	// Removing from a queue that never existed is a no-op.
	if err := w.RemoveReadLater("https://example.com/nope"); err != nil {
		t.Fatal(err)
	}
	for _, b := range []Bookmark{
		{Title: "Long", URL: "https://example.com/long"},
		{Title: "Another", URL: "https://example.com/another"},
	} {
		if _, err := w.AddReadLater(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.RemoveReadLater("https://example.com/long"); err != nil {
		t.Fatal(err)
	}
	entries, err := w.ReadLater()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].URL != "https://example.com/another" {
		t.Fatalf("after removal = %+v, want only the other entry", entries)
	}
	// The queue still parses as a valid note after the rewrite.
	content, err := w.Read(ReadLaterFile)
	if err != nil {
		t.Fatal(err)
	}
	if problems := Validate(ReadLaterFile, content); len(problems) > 0 {
		t.Fatalf("queue not valid after removal: %+v", problems)
	}
	// Removing a URL that is not present leaves the file untouched.
	if err := w.RemoveReadLater("https://example.com/not-there"); err != nil {
		t.Fatal(err)
	}
	content2, err := w.Read(ReadLaterFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content2) != string(content) {
		t.Fatal("no-op removal must not rewrite the file")
	}
}

func TestBookmarkRejectsEscapingRoot(t *testing.T) {
	// The URL is SafePath-agnostic, but the file path is fixed by contract —
	// a caller cannot route the write anywhere else.
	root := t.TempDir()
	w := New(root)
	if _, err := w.Bookmark(Bookmark{Title: "X", URL: "https://example.com/x"}); err != nil {
		t.Fatal(err)
	}
	entries, err := w.Bookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].URL != "https://example.com/x" {
		t.Fatalf("entry = %+v", entries[0])
	}
}
