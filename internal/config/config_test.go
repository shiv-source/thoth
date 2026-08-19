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

func TestToTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"under home becomes tilde form", filepath.Join(home, ".thoth", "dev", "wiki"), "~/.thoth/dev/wiki"},
		{"home itself", home, "~"},
		{"outside home passes through", "/tmp/notes", "/tmp/notes"},
		{"tilde form passes through", "~/.thoth/wiki", "~/.thoth/wiki"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTilde(tt.in); got != tt.want {
				t.Fatalf("ToTilde(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
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
