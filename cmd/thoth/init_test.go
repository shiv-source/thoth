package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandUsesDefaultPath(t *testing.T) {
	// Point HOME at a temp dir so the default target (~/.thoth/wiki) lands
	// there instead of the real home directory.
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := newRootCmd()
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	p := filepath.Join(home, ".thoth", "wiki", "CLAUDE.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("default-path init did not scaffold CLAUDE.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".thoth", "wiki", "inbox")); err != nil {
		t.Fatalf("default-path init did not scaffold inbox: %v", err)
	}
}

func TestInitCommandExpandsTildeInTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := newRootCmd()
	root.SetArgs([]string{"init", "~/wiki2"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "wiki2", "CLAUDE.md")); err != nil {
		t.Fatalf("tilde target not scaffolded under HOME: %v", err)
	}
}

func TestInitCommandErrorOnUnwritableTarget(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	// MkdirAll under a regular file fails.
	target := filepath.Join(blocker, "wiki")

	root := newRootCmd()
	root.SetArgs([]string{"init", target})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unwritable target")
	}
	if !strings.Contains(err.Error(), "scaffold") {
		t.Fatalf("expected scaffold error, got %v", err)
	}
}

func TestInitCommandTooManyArgs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"init", "a", "b"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for more than one argument")
	}
}
