package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestWikiToolsBoundToSafePath(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(wiki.New(dir), nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../x.md", "a/../../x.md", "/etc/passwd"} {
		if _, err := read.Run(ctx, map[string]any{"path": bad}); err == nil {
			t.Fatalf("read_file(%q) should be rejected", bad)
		}
	}

	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../x.md", "/tmp/evil.md"} {
		if _, err := write.Run(ctx, map[string]any{"path": bad, "content": "x"}); err == nil {
			t.Fatalf("write_file(%q) should be rejected", bad)
		}
	}

	list, err := reg.Get("list")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := list.Run(ctx, map[string]any{"path": ".."}); err == nil {
		t.Fatal("list(..) should be rejected")
	}
}

func TestWriteFileProducesParseableNote(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(wiki.New(dir), nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: A Note\ntype: meeting\n---\nbody text"
	if _, err := write.Run(ctx, map[string]any{"path": "meetings/2026-08-21-x.md", "content": content}); err != nil {
		t.Fatalf("write_file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "meetings", "2026-08-21-x.md"))
	if err != nil {
		t.Fatalf("read back note: %v", err)
	}
	meta, body, err := wiki.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Title != "A Note" || meta.Kind != "meeting" {
		t.Fatalf("meta = %+v", meta)
	}
	if string(body) != "body text" {
		t.Fatalf("body = %q", body)
	}

	list, err := reg.Get("list")
	if err != nil {
		t.Fatal(err)
	}
	out, err := list.Run(ctx, map[string]any{"path": "meetings"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "2026-08-21-x.md\tfile") {
		t.Fatalf("list output = %q", out)
	}

	// read_file reads the note back through the tool; a missing file and a
	// missing directory surface as errors.
	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	got, err := read.Run(ctx, map[string]any{"path": "meetings/2026-08-21-x.md"})
	if err != nil {
		t.Fatalf("read_file: %v", err)
	}
	if !strings.Contains(got, "body text") {
		t.Fatalf("read_file output = %q", got)
	}
	if _, err := read.Run(ctx, map[string]any{"path": "nope.md"}); err == nil {
		t.Fatal("read_file of a missing file should error")
	}
	if _, err := list.Run(ctx, map[string]any{"path": "nope"}); err == nil {
		t.Fatal("list of a missing directory should error")
	}
}

func TestWikiToolsRejectSymlinkEscape(t *testing.T) {
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
	reg, err := registry(wiki.New(root), nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"escape/secret.txt", "secret.txt"} {
		if _, err := read.Run(ctx, map[string]any{"path": p}); err == nil {
			t.Fatalf("read_file(%q) must reject a symlink escaping the root", p)
		}
	}

	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.Run(ctx, map[string]any{"path": "escape/new.txt", "content": "x"}); err == nil {
		t.Fatal("write_file through an escaping symlinked dir succeeded")
	}

	list, err := reg.Get("list")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := list.Run(ctx, map[string]any{"path": "escape"}); err == nil {
		t.Fatal("list of an escaping symlink succeeded")
	}
}

func TestWikiWriteFileAtomicLeavesNoPartialFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg, err := registry(wiki.New(root), nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	// Renaming onto an existing directory fails after the temp file is fully
	// written; the temp file must be cleaned up and the target untouched.
	if _, err := write.Run(context.Background(), map[string]any{"path": "dir", "content": "x"}); err == nil {
		t.Fatal("write over a directory succeeded, want error")
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

func TestWikiToolsFollowRootSwap(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	w := wiki.New(rootA)
	reg, err := registry(w, nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()
	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.Run(ctx, map[string]any{"path": "note.md", "content": "a"}); err != nil {
		t.Fatalf("write before swap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootA, "note.md")); err != nil {
		t.Fatalf("old-root note missing: %v", err)
	}

	// Swap the live root (onSettingsSaved): the same registry must now serve
	// the new root, so system prompt and tools agree.
	w.SetRoot(rootB)
	if _, err := write.Run(ctx, map[string]any{"path": "note.md", "content": "b"}); err != nil {
		t.Fatalf("write after swap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootB, "note.md")); err != nil {
		t.Fatalf("new-root note missing: %v", err)
	}

	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	got, err := read.Run(ctx, map[string]any{"path": "note.md"})
	if err != nil {
		t.Fatalf("read after swap: %v", err)
	}
	if got != "b" {
		t.Fatalf("read after swap = %q, want %q (tools must follow the new root)", got, "b")
	}
}

func TestWikiFSErrorPaths(t *testing.T) {
	root := t.TempDir()
	fsys := wikiFS{wiki: wiki.New(root)}

	// SafePath rejection (defense in depth: the tools already cleanRel, but
	// the FS seam must reject escapes on its own too).
	if err := fsys.WriteFile("../escape.md", []byte("x"), 0o644); err == nil {
		t.Fatal("WriteFile with an escaping path succeeded")
	}
	if err := fsys.MkdirAll("../escape", 0o755); err == nil {
		t.Fatal("MkdirAll with an escaping path succeeded")
	}

	// A file squatting on a directory name makes MkdirAll fail.
	if err := os.WriteFile(filepath.Join(root, "block"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsys.MkdirAll("block/sub", 0o755); err == nil {
		t.Fatal("MkdirAll under a file succeeded")
	}

	// Listing a missing directory surfaces the error.
	if _, err := fsys.ReadDir("nope"); err == nil {
		t.Fatal("ReadDir of a missing directory succeeded")
	}
}

func TestIndexSearchTool(t *testing.T) {
	ix := openIndex(t)
	n := index.Note{
		Path: "meetings/standup.md", Title: "Standup", Kind: "meeting",
		Body: "Decided to ship the native agent.", UpdatedAt: time.Now(),
	}
	if err := ix.Upsert(n); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	reg, err := registry(wiki.New(t.TempDir()), ix)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	search, err := reg.Get("search")
	if err != nil {
		t.Fatal(err)
	}
	out, err := search.Run(context.Background(), map[string]any{"query": "native agent"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(out, "meetings/standup.md") {
		t.Fatalf("search output = %q, want the note path", out)
	}
}
