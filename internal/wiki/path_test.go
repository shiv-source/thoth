package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafePath(t *testing.T) {
	root := t.TempDir()
	if got, err := SafePath(root, "meetings/a.md"); err != nil || got == "" {
		t.Fatalf("SafePath: %v %q", err, got)
	}
	for _, bad := range []string{"../x.md", "a/../../x.md", "/etc/passwd"} {
		if _, err := SafePath(root, bad); err == nil {
			t.Fatalf("expected rejection of %q", bad)
		}
	}
}

func TestSafePathRejectsSymlinkEscape(t *testing.T) {
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
	for _, rel := range []string{"escape/secret.txt", "secret.txt", "escape", "escape"} {
		if _, err := SafePath(root, rel); err == nil {
			t.Fatalf("SafePath(%q) must reject a symlink escaping the root", rel)
		}
	}
	// A write target that does not exist yet resolves through its deepest
	// existing ancestor, so the escaping parent directory is still caught.
	if _, err := SafePath(root, "escape/new.txt"); err == nil {
		t.Fatal("SafePath must resolve the deepest existing ancestor of a new file")
	}
}

func TestSafePathAllowsSymlinkInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real.txt"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := SafePath(root, "alias"); err != nil {
		t.Fatalf("SafePath must allow a symlink inside the root: %v", err)
	}
}

func TestSafePathWithMissingRoot(t *testing.T) {
	if _, err := SafePath(filepath.Join(t.TempDir(), "missing"), "x.md"); err == nil {
		t.Fatal("SafePath with a missing root must error")
	}
}
