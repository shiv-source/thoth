package tools

import (
	"context"
	"testing"
)

func TestEditFileTool(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.WriteFile("note.md", []byte("alpha beta alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewEditFile(fs)
	ctx := context.Background()

	cases := []struct {
		name    string
		args    map[string]any
		wantErr bool
		wantOut string
	}{
		{
			name:    "single occurrence replaced",
			args:    map[string]any{"path": "note.md", "old_text": "beta", "new_text": "gamma"},
			wantOut: "edited note.md",
		},
		{
			name:    "missing old_text",
			args:    map[string]any{"path": "note.md", "old_text": "nope", "new_text": "x"},
			wantErr: true,
		},
		{
			name:    "ambiguous multiple occurrences",
			args:    map[string]any{"path": "note.md", "old_text": "alpha", "new_text": "x"},
			wantErr: true,
		},
		{
			name:    "empty old_text",
			args:    map[string]any{"path": "note.md", "old_text": "", "new_text": "x"},
			wantErr: true,
		},
		{
			name:    "missing file",
			args:    map[string]any{"path": "nope.md", "old_text": "a", "new_text": "b"},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Re-seed before each case so prior edits don't accumulate.
			if err := fs.WriteFile("note.md", []byte("alpha beta alpha"), 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := tl.Run(ctx, tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("edit_file succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("edit_file: %v", err)
			}
			if out != tc.wantOut {
				t.Fatalf("out = %q, want %q", out, tc.wantOut)
			}
			if tc.name == "single occurrence replaced" {
				data, _ := fs.ReadFile("note.md")
				if string(data) != "alpha gamma alpha" {
					t.Fatalf("content = %q, want %q", data, "alpha gamma alpha")
				}
			}
		})
	}
}

func TestAppendFileTool(t *testing.T) {
	fs := newTestFS(t)
	tl := NewAppendFile(fs)
	ctx := context.Background()

	out, err := tl.Run(ctx, map[string]any{"path": "log.md", "content": "line1\n"})
	if err != nil {
		t.Fatalf("append to missing file: %v", err)
	}
	if out != "appended to log.md" {
		t.Fatalf("out = %q", out)
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "log.md", "content": "line2\n"}); err != nil {
		t.Fatalf("append again: %v", err)
	}
	data, err := fs.ReadFile("log.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\nline2\n" {
		t.Fatalf("content = %q", data)
	}

	if _, err := tl.Run(ctx, map[string]any{"path": "nested/deep.md", "content": "deep"}); err != nil {
		t.Fatalf("append creating parents: %v", err)
	}
	data, err = fs.ReadFile("nested/deep.md")
	if err != nil || string(data) != "deep" {
		t.Fatalf("nested content = %q, %v", data, err)
	}
}

func TestRenameFileTool(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.WriteFile("old.md", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewRenameFile(fs)
	ctx := context.Background()

	out, err := tl.Run(ctx, map[string]any{"path": "old.md", "new_path": "new.md"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if out != "moved old.md to new.md" {
		t.Fatalf("out = %q", out)
	}
	if _, err := fs.Stat("old.md"); err == nil {
		t.Fatal("old path still exists")
	}
	if _, err := fs.Stat("new.md"); err != nil {
		t.Fatalf("new path missing: %v", err)
	}

	if _, err := tl.Run(ctx, map[string]any{"path": "../escape.md", "new_path": "x.md"}); err == nil {
		t.Fatal("escaping source succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "new.md", "new_path": "../escape.md"}); err == nil {
		t.Fatal("escaping destination succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "new.md", "new_path": "sub/deep/new.md"}); err != nil {
		t.Fatalf("rename creating destination parents: %v", err)
	}
}

func TestDeleteFileTool(t *testing.T) {
	fs := newTestFS(t)
	if err := fs.WriteFile("doomed.md", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := NewDeleteFile(fs)
	ctx := context.Background()

	out, err := tl.Run(ctx, map[string]any{"path": "doomed.md"})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if out != "deleted doomed.md" {
		t.Fatalf("out = %q", out)
	}
	if _, err := fs.Stat("doomed.md"); err == nil {
		t.Fatal("file still exists")
	}

	if _, err := tl.Run(ctx, map[string]any{"path": "doomed.md"}); err == nil {
		t.Fatal("deleting a missing file succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "../escape.md"}); err == nil {
		t.Fatal("deleting an escaping path succeeded")
	}
}
