package tools

import (
	"context"
	"strings"
	"testing"
)

func TestGrepTool(t *testing.T) {
	fs := newTestFS(t)
	if err := writeNested(fs, "meetings/standup.md", []byte("shipped the native agent\nmoved on")); err != nil {
		t.Fatal(err)
	}
	if err := writeNested(fs, "knowledge/ai.md", []byte("no mention here")); err != nil {
		t.Fatal(err)
	}
	if err := writeNested(fs, "projects/agent.md", []byte("the agent rocks")); err != nil {
		t.Fatal(err)
	}

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
		t.Fatalf("non-matching file included: %q", out)
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
		path := "note" + string(rune('a'+i%26)) + string(rune('a'+(i/26))) + ".txt"
		if err := fs.WriteFile(path, []byte("needle\n"), 0o644); err != nil {
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

func TestGrepSkipsHidden(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.WriteFile(".hidden.txt", []byte("needle"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewGrep(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{"pattern": "needle"})
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if out != "" {
		t.Fatalf("hidden file matched: %q", out)
	}
}
