package gitutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesRepository(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected .git to exist: %v", err)
	}
	// Idempotent: a second init is a no-op, not an error.
	if err := Init(dir); err != nil {
		t.Fatalf("second Init: %v", err)
	}
}

func TestInitErrorsWithoutGit(t *testing.T) {
	// A PATH without git forces the failure path (fixed sanitized message).
	t.Setenv("PATH", t.TempDir())
	if err := Init(t.TempDir()); err == nil {
		t.Fatal("expected an error when git is unavailable")
	}
}
