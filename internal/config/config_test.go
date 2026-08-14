package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.WikiPath != "~/.thoth/wiki" || c.Host != "127.0.0.1" || c.Port != 8333 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c != Default() {
		t.Fatalf("expected defaults, got %+v", c)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.toml")
	want := Default()
	want.WikiPath = "/tmp/custom/wiki"
	want.Port = 9999
	want.PermissionMode = "acceptEdits"

	if err := Save(p, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Fatalf("round trip mismatch: want %+v got %+v", want, got)
	}
}

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
