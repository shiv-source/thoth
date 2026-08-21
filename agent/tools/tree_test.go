package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListTreeTool(t *testing.T) {
	fs := newTestFS(t)
	mkNote := func(path, title string) {
		t.Helper()
		content := FormatNote(Note{Title: title, Date: "2026-08-21"}, "")
		if err := writeNested(fs, path, content); err != nil {
			t.Fatal(err)
		}
	}
	mkNote("meetings/standup.md", "Standup")
	mkNote("meetings/planning.md", "Planning")
	mkNote("knowledge/ai.md", "AI Notes")
	if err := fs.MkdirAll("attachments", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("attachments/x.txt", []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("todos", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("todos/TODO.md", []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := NewListTree(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("list_tree: %v", err)
	}
	// dirs first, then notes, with titles; attachments and non-notes skipped.
	want := "knowledge/\n  ai.md — AI Notes\nmeetings/\n  planning.md — Planning\n  standup.md — Standup\ntodos/\n  TODO.md\n"
	if out != want {
		t.Fatalf("list_tree:\ngot  %q\nwant %q", out, want)
	}
}

func TestListTreeToolSubpathAndCaps(t *testing.T) {
	fs := newTestFS(t)
	mkNote := func(path, title string) {
		t.Helper()
		if err := writeNested(fs, path, FormatNote(Note{Title: title}, "")); err != nil {
			t.Fatal(err)
		}
	}
	mkNote("meetings/standup.md", "Standup")

	tl := NewListTree(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{"path": "meetings"})
	if err != nil {
		t.Fatalf("list_tree subpath: %v", err)
	}
	if out != "standup.md — Standup\n" {
		t.Fatalf("subpath tree = %q", out)
	}

	// Byte cap: a deeply nested or huge tree truncates with the marker.
	body := strings.Repeat("x", 500)
	if err := fs.WriteFile("big.md", FormatNote(Note{Title: strings.Repeat("t", 200)}, body), 0o644); err != nil {
		t.Fatal(err)
	}
	tiny := NewListTree(fs, 16)
	out, err = tiny.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("capped list_tree: %v", err)
	}
	if len(out) > 16+len(truncationMarker(16)) || !strings.Contains(out, "truncated") {
		t.Fatalf("capped tree not truncated: len=%d %q", len(out), out)
	}
}

func TestGrepTool(t *testing.T) {
	fs := newTestFS(t)
	mkNote := func(path, body string) {
		t.Helper()
		if err := writeNested(fs, path, FormatNote(Note{Title: path}, body)); err != nil {
			t.Fatal(err)
		}
	}
	mkNote("meetings/standup.md", "shipped the native agent\nmoved on")
	mkNote("knowledge/ai.md", "no mention here")
	mkNote("projects/agent.md", "the agent rocks")

	tl := NewGrep(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{"pattern": "agent"})
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if !strings.Contains(out, "meetings/standup.md:1:shipped the native agent") {
		t.Fatalf("match missing: %q", out)
	}
	if !strings.Contains(out, "projects/agent.md:1:the agent rocks") {
		t.Fatalf("match missing: %q", out)
	}
	if strings.Contains(out, "knowledge/ai.md") {
		t.Fatalf("non-matching note included: %q", out)
	}

	// Path restriction.
	out, err = tl.Run(context.Background(), map[string]any{"pattern": "agent", "path": "projects"})
	if err != nil {
		t.Fatalf("grep restricted: %v", err)
	}
	if strings.Contains(out, "standup") {
		t.Fatalf("restricted grep escaped its path: %q", out)
	}

	// Invalid regexp errors.
	if _, err := tl.Run(context.Background(), map[string]any{"pattern": "["}); err == nil {
		t.Fatal("invalid pattern succeeded")
	}
}

func TestGrepToolCapsMatches(t *testing.T) {
	fs := newTestFS(t)
	for i := 0; i < 30; i++ {
		path := "note" + string(rune('a'+i%26)) + string(rune('a'+(i/26))) + ".md"
		if err := fs.WriteFile(path, FormatNote(Note{Title: path}, "needle\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tl := NewGrep(fs, 10)
	out, err := tl.Run(context.Background(), map[string]any{"pattern": "needle"})
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if strings.Count(out, "\n") > 10 {
		t.Fatalf("more than 10 matches returned: %q", out)
	}
	if !strings.Contains(out, "grep truncated") {
		t.Fatalf("truncation marker missing: %q", out)
	}
}

func TestListRecentTool(t *testing.T) {
	fs := newTestFS(t)
	mkNote := func(path string) {
		t.Helper()
		if err := fs.WriteFile(path, FormatNote(Note{Title: "T" + path}, ""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkNote("old.md")
	time.Sleep(10 * time.Millisecond)
	mkNote("newer.md")
	time.Sleep(10 * time.Millisecond)
	mkNote("newest.md")

	tl := NewListRecent(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("list_recent: %v", err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 notes, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "newest.md") || !strings.Contains(lines[1], "newer.md") || !strings.Contains(lines[2], "old.md") {
		t.Fatalf("not newest-first: %q", out)
	}

	limited, err := tl.Run(context.Background(), map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("list_recent limited: %v", err)
	}
	if strings.Count(limited, "\n") > 1 {
		t.Fatalf("limit ignored: %q", limited)
	}
}

func TestSearchByTagTool(t *testing.T) {
	fs := newTestFS(t)
	mkTagged := func(path string, tags []string) {
		t.Helper()
		note := Note{Title: path, Tags: tags}
		if err := writeNested(fs, path, FormatNote(note, "")); err != nil {
			t.Fatal(err)
		}
	}
	mkTagged("meetings/a.md", []string{"work", "sync"})
	mkTagged("meetings/b.md", []string{"sync"})
	mkTagged("knowledge/c.md", []string{"personal"})

	tl := NewSearchByTag(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{"tag": "sync"})
	if err != nil {
		t.Fatalf("search_by_tag: %v", err)
	}
	if !strings.Contains(out, "meetings/a.md") || !strings.Contains(out, "meetings/b.md") {
		t.Fatalf("matching notes missing: %q", out)
	}
	if strings.Contains(out, "knowledge/c.md") {
		t.Fatalf("non-matching note included: %q", out)
	}

	limited, err := tl.Run(context.Background(), map[string]any{"tag": "sync", "limit": 1})
	if err != nil {
		t.Fatalf("limited search_by_tag: %v", err)
	}
	if strings.Count(limited, "\n") > 0 {
		t.Fatalf("limit ignored: %q", limited)
	}

	if _, err := tl.Run(context.Background(), map[string]any{}); err == nil {
		t.Fatal("missing tag succeeded")
	}
}

// writeNested writes content at rel, creating parent directories as needed
// (FS.WriteFile requires its parent to exist, mirroring atomic writes).
func writeNested(fs FS, rel string, content []byte) error {
	if dir := filepath.Dir(rel); dir != "." {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return fs.WriteFile(rel, content, 0o644)
}
