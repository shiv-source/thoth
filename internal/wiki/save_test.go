package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKebab(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Go Channels & Goroutines", "go-channels-goroutines"},
		{"Hello, World!", "hello-world"},
		{"   Multiple   Spaces  ", "multiple-spaces"},
		{"UPPER lower 123", "upper-lower-123"},
		{"hyphen-ated words", "hyphen-ated-words"},
		{"em—dash and: colon", "em-dash-and-colon"},
		{"!!!", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := Kebab(tt.in); got != tt.want {
			t.Errorf("Kebab(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestKebabCapsAt60Runes(t *testing.T) {
	long := strings.Repeat("a", 80) + " " + strings.Repeat("b", 20)
	got := Kebab(long)
	if len(got) > 61 { // 60 runes + one possible hyphen
		t.Fatalf("Kebab stem not capped: %d runes", len(got))
	}
}

func TestDefaultTitle(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"atx heading", "# Renovate the GitHub Action\n\nbody", "Renovate the GitHub Action"},
		{"atx heading with more levels", "### Deep heading\nbody", "Deep heading"},
		{"first non-empty line", "\n\nFirst line\nsecond", "First line"},
		{"bare heading skipped", "#\n\nReal content", "Real content"},
		{"empty body", "", ""},
		{"whitespace body", "  \n\t\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultTitle(tt.body); got != tt.want {
				t.Errorf("DefaultTitle(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestValidFolder(t *testing.T) {
	tests := []struct {
		folder string
		ok     bool
	}{
		{"knowledge", true},
		{"projects", true},
		{"", false},
		{"../evil", false},
		{"a/b", false},
		{"a\\b", false},
		{".", false},
		{"..", false},
		{".hidden", false},
		{"attachments", false},
	}
	for _, tt := range tests {
		err := ValidFolder(tt.folder)
		if (err == nil) != tt.ok {
			t.Errorf("ValidFolder(%q) error = %v, want ok=%v", tt.folder, err, tt.ok)
		}
	}
}

func TestSave(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) }

	rel, err := w.Save(SaveOptions{Folder: "knowledge", Title: "Go Channels", Body: "content\n", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if rel != "knowledge/go-channels.md" {
		t.Fatalf("rel = %q, want knowledge/go-channels.md", rel)
	}
	content, err := w.Read(rel)
	if err != nil {
		t.Fatal(err)
	}
	meta, body, err := ParseNote(content)
	if err != nil {
		t.Fatalf("saved note does not parse: %v", err)
	}
	if meta.Title != "Go Channels" || meta.Date != "2026-08-23" || meta.Kind != "knowledge" {
		t.Fatalf("frontmatter = %+v", meta)
	}
	if string(body) != "content\n" {
		t.Fatalf("body = %q", body)
	}
	if problems := Validate(rel, content); len(problems) > 0 {
		t.Fatalf("saved note not valid: %+v", problems)
	}
}

func TestSaveDerivesTitleFromBody(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	rel, err := w.Save(SaveOptions{Folder: "knowledge", Body: "# The Answer\n\nbody", Now: func() time.Time {
		return time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatal(err)
	}
	if rel != "knowledge/the-answer.md" {
		t.Fatalf("rel = %q, want knowledge/the-answer.md", rel)
	}
}

func TestSaveDatePrefixFolders(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) }
	for _, folder := range []string{"meetings", "daily"} {
		rel, err := w.Save(SaveOptions{Folder: folder, Title: "Standup", Body: "body\n", Now: now})
		if err != nil {
			t.Fatalf("%s: %v", folder, err)
		}
		want := folder + "/2026-08-23-standup.md"
		if rel != want {
			t.Errorf("%s rel = %q, want %q", folder, rel, want)
		}
	}
}

func TestSaveErrors(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Now() }

	if _, err := w.Save(SaveOptions{Folder: "", Title: "x", Body: "b", Now: now}); err == nil {
		t.Error("empty folder must fail")
	}
	if _, err := w.Save(SaveOptions{Folder: "../evil", Title: "x", Body: "b", Now: now}); err == nil {
		t.Error("escaping folder must fail")
	}
	if _, err := w.Save(SaveOptions{Folder: "knowledge", Title: "", Body: "", Now: now}); err == nil {
		t.Error("empty title/body must fail")
	}
	if _, err := w.Save(SaveOptions{Folder: "attachments", Title: "x", Body: "b", Now: now}); err == nil {
		t.Error("attachments folder must fail")
	}
}

func TestSaveSourceURL(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) }
	rel, err := w.Save(SaveOptions{
		Folder:    "knowledge",
		Title:     "Capture",
		Body:      "> quote\n",
		SourceURL: "https://example.com/page",
		Now:       now,
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := w.Read(rel)
	if err != nil {
		t.Fatal(err)
	}
	meta, _, err := ParseNote(content)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Source != "https://example.com/page" {
		t.Fatalf("source = %q, want the URL", meta.Source)
	}
	if problems := Validate(rel, content); len(problems) > 0 {
		t.Fatalf("saved note not valid: %+v", problems)
	}
}

func TestSaveRejectsBadSourceURL(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Now() }
	for _, u := range []string{"not a url", "ftp://example.com/x", "javascript:alert(1)"} {
		if _, err := w.Save(SaveOptions{Folder: "knowledge", Title: "x", Body: "b", SourceURL: u, Now: now}); err == nil {
			t.Errorf("Save with source URL %q must fail", u)
		}
	}
	// Empty source is fine — most notes have no provenance.
	if _, err := w.Save(SaveOptions{Folder: "knowledge", Title: "x", Body: "b", Now: now}); err != nil {
		t.Fatalf("Save without source must succeed: %v", err)
	}
}

func TestSaveCreatesParentDirs(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC) }
	rel, err := w.Save(SaveOptions{Folder: "projects", Title: "Onboard", Body: "body", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatalf("note file not created: %v", err)
	}
}

func TestDefaultTitleStripsBlockquote(t *testing.T) {
	// A captured selection's body is a blockquote; the derived title must not
	// inherit the ">" marker.
	if got := DefaultTitle("> Welcome to the docs\n\n> — [Source](https://example.com)"); got != "Welcome to the docs" {
		t.Fatalf("DefaultTitle = %q, want %q", got, "Welcome to the docs")
	}
	if got := DefaultTitle("> just a marker"); got != "just a marker" {
		t.Fatalf("DefaultTitle = %q", got)
	}
}

func TestSaveDoesNotOverwriteOnSlugCollision(t *testing.T) {
	root := t.TempDir()
	w := New(root)
	now := func() time.Time { return time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC) }
	first, err := w.Save(SaveOptions{Folder: "inbox", Title: "Same Title", Body: "first", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if first != "inbox/same-title.md" {
		t.Fatalf("first rel = %q", first)
	}
	second, err := w.Save(SaveOptions{Folder: "inbox", Title: "Same Title", Body: "second", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if second != "inbox/same-title-2.md" {
		t.Fatalf("second rel = %q, want inbox/same-title-2.md", second)
	}
	third, err := w.Save(SaveOptions{Folder: "inbox", Title: "Same Title", Body: "third", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if third != "inbox/same-title-3.md" {
		t.Fatalf("third rel = %q, want inbox/same-title-3.md", third)
	}
	// Both files exist and hold their own body — nothing was overwritten.
	b1, err := w.Read(first)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := w.Read(second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b1), "first") || !strings.Contains(string(b2), "second") {
		t.Fatalf("collision overwrote a note: %q / %q", b1, b2)
	}
}
