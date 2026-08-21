package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestListTreeTool(t *testing.T) {
	fs := newTestFS(t)
	writeNote(t, fs, "meetings/standup.md", wiki.NoteMeta{Title: "Standup", Date: "2026-08-21"}, "")
	writeNote(t, fs, "meetings/planning.md", wiki.NoteMeta{Title: "Planning"}, "")
	writeNote(t, fs, "knowledge/ai.md", wiki.NoteMeta{Title: "AI Notes"}, "")
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

	tl := NewListTree(fs, 0, 0)
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
	writeNote(t, fs, "meetings/standup.md", wiki.NoteMeta{Title: "Standup"}, "")

	tl := NewListTree(fs, 0, 0)
	out, err := tl.Run(context.Background(), map[string]any{"path": "meetings"})
	if err != nil {
		t.Fatalf("list_tree subpath: %v", err)
	}
	if out != "standup.md — Standup\n" {
		t.Fatalf("subpath tree = %q", out)
	}

	body := strings.Repeat("x", 500)
	writeNote(t, fs, "big.md", wiki.NoteMeta{Title: strings.Repeat("t", 200)}, body)
	tiny := NewListTree(fs, 16, 0)
	out, err = tiny.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("capped list_tree: %v", err)
	}
	if len(out) > 16+len(agenttools.TruncationMarker(16)) || !strings.Contains(out, "truncated") {
		t.Fatalf("capped tree not truncated: len=%d %q", len(out), out)
	}
}

func TestListRecentTool(t *testing.T) {
	fs := newTestFS(t)
	writeNote(t, fs, "old.md", wiki.NoteMeta{Title: "TOld"}, "")
	time.Sleep(10 * time.Millisecond)
	writeNote(t, fs, "newer.md", wiki.NoteMeta{Title: "TNewer"}, "")
	time.Sleep(10 * time.Millisecond)
	writeNote(t, fs, "newest.md", wiki.NoteMeta{Title: "TNewest"}, "")

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
	writeNote(t, fs, "meetings/a.md", wiki.NoteMeta{Title: "A", Tags: []string{"work", "sync"}}, "")
	writeNote(t, fs, "meetings/b.md", wiki.NoteMeta{Title: "B", Tags: []string{"sync"}}, "")
	writeNote(t, fs, "knowledge/c.md", wiki.NoteMeta{Title: "C", Tags: []string{"personal"}}, "")

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
