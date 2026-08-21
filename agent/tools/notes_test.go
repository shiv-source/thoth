package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWriteNoteTool(t *testing.T) {
	fs := newTestFS(t)
	fixed := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	tl := NewWriteNote(fs, func() time.Time { return fixed })

	cases := []struct {
		name    string
		args    map[string]any
		wantOut string
		check   func(t *testing.T, content string)
	}{
		{
			name: "scaffolds frontmatter with folder type",
			args: map[string]any{
				"path":  "meetings/2026-08-21-standup.md",
				"title": "Standup",
				"body":  "Shipped the catalog.",
				"tags":  []any{"work", "sync"},
			},
			wantOut: "wrote note meetings/2026-08-21-standup.md",
			check: func(t *testing.T, content string) {
				want := "---\ntitle: Standup\ndate: 2026-08-21\ntype: meeting\ntags: [work, sync]\n---\nShipped the catalog.\n"
				if content != want {
					t.Fatalf("content:\ngot  %q\nwant %q", content, want)
				}
			},
		},
		{
			name: "explicit type and no tags",
			args: map[string]any{
				"path":  "knowledge/ai.md",
				"title": "AI",
				"body":  "body",
				"type":  "knowledge",
			},
			wantOut: "wrote note knowledge/ai.md",
			check: func(t *testing.T, content string) {
				want := "---\ntitle: AI\ndate: 2026-08-21\ntype: knowledge\n---\nbody\n"
				if content != want {
					t.Fatalf("content:\ngot  %q\nwant %q", content, want)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tl.Run(context.Background(), tc.args)
			if err != nil {
				t.Fatalf("write_note: %v", err)
			}
			if out != tc.wantOut {
				t.Fatalf("out = %q, want %q", out, tc.wantOut)
			}
			path := tc.args["path"].(string)
			data, err := fs.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			tc.check(t, string(data))
		})
	}
}

func TestWriteNoteToolRejectsInvalidInput(t *testing.T) {
	fs := newTestFS(t)
	tl := NewWriteNote(fs, nil)
	ctx := context.Background()

	if _, err := tl.Run(ctx, map[string]any{"path": "x.md", "title": "", "body": "b"}); err == nil {
		t.Fatal("empty title succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "   ", "title": "T", "body": "b"}); err == nil {
		t.Fatal("blank title succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "note.txt", "title": "T", "body": "b"}); err == nil {
		t.Fatal("extension-less path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "note", "title": "T", "body": "b"}); err == nil {
		t.Fatal("no-extension path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "../escape.md", "title": "T", "body": "b"}); err == nil {
		t.Fatal("escaping path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"title": "T", "body": "b"}); err == nil {
		t.Fatal("missing path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "x.md", "body": "b"}); err == nil {
		t.Fatal("missing title succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "x.md", "title": "T"}); err == nil {
		t.Fatal("missing body succeeded")
	}
}

func TestReadNoteTool(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.MkdirAll("meetings", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("meetings/standup.md", []byte(
		"---\ntitle: Standup\ndate: 2026-08-21\ntype: meeting\ntags: [work]\n---\nTalked about the catalog.",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewReadNote(fs, 0)
	out, err := tl.Run(context.Background(), map[string]any{"path": "meetings/standup.md"})
	if err != nil {
		t.Fatalf("read_note: %v", err)
	}
	want := "title: Standup\ndate: 2026-08-21\ntype: meeting\ntags: [work]\n---\nTalked about the catalog.\n"
	if out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
}

func TestReadNoteToolErrors(t *testing.T) {
	fs := newTestFS(t)
	tl := NewReadNote(fs, 0)
	ctx := context.Background()
	if _, err := tl.Run(ctx, map[string]any{"path": "missing.md"}); err == nil {
		t.Fatal("read of a missing note succeeded")
	}
	if err := fs.WriteFile("not-note.md", []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "not-note.md"}); err == nil {
		t.Fatal("read of a note without frontmatter succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{}); err == nil {
		t.Fatal("missing path succeeded")
	}
}

func TestReadNoteCapsOutput(t *testing.T) {
	fs := newTestFS(t)
	body := strings.Repeat("x", 500)
	if err := fs.WriteFile("big.md", []byte("---\ntitle: Big\n---\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewReadNote(fs, 128)
	out, err := tl.Run(context.Background(), map[string]any{"path": "big.md"})
	if err != nil {
		t.Fatalf("read_note: %v", err)
	}
	if len(out) != 128+len(truncationMarker(128)) {
		t.Fatalf("out length = %d, want %d", len(out), 128+len(truncationMarker(128)))
	}
	if !strings.Contains(out, "[output truncated: file exceeds 128 bytes]") {
		t.Fatalf("truncation marker missing: %q", out)
	}
}
