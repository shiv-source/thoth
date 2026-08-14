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
