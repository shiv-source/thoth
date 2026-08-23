package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestGetTodosTool(t *testing.T) {
	fs := newTestFS(t)
	tl := NewGetTodos(fs, "")

	out, err := tl.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("get_todos: %v", err)
	}
	if !strings.Contains(out, "no TODO list") {
		t.Fatalf("missing-file message = %q", out)
	}

	update := NewUpdateTodos(fs, "")
	if _, err := update.Run(context.Background(), map[string]any{"content": "## Now\n- ship it"}); err != nil {
		t.Fatalf("update_todos: %v", err)
	}
	out, err = tl.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("get_todos after update: %v", err)
	}
	if out != "## Now\n- ship it" {
		t.Fatalf("todos = %q", out)
	}
}

func TestUpdateTodosToolCreatesParents(t *testing.T) {
	fs := newTestFS(t)
	custom := NewUpdateTodos(fs, "custom/tasks.md")
	if _, err := custom.Run(context.Background(), map[string]any{"content": "x"}); err != nil {
		t.Fatalf("update_todos custom path: %v", err)
	}
	if _, err := fs.Stat("custom/tasks.md"); err != nil {
		t.Fatalf("custom todos file missing: %v", err)
	}
}

func TestGetInboxTool(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.MkdirAll("inbox", 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.md", "b.md"} {
		if err := fs.WriteFile("inbox/"+f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tl := NewGetInbox(fs, "")
	out, err := tl.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("get_inbox: %v", err)
	}
	if out != "a.md\nb.md" {
		t.Fatalf("inbox = %q", out)
	}

	empty := NewGetInbox(newTestFS(t), "")
	out, err = empty.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("get_inbox missing: %v", err)
	}
	if !strings.Contains(out, "no inbox") {
		t.Fatalf("missing-inbox message = %q", out)
	}
}

func TestFileInboxTool(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.MkdirAll("inbox", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("inbox/capture.md", []byte("---\ntitle: Capture\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewFileInbox(fs, "")
	out, err := tl.Run(context.Background(), map[string]any{"path": "capture.md", "dest": "projects/alpha"})
	if err != nil {
		t.Fatalf("file_inbox: %v", err)
	}
	if out != "filed inbox/capture.md to projects/alpha/capture.md" {
		t.Fatalf("out = %q", out)
	}
	if _, err := fs.Stat("inbox/capture.md"); err == nil {
		t.Fatal("inbox entry still present")
	}
	if _, err := fs.Stat("projects/alpha/capture.md"); err != nil {
		t.Fatalf("filed entry missing: %v", err)
	}

	if _, err := tl.Run(context.Background(), map[string]any{"path": "a/b.md", "dest": "x"}); err == nil {
		t.Fatal("sub-path entry succeeded")
	}
	if _, err := tl.Run(context.Background(), map[string]any{"path": "../escape.md", "dest": "x"}); err == nil {
		t.Fatal("escaping entry succeeded")
	}
	if _, err := tl.Run(context.Background(), map[string]any{"path": "capture.md", "dest": "../escape"}); err == nil {
		t.Fatal("escaping destination succeeded")
	}
}

func TestRememberTool(t *testing.T) {
	fs := newTestFS(t)
	fixed := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	tl := NewRemember(fs, "", func() time.Time { return fixed })

	out, err := tl.Run(context.Background(), map[string]any{"fact": "the wiki is fast"})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}
	if out != "remembered to knowledge/memory.md" {
		t.Fatalf("out = %q", out)
	}

	data, err := fs.ReadFile("knowledge/memory.md")
	if err != nil {
		t.Fatalf("memory note missing: %v", err)
	}
	meta, body, err := wiki.ParseNote(data)
	if err != nil {
		t.Fatalf("memory note does not parse: %v\n%s", err, data)
	}
	if meta.Title != "Memory" || meta.Date != "2026-08-21" {
		t.Fatalf("memory note = %+v", meta)
	}
	if !strings.Contains(string(body), "- 2026-08-21T09:00:00Z the wiki is fast") {
		t.Fatalf("fact line missing: %q", body)
	}

	if _, err := tl.Run(context.Background(), map[string]any{"fact": "and kind"}); err != nil {
		t.Fatalf("remember again: %v", err)
	}
	data, err = fs.ReadFile("knowledge/memory.md")
	if err != nil {
		t.Fatal(err)
	}
	_, body, err = wiki.ParseNote(data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if strings.Count(string(body), "\n- ") != 2 {
		t.Fatalf("expected two fact lines, got %q", body)
	}
}

func TestOpsToolsCtxCancelled(t *testing.T) {
	fs := newTestFS(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewGetTodos(fs, "").Run(ctx, map[string]any{}); err == nil {
		t.Fatal("get_todos on cancelled ctx succeeded")
	}
	if _, err := NewUpdateTodos(fs, "").Run(ctx, map[string]any{"content": "x"}); err == nil {
		t.Fatal("update_todos on cancelled ctx succeeded")
	}
	if _, err := NewGetInbox(fs, "").Run(ctx, map[string]any{}); err == nil {
		t.Fatal("get_inbox on cancelled ctx succeeded")
	}
	if _, err := NewRemember(fs, "", nil).Run(ctx, map[string]any{"fact": "x"}); err == nil {
		t.Fatal("remember on cancelled ctx succeeded")
	}
}

func TestOpsToolsFSErrors(t *testing.T) {
	ctx := context.Background()
	base := newTestFS(t)
	if err := base.MkdirAll("todos", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := base.WriteFile("todos/TODO.md", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		fail  string
		build func(agenttools.FS) agenttools.Tool
		args  map[string]any
	}{
		{"get_todos read error", "ReadFile", func(f agenttools.FS) agenttools.Tool { return NewGetTodos(f, "") }, map[string]any{}},
		{"update_todos missing content", "", func(f agenttools.FS) agenttools.Tool { return NewUpdateTodos(f, "") }, map[string]any{}},
		{"update_todos non-string content", "", func(f agenttools.FS) agenttools.Tool { return NewUpdateTodos(f, "") }, map[string]any{"content": 7}},
		{"update_todos mkdir error", "MkdirAll", func(f agenttools.FS) agenttools.Tool { return NewUpdateTodos(f, "sub/todo.md") }, map[string]any{"content": "x"}},
		{"update_todos write error", "WriteFile", func(f agenttools.FS) agenttools.Tool { return NewUpdateTodos(f, "") }, map[string]any{"content": "x"}},
		{"get_inbox read error", "ReadDir", func(f agenttools.FS) agenttools.Tool { return NewGetInbox(f, "") }, map[string]any{}},
		{"file_inbox missing path", "", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"dest": "x"}},
		{"file_inbox missing dest", "", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"path": "a.md"}},
		{"file_inbox non-string path", "", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"path": 5, "dest": "x"}},
		{"file_inbox escaping path", "", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"path": "../a.md", "dest": "x"}},
		{"file_inbox escaping dest", "", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"path": "a.md", "dest": "../x"}},
		{"file_inbox mkdir error", "MkdirAll", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"path": "a.md", "dest": "sub"}},
		{"file_inbox rename error", "Rename", func(f agenttools.FS) agenttools.Tool { return NewFileInbox(f, "") }, map[string]any{"path": "a.md", "dest": "x"}},
		{"remember missing fact", "", func(f agenttools.FS) agenttools.Tool { return NewRemember(f, "", nil) }, map[string]any{}},
		{"remember non-string fact", "", func(f agenttools.FS) agenttools.Tool { return NewRemember(f, "", nil) }, map[string]any{"fact": 5}},
		{"remember read error", "ReadFile", func(f agenttools.FS) agenttools.Tool { return NewRemember(f, "", nil) }, map[string]any{"fact": "x"}},
		{"remember mkdir error", "MkdirAll", func(f agenttools.FS) agenttools.Tool { return NewRemember(f, "sub/mem.md", nil) }, map[string]any{"fact": "x"}},
		{"remember write error", "WriteFile", func(f agenttools.FS) agenttools.Tool { return NewRemember(f, "", nil) }, map[string]any{"fact": "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var fs agenttools.FS = base
			if tc.fail != "" {
				fs = failingFS{FS: base, fail: tc.fail}
			}
			if _, err := tc.build(fs).Run(ctx, tc.args); err == nil {
				t.Fatalf("expected an error, got success")
			}
		})
	}
}
