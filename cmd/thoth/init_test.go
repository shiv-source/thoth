package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCommandScaffolds(t *testing.T) {
	target := filepath.Join(t.TempDir(), "wiki")
	root := newRootCmd()
	root.SetArgs([]string{"init", target})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "CLAUDE.md")); err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	for _, f := range []string{"inbox", "meetings", "todos"} {
		if _, err := os.Stat(filepath.Join(target, f)); err != nil {
			t.Fatalf("folder %s not created: %v", f, err)
		}
	}
}
