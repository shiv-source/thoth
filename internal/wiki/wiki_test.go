package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWikiReadAndTree(t *testing.T) {
	dir := t.TempDir()
	if err := Scaffold(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "projects", "thoth"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "projects", "thoth", "project.md"), []byte("---\ntitle: Thoth\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := New(dir)
	if !w.Exists() {
		t.Fatal("Exists() must be true")
	}
	b, err := w.Read("projects/thoth/project.md")
	if err != nil || string(b) != "---\ntitle: Thoth\n---\nBody" {
		t.Fatalf("Read: %v %q", err, b)
	}
	if _, err := w.Read("../escape.md"); err == nil {
		t.Fatal("Read must reject escaping paths")
	}

	tree, err := w.Tree()
	if err != nil {
		t.Fatal(err)
	}
	var projects *Node
	for i := range tree {
		if tree[i].Name == "projects" {
			projects = &tree[i]
		}
	}
	if projects == nil || !projects.IsDir || len(projects.Children) != 1 {
		t.Fatalf("expected projects dir with one child, got %+v", projects)
	}
}

func TestWikiNotExists(t *testing.T) {
	if New(filepath.Join(t.TempDir(), "missing")).Exists() {
		t.Fatal("Exists() must be false for missing dir")
	}
}

func TestWikiTreeErrorOnMissingRoot(t *testing.T) {
	w := New(filepath.Join(t.TempDir(), "missing"))
	if _, err := w.Tree(); err == nil {
		t.Fatal("expected error walking a missing root")
	}
}

func TestWikiTreeErrorOnUnreadableSubdir(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	// tree() recurses into locked and its readdir fails: the recursion must
	// surface the error instead of swallowing it.
	if _, err := New(dir).Tree(); err == nil {
		t.Fatal("expected error reading an unreadable subdir")
	}
}

func TestWikiReadMissingNote(t *testing.T) {
	w := New(t.TempDir())
	if _, err := w.Read("meetings/nope.md"); err == nil {
		t.Fatal("expected error reading a missing note")
	}
}
