package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/wiki"
)

// newTestFS returns an OSFS bound to a fresh temp directory.
func newTestFS(t *testing.T) *agenttools.OSFS {
	t.Helper()
	fs, err := agenttools.NewOSFS(t.TempDir())
	if err != nil {
		t.Fatalf("NewOSFS: %v", err)
	}
	return fs
}

// writeNote writes a note with parent directories, using the wiki's FormatNote
// so tests exercise the real note contract.
func writeNote(t *testing.T, fs agenttools.FS, rel string, meta wiki.NoteMeta, body string) {
	t.Helper()
	if dir := filepath.Dir(rel); dir != "." {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := fs.WriteFile(rel, wiki.FormatNote(meta, body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteNoteTool(t *testing.T) {
	fs := newTestFS(t)
	fixed := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	tl := NewWriteNote(fs, func() time.Time { return fixed })

	cases := []struct {
		name    string
		args    map[string]any
		wantOut string
		check   func(t *testing.T, meta wiki.NoteMeta, body string)
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
			check: func(t *testing.T, meta wiki.NoteMeta, body string) {
				if meta.Title != "Standup" || meta.Date != "2026-08-21" || meta.Kind != "meeting" {
					t.Fatalf("meta = %+v", meta)
				}
				if strings.Join(meta.Tags, ",") != "work,sync" {
					t.Fatalf("tags = %v", meta.Tags)
				}
				if body != "Shipped the catalog.\n" {
					t.Fatalf("body = %q", body)
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
			check: func(t *testing.T, meta wiki.NoteMeta, body string) {
				if meta.Kind != "knowledge" || len(meta.Tags) != 0 {
					t.Fatalf("meta = %+v", meta)
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
			meta, body, err := wiki.ParseNote(data)
			if err != nil {
				t.Fatalf("note does not parse in the wiki: %v\n%s", err, data)
			}
			tc.check(t, meta, string(body))
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
		t.Fatal("non-markdown path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "note", "title": "T", "body": "b"}); err == nil {
		t.Fatal("extension-less path succeeded")
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
	writeNote(t, fs, "meetings/standup.md", wiki.NoteMeta{
		Title: "Standup", Date: "2026-08-21", Kind: "meeting", Tags: []string{"work"},
	}, "Talked about the catalog.")
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
	writeNote(t, fs, "big.md", wiki.NoteMeta{Title: "Big"}, body)
	tl := NewReadNote(fs, 128)
	out, err := tl.Run(context.Background(), map[string]any{"path": "big.md"})
	if err != nil {
		t.Fatalf("read_note: %v", err)
	}
	if len(out) != 128+len(agenttools.TruncationMarker(128)) {
		t.Fatalf("out length = %d, want %d", len(out), 128+len(agenttools.TruncationMarker(128)))
	}
	if !strings.Contains(out, "[output truncated: file exceeds 128 bytes]") {
		t.Fatalf("truncation marker missing: %q", out)
	}
}
