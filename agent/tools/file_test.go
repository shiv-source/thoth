package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestFS returns an OSFS bound to a fresh temp directory.
func newTestFS(t *testing.T) *OSFS {
	t.Helper()
	fs, err := NewOSFS(t.TempDir())
	if err != nil {
		t.Fatalf("NewOSFS: %v", err)
	}
	return fs
}

func TestNewOSFSValidation(t *testing.T) {
	if _, err := NewOSFS(""); err == nil {
		t.Fatal("empty root accepted")
	}
	if _, err := NewOSFS(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("non-existent root accepted")
	}
}

func TestOSFSRejectsTraversal(t *testing.T) {
	fs := newTestFS(t)
	cases := []string{
		"..",
		"../..",
		"../x",
		"../../x",
		"a/../../b",
		"a/../..",
		"/etc/passwd",
		"/",
		"",
	}
	for _, p := range cases {
		t.Run("path_"+strings.NewReplacer("/", "_", ".", "d").Replace(p), func(t *testing.T) {
			if _, err := fs.ReadFile(p); err == nil {
				t.Fatalf("ReadFile(%q) succeeded, want error", p)
			}
			if _, err := fs.ReadDir(p); err == nil {
				t.Fatalf("ReadDir(%q) succeeded, want error", p)
			}
			if err := fs.WriteFile(p, []byte("x"), 0o644); err == nil {
				t.Fatalf("WriteFile(%q) succeeded, want error", p)
			}
			if err := fs.MkdirAll(p, 0o755); err == nil {
				t.Fatalf("MkdirAll(%q) succeeded, want error", p)
			}
		})
	}
}

func TestToolsRejectTraversal(t *testing.T) {
	fs := newTestFS(t)
	ctx := context.Background()
	if _, err := NewReadFile(fs, 0).Run(ctx, map[string]any{"path": "../x"}); err == nil {
		t.Fatal("read_file accepted ../x")
	}
	if _, err := NewWriteFile(fs).Run(ctx, map[string]any{"path": "/etc/x", "content": "x"}); err == nil {
		t.Fatal("write_file accepted an absolute path")
	}
	if _, err := NewList(fs).Run(ctx, map[string]any{"path": "a/../../b"}); err == nil {
		t.Fatal("list accepted an escaping path")
	}
}

func TestOSFSSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "secret.txt")); err != nil {
		t.Fatal(err)
	}
	fs, err := NewOSFS(root)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fs.ReadFile("escape/secret.txt"); err == nil {
		t.Fatal("read through escaping symlinked dir succeeded")
	}
	if _, err := fs.ReadFile("secret.txt"); err == nil {
		t.Fatal("read of a symlink pointing outside root succeeded")
	}
	if err := fs.WriteFile("escape/new.txt", []byte("x"), 0o644); err == nil {
		t.Fatal("write through escaping symlinked dir succeeded")
	}
	if _, err := fs.ReadDir("escape"); err == nil {
		t.Fatal("list of escaping symlink succeeded")
	}
}

func TestOSFSSymlinkInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real.txt"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	fs, err := NewOSFS(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fs.ReadFile("alias")
	if err != nil {
		t.Fatalf("read through in-root symlink: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("read = %q, want hello", got)
	}
}

func TestOSFSWriteReadList(t *testing.T) {
	root := t.TempDir()
	fs, err := NewOSFS(root)
	if err != nil {
		t.Fatal(err)
	}

	if err := fs.WriteFile("notes.md", []byte("# hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := fs.MkdirAll("a/b", 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := fs.WriteFile("a/b/deep.txt", []byte("deep"), 0o644); err != nil {
		t.Fatalf("WriteFile nested: %v", err)
	}

	data, err := fs.ReadFile("notes.md")
	if err != nil || string(data) != "# hi" {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
	data, err = fs.ReadFile("a/b/deep.txt")
	if err != nil || string(data) != "deep" {
		t.Fatalf("ReadFile nested = %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes.md")); err != nil {
		t.Fatalf("notes.md not persisted on disk: %v", err)
	}

	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir("empty")
	if err != nil || len(entries) != 0 {
		t.Fatalf("ReadDir empty = %v, %v", entries, err)
	}
}

func TestWriteFileToolCreatesParentsAndPersists(t *testing.T) {
	fs := newTestFS(t)
	tl := NewWriteFile(fs)
	got, err := tl.Run(context.Background(), map[string]any{
		"path":    "docs/notes/deep.md",
		"content": "body",
	})
	if err != nil {
		t.Fatalf("write_file: %v", err)
	}
	if got != "wrote docs/notes/deep.md" {
		t.Fatalf("write_file result = %q", got)
	}
	data, err := fs.ReadFile("docs/notes/deep.md")
	if err != nil || string(data) != "body" {
		t.Fatalf("read back = %q, %v", data, err)
	}
}

func TestWriteFileArgValidation(t *testing.T) {
	fs := newTestFS(t)
	tl := NewWriteFile(fs)
	ctx := context.Background()
	if _, err := tl.Run(ctx, map[string]any{"content": "x"}); err == nil {
		t.Fatal("missing path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "x"}); err == nil {
		t.Fatal("missing content succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": "x", "content": 42}); err == nil {
		t.Fatal("non-string content succeeded")
	}
}

func TestWriteFileFailedWriteLeavesNoPartialFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	fs, err := NewOSFS(root)
	if err != nil {
		t.Fatal(err)
	}
	// Renaming onto an existing directory fails after the temp file is fully
	// written; the temp file must be cleaned up and the target untouched.
	if err := fs.WriteFile("dir", []byte("x"), 0o644); err == nil {
		t.Fatal("write over a directory succeeded, want error")
	}
	if info, err := os.Stat(filepath.Join(root, "dir")); err != nil || !info.IsDir() {
		t.Fatalf("target directory damaged: %v %+v", err, info)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".thoth-write-") {
			t.Fatalf("leftover temp file %q", e.Name())
		}
	}
}

func TestReadFileTool(t *testing.T) {
	fs := newTestFS(t)
	big := strings.Repeat("x", 3000)
	if err := fs.WriteFile("big.txt", []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("small.txt", []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := NewReadFile(fs, 1024)
	ctx := context.Background()

	got, err := tl.Run(ctx, map[string]any{"path": "small.txt"})
	if err != nil || got != "tiny" {
		t.Fatalf("read small = %q, %v", got, err)
	}

	got, err = tl.Run(ctx, map[string]any{"path": "big.txt"})
	if err != nil {
		t.Fatalf("read big: %v", err)
	}
	want := strings.Repeat("x", 1024) + "\n\n[output truncated: file exceeds 1024 bytes]"
	if got != want {
		t.Fatalf("truncated read mismatch: got %d bytes, want %d", len(got), len(want))
	}

	if _, err := tl.Run(ctx, map[string]any{}); err == nil {
		t.Fatal("missing path succeeded")
	}
	if _, err := tl.Run(ctx, map[string]any{"path": 42}); err == nil {
		t.Fatal("non-string path succeeded")
	}

	def := NewReadFile(fs, 0)
	huge := strings.Repeat("x", maxReadBytes+1)
	if err := fs.WriteFile("huge.txt", []byte(huge), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = def.Run(ctx, map[string]any{"path": "huge.txt"})
	if err != nil {
		t.Fatalf("read huge default: %v", err)
	}
	if len(got) != maxReadBytes+len(truncationMarker(maxReadBytes)) {
		t.Fatalf("default-cap read length = %d, want %d", len(got), maxReadBytes+len(truncationMarker(maxReadBytes)))
	}
	if !strings.Contains(got, "[output truncated: file exceeds 131072 bytes]") {
		t.Fatal("default truncation marker missing")
	}

	if err := fs.MkdirAll("adir", 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := def.Run(ctx, map[string]any{"path": "adir"}); err == nil {
		t.Fatal("read of a directory succeeded")
	}
}

func TestListTool(t *testing.T) {
	fs := newTestFS(t)
	for _, f := range []string{"a.txt", "Z.md"} {
		if err := fs.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := fs.MkdirAll("docs/sub", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("docs/sub/x.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := NewList(fs)
	ctx := context.Background()

	got, err := tl.Run(ctx, map[string]any{"path": "."})
	if err != nil {
		t.Fatalf("list .: %v", err)
	}
	want := "Z.md\tfile\na.txt\tfile\ndocs\tdir"
	if got != want {
		t.Fatalf("list . = %q, want %q", got, want)
	}

	got, err = tl.Run(ctx, map[string]any{"path": "docs/sub"})
	if err != nil {
		t.Fatalf("list docs/sub: %v", err)
	}
	if got != "x.txt\tfile" {
		t.Fatalf("list docs/sub = %q", got)
	}

	got, err = tl.Run(ctx, map[string]any{})
	if err != nil || got != want {
		t.Fatalf("list default = %q, %v", got, err)
	}

	if err := fs.MkdirAll("empty", 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = tl.Run(ctx, map[string]any{"path": "empty"})
	if err != nil || got != "" {
		t.Fatalf("list empty = %q, %v", got, err)
	}

	if _, err := tl.Run(ctx, map[string]any{"path": "nope"}); err == nil {
		t.Fatal("list of missing dir succeeded")
	}
}
