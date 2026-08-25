package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestBookmarkURLWithParensRoundTrips(t *testing.T) {
	// A URL containing ")" (Wikipedia titles, …) must round-trip through the
	// link line so dedup and the reader see the full URL, not a truncated one.
	root := t.TempDir()
	w := New(root)
	url := "https://en.wikipedia.org/wiki/Go_(programming_language)"
	if _, err := w.Bookmark(Bookmark{Title: "Go", URL: url}); err != nil {
		t.Fatal(err)
	}
	// Re-saving the same URL is a duplicate — the escaped form parsed back.
	if _, err := w.Bookmark(Bookmark{Title: "Go again", URL: url}); !errors.Is(err, ErrDuplicateBookmark) {
		t.Fatalf("expected ErrDuplicateBookmark, got %v", err)
	}
	entries, err := w.Bookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].URL != url {
		t.Fatalf("entries = %+v, want the full URL", entries)
	}
	content, err := w.Read(BookmarkFile)
	if err != nil {
		t.Fatal(err)
	}
	// The ")" is percent-encoded in the file (once) so the markdown line parses.
	if strings.Count(string(content), "Go_(programming_language)") != 0 {
		t.Fatalf("parenthesized URL must be percent-encoded in the file:\n%s", content)
	}
	if strings.Count(string(content), "%28") != 1 || strings.Count(string(content), "%29") != 1 {
		t.Fatalf("URL should be encoded exactly once:\n%s", content)
	}
}

func TestBookmarkTitleWithBracketsRoundTrips(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	title := "Go [1.22] notes]"
	if _, err := w.Bookmark(Bookmark{Title: title, URL: "https://example.com/a"}); err != nil {
		t.Fatal(err)
	}
	entries, err := w.Bookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Title != title {
		t.Fatalf("title = %q, want %q", entries[0].Title, title)
	}
}

func TestBookmarkDedupNormalizesURL(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	if _, err := w.Bookmark(Bookmark{Title: "A", URL: "https://EXAMPLE.com:443/read#section"}); err != nil {
		t.Fatal(err)
	}
	// Default port, case, and fragment collapse; scheme and path stay distinct.
	for _, dup := range []string{"https://example.com/read", "HTTPS://example.com/read#other"} {
		if _, err := w.Bookmark(Bookmark{Title: "dup", URL: dup}); !errors.Is(err, ErrDuplicateBookmark) {
			t.Fatalf("%q should dedup, got %v", dup, err)
		}
	}
	if _, err := w.Bookmark(Bookmark{Title: "b", URL: "http://example.com/read"}); err != nil {
		t.Fatalf("http and https are distinct resources, must not dedup: %v", err)
	}
	if _, err := w.Bookmark(Bookmark{Title: "c", URL: "https://example.com/read/"}); err != nil {
		t.Fatalf("trailing slash is a distinct path, must not dedup: %v", err)
	}
}

func TestRemoveReadLaterNormalizesURL(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	if _, err := w.AddReadLater(Bookmark{Title: "Long", URL: "https://example.com/long#keep"}); err != nil {
		t.Fatal(err)
	}
	// Removing with a different fragment still matches.
	if err := w.RemoveReadLater("https://example.com/long#different"); err != nil {
		t.Fatal(err)
	}
	entries, err := w.ReadLater()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("after normalized removal = %+v, want empty", entries)
	}
}

func TestBookmarkValidationRejectsMultilineTitle(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	b := Bookmark{Title: "line one\n- [evil](https://evil.com)", URL: "https://example.com/x"}
	if _, err := w.Bookmark(b); err == nil {
		t.Fatal("expected error for a multiline title")
	}
	if _, err := w.AddReadLater(b); err == nil {
		t.Fatal("expected AddReadLater error for a multiline title")
	}
}

func TestConcurrentBookmarksDoNotLoseWrites(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	const n = 8
	urls := make([]string, n)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/link/%d", i)
	}
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = w.Bookmark(Bookmark{Title: fmt.Sprintf("Link %d", i), URL: urls[i]})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("bookmark %d: %v", i, err)
		}
	}
	entries, err := w.Bookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != n {
		t.Fatalf("got %d entries, want %d (a lost update raced)", len(entries), n)
	}
}
