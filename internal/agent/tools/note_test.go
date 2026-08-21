package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/wiki"
)

func TestWikiToolSchemas(t *testing.T) {
	fs := newTestFS(t)
	now := func() time.Time { return time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC) }
	tools := []struct {
		name string
		got  string
	}{
		{NewWriteNote(fs, now).Name(), "write_note"},
		{NewReadNote(fs, 0).Name(), "read_note"},
		{NewListTree(fs, 0, 0).Name(), "list_tree"},
		{NewListRecent(fs, 0).Name(), "list_recent"},
		{NewSearchByTag(fs, 0).Name(), "search_by_tag"},
		{NewGetTodos(fs, "").Name(), "get_todos"},
		{NewUpdateTodos(fs, "").Name(), "update_todos"},
		{NewGetInbox(fs, "").Name(), "get_inbox"},
		{NewFileInbox(fs, "").Name(), "file_inbox"},
		{NewRemember(fs, "", now).Name(), "remember"},
	}
	for _, tc := range tools {
		if tc.got != tc.name {
			t.Fatalf("tool name = %q, want %q", tc.name, tc.got)
		}
	}
}

func TestWikiToolNamesAndSchemasPopulated(t *testing.T) {
	fs := newTestFS(t)
	now := func() time.Time { return time.Now() }
	// tooler is the common surface every wiki tool implements.
	type tooler interface {
		Name() string
		Description() string
		Schema() map[string]any
	}
	var tools []tooler
	for _, tl := range []any{
		NewWriteNote(fs, now), NewReadNote(fs, 0), NewListTree(fs, 0, 0),
		NewListRecent(fs, 0), NewSearchByTag(fs, 0), NewGetTodos(fs, ""),
		NewUpdateTodos(fs, ""), NewGetInbox(fs, ""), NewFileInbox(fs, ""),
		NewRemember(fs, "", now),
	} {
		tools = append(tools, tl.(tooler))
	}
	for _, tl := range tools {
		if tl.Name() == "" || tl.Description() == "" || tl.Schema()["type"] != "object" {
			t.Fatalf("tool %q missing name/description/schema", tl.Name())
		}
	}
}

func TestNoteTypeFor(t *testing.T) {
	cases := []struct {
		rel  string
		want string
	}{
		{"meetings/2026-08-21-x.md", "meeting"},
		{"projects/foo.md", "project"},
		{"setup/config.md", "setup"}, // ends in "p"; the rule only trims a trailing "s"
		{"root.md", ""},
	}
	for _, tc := range cases {
		if got := noteTypeFor(tc.rel); got != tc.want {
			t.Fatalf("noteTypeFor(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}

func TestWalkNotesSkipsNonNotesAndHidden(t *testing.T) {
	fs := newTestFS(t)
	writeNote(t, fs, "a.md", wiki.NoteMeta{Title: "A"}, "")
	writeNote(t, fs, "b.txt", wiki.NoteMeta{Title: "B"}, "")
	if err := fs.WriteFile(".hidden.md", []byte("---\ntitle: Hidden\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("CLAUDE.md", []byte("# rules"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("attachments", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("attachments/x.md", []byte("---\ntitle: Asset\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got []string
	err := walkNotes(fs, ".", func(rel string, _ wiki.NoteMeta, _ []byte) error {
		got = append(got, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walkNotes: %v", err)
	}
	want := []string{"a.md"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("walkNotes = %v, want %v", got, want)
	}
}
