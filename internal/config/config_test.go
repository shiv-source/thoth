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

func TestLoadRejectsMalformedTOML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte("this is not valid toml {{{\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for malformed TOML")
	}
}

func TestLoadRejectsWrongTypes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte("port = \"not a number\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for wrong TOML types")
	}
}

func TestLoadUnreadablePath(t *testing.T) {
	// A directory path is not IsNotExist, so Load must surface the read error.
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected error reading a directory as a file")
	}
}

func TestSaveErrorWhenParentIsFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	// MkdirAll on a path under a regular file fails.
	if err := Save(filepath.Join(blocker, "config.toml"), Default()); err == nil {
		t.Fatal("expected error when parent dir cannot be created")
	}
}

func TestSaveErrorWhenTargetIsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := Save(filepath.Join(dir, "config.toml"), Default()); err != nil {
		t.Fatalf("save to fresh path: %v", err)
	}
	// WriteFile onto an existing directory fails.
	if err := Save(dir, Default()); err == nil {
		t.Fatal("expected error writing over a directory")
	}
}
