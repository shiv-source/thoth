package wiki

import (
	"os"
	"path/filepath"
	"sync"
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
	// Attachments are indexed by filename but hidden from the tree; an
	// uppercase-extension note is a note.
	if err := os.MkdirAll(filepath.Join(dir, "attachments"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "attachments", "logo.png"), []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "projects", "thoth", "README.MD"), []byte("---\ntitle: Readme\n---\nBody"), 0o644); err != nil {
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
	if len(projects.Children[0].Children) != 2 {
		t.Fatalf("expected thoth dir with project.md and README.MD, got %+v", projects.Children[0].Children)
	}
	// The root rulebook is not a note — hidden from the tree. A nested
	// CLAUDE.md still shows (only the root-level one is excluded).
	for i := range tree {
		if tree[i].Name == "CLAUDE.md" {
			t.Fatalf("root CLAUDE.md must be hidden from the tree: %+v", tree[i])
		}
	}
	// The reserved attachments directory is hidden from the tree too.
	for i := range tree {
		if tree[i].Name == "attachments" {
			t.Fatalf("reserved attachments dir must be hidden from the tree: %+v", tree[i])
		}
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

func TestWikiTreeSkipsUnreadableSubdir(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	// A readable sibling note must still surface: the tree keeps the locked
	// folder as an error node instead of failing the whole walk.
	if err := os.WriteFile(filepath.Join(dir, "open.md"), []byte("---\ntitle: Open\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}

	tree, err := New(dir).Tree()
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]*Node, len(tree))
	for i := range tree {
		names[tree[i].Name] = &tree[i]
	}
	if names["open.md"] == nil {
		t.Fatalf("readable sibling note missing from tree: %+v", names)
	}
	lockedNode := names["locked"]
	if lockedNode == nil || !lockedNode.IsDir {
		t.Fatalf("expected a locked dir node, got %+v", lockedNode)
	}
	if lockedNode.Error == "" {
		t.Fatal("expected the locked dir to carry an error")
	}
}

func TestWikiReadMissingNote(t *testing.T) {
	w := New(t.TempDir())
	if _, err := w.Read("meetings/nope.md"); err == nil {
		t.Fatal("expected error reading a missing note")
	}
}

func TestIndexable(t *testing.T) {
	tests := []struct {
		name string
		rel  string
		want bool
	}{
		{"note", "notes/a.md", true},
		{"attachment anywhere", "attachments/install.sh", true},
		{"image outside attachments", "images/logo.png", true},
		{"uppercase note", "NOTES/UPPER.MD", true},
		{"root rulebook", "CLAUDE.md", false},
		{"dotfile", ".gitkeep", false},
		{"hidden directory file", ".git/config", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Indexable(tt.rel); got != tt.want {
				t.Fatalf("Indexable(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}

func TestIsImagePath(t *testing.T) {
	tests := []struct {
		name string
		rel  string
		want bool
	}{
		{"png", "attachments/logo.png", true},
		{"jpg", "images/photo.jpg", true},
		{"jpeg", "images/photo.jpeg", true},
		{"gif", "attachments/anim.gif", true},
		{"svg", "images/logo.svg", true},
		{"webp", "images/logo.webp", true},
		{"uppercase", "IMAGES/LOGO.PNG", true},
		{"markdown note", "notes/a.md", false},
		{"script", "attachments/install.sh", false},
		{"config", "attachments/x.yaml", false},
		{"no extension", "attachments/README", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImagePath(tt.rel); got != tt.want {
				t.Fatalf("IsImagePath(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}

func TestVisible(t *testing.T) {
	tests := []struct {
		name  string
		rel   string
		isDir bool
		want  bool
	}{
		{"note", "notes/a.md", false, true},
		{"nested note", "projects/thoth/project.md", false, true},
		{"uppercase note", "NOTES/UPPER.MD", false, true},
		{"long markdown extension", "notes/note.markdown", false, true},
		{"non-markdown file", "images/logo.png", false, false},
		{"script attachment", "attachments/install.sh", false, false},
		{"reserved attachments dir", "attachments", true, false},
		{"attachment inside reserved dir", "attachments/x.yaml", false, false},
		{"nested folder named attachments", "projects/x/attachments", true, true},
		{"root rulebook", "CLAUDE.md", false, false},
		{"dotfile", ".gitkeep", false, false},
		{"hidden directory", ".git/config", false, false},
		{"hidden note inside visible dir", "notes/.draft.md", false, false},
		{"uppercase hidden note", "notes/.DRAFT.md", false, false},
		{"directory", "notes", true, true},
		{"hidden directory node", ".git", true, false},
		{"directory named like a note", "notes.md", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Visible(tt.rel, tt.isDir); got != tt.want {
				t.Fatalf("Visible(%q, %v) = %v, want %v", tt.rel, tt.isDir, got, tt.want)
			}
		})
	}
}

// TestWikiRootConcurrentSwapAndRead exercises the settings wiki-path change
// path: onSettingsSaved swaps the root from the settings HTTP handler
// goroutine while every turn reads it through Read. The bare field this
// replaces raced under `go test -race`; the guarded getter/setter must not.
func TestWikiRootConcurrentSwapAndRead(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for _, dir := range []string{a, b} {
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("---\ntitle: rb\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w := New(a)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_, _ = w.Read("CLAUDE.md") // reads the live root; must not race the swap
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			w.SetRoot(b)
			w.SetRoot(a)
		}
		close(stop)
	}()
	wg.Wait()
	if w.Root() != a {
		t.Fatalf("root = %q, want %q", w.Root(), a)
	}
}
