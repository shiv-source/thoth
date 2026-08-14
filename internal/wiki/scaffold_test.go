package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCreatesSkeletonAndRulebook(t *testing.T) {
	dir := t.TempDir()

	if err := Scaffold(dir); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	for _, folder := range []string{
		"inbox", "meetings", "projects", "links", "setup", "knowledge", "todos", "daily",
	} {
		fi, err := os.Stat(filepath.Join(dir, folder))
		if err != nil || !fi.IsDir() {
			t.Fatalf("expected folder %s to exist: %v", folder, err)
		}
	}

	rulebook, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md: %v", err)
	}
	if string(rulebook) != Rulebook() {
		t.Fatal("scaffolded CLAUDE.md must equal Rulebook() — they are generated from one source")
	}
	for _, want := range []string{"save protocol", "Folder map", "NEVER store secrets"} {
		if !strings.Contains(string(rulebook), want) {
			t.Fatalf("rulebook missing section %q", want)
		}
	}
}

func TestScaffoldKeepsExistingCLAUDE(t *testing.T) {
	dir := t.TempDir()
	custom := "# my custom rules\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Scaffold(dir); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != custom {
		t.Fatalf("existing CLAUDE.md was overwritten: %q", b)
	}
}

func TestScaffoldIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := Scaffold(dir); err != nil {
		t.Fatal(err)
	}
	if err := Scaffold(dir); err != nil {
		t.Fatalf("second Scaffold: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Fatalf("CLAUDE.md missing after second Scaffold: %v", err)
	}
}

func TestScaffoldErrorWhenParentIsFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Scaffold(filepath.Join(blocker, "wiki")); err == nil {
		t.Fatal("expected error when scaffold target is under a file")
	}
}
