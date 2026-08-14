package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServeRejectsMalformedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	thothDir := filepath.Join(home, ".thoth")
	if err := os.MkdirAll(thothDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thothDir, "config.toml"),
		[]byte("not valid toml {{{\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"serve"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for malformed config")
	}
}

func TestServeErrorWhenWikiScaffoldFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	thothDir := filepath.Join(home, ".thoth")
	if err := os.MkdirAll(thothDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// wiki_path points under a regular file: MkdirAll in Scaffold fails.
	blocker := filepath.Join(home, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := "wiki_path = " + filepath.Join(blocker, "wiki") + "\nhost = \"127.0.0.1\"\nport = 8333\n"
	if err := os.WriteFile(filepath.Join(thothDir, "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"serve"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when wiki scaffold fails")
	}
}

