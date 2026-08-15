package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	got, err := ExpandHome("~/.thoth/wiki")
	if err != nil {
		t.Fatalf("ExpandHome: %v", err)
	}
	if got != filepath.Join(home, ".thoth", "wiki") {
		t.Fatalf("got %q", got)
	}
	if got, _ := ExpandHome("/abs/path"); got != "/abs/path" {
		t.Fatalf("absolute path must pass through, got %q", got)
	}
}

func TestExpandHomeBareTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	got, err := ExpandHome("~")
	if err != nil {
		t.Fatalf("ExpandHome(~): %v", err)
	}
	if got != home {
		t.Fatalf("ExpandHome(~) = %q, want %q", got, home)
	}
}
