package wiki

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestScaffoldCreatesSkeletonAndRulebook(t *testing.T) {
	dir := t.TempDir()

	if err := Scaffold(dir); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	for _, folder := range []string{
		"inbox", "meetings", "projects", "links", "setup", "knowledge", "todos", "daily", AttachmentsDir,
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

func TestScaffoldErrorWhenFolderIsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "inbox"), []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ScaffoldWithOptions(dir, ScaffoldOptions{Folders: []string{"inbox"}, GitInit: false}); err == nil {
		t.Fatal("expected error when a scaffold folder name is blocked by a file")
	}
}

func TestEnsureReservedDirErrorWhenRootIsFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(root, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureReservedDir(root); err == nil {
		t.Fatal("expected error when the wiki root is a file")
	}
}

func TestFoldersReturnsCopy(t *testing.T) {
	got := Folders()
	if !slices.Equal(got, defaultFolders) {
		t.Fatalf("Folders() = %v, want %v", got, defaultFolders)
	}
	// The returned slice is a copy: mutating it must not mutate the default.
	got[0] = "mutated"
	if Folders()[0] == "mutated" {
		t.Fatal("Folders() must return a copy of the defaults")
	}
}

func TestScaffoldCustomFolders(t *testing.T) {
	dir := t.TempDir()
	custom := []string{"journal", "recipes"}
	if err := ScaffoldWithOptions(dir, ScaffoldOptions{Folders: custom, GitInit: true}); err != nil {
		t.Fatalf("ScaffoldWithOptions: %v", err)
	}
	for _, folder := range custom {
		fi, err := os.Stat(filepath.Join(dir, folder))
		if err != nil || !fi.IsDir() {
			t.Fatalf("expected folder %s to exist: %v", folder, err)
		}
	}
	// Default folders are not created for a custom set.
	for _, folder := range defaultFolders {
		if _, err := os.Stat(filepath.Join(dir, folder)); err == nil {
			t.Fatalf("default folder %s must not exist for a custom set", folder)
		}
	}
	rulebook, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(rulebook) != RulebookFor(custom) {
		t.Fatal("scaffolded CLAUDE.md must equal RulebookFor(custom)")
	}
	for _, want := range []string{"- journal/ —", "- recipes/ —"} {
		if !strings.Contains(string(rulebook), want) {
			t.Fatalf("rulebook missing custom folder map line %q", want)
		}
	}
}

func TestScaffoldInitializesGit(t *testing.T) {
	dir := t.TempDir()
	if err := Scaffold(dir); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected a git repo to be initialized: %v", err)
	}
	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".DS_Store", "*.db"} {
		if !strings.Contains(string(gitignore), want) {
			t.Fatalf(".gitignore missing %q: %q", want, gitignore)
		}
	}
	// Idempotent: a second scaffold must not error.
	if err := Scaffold(dir); err != nil {
		t.Fatalf("second Scaffold: %v", err)
	}
}

// TestEnsureGitRepoInitsUnversionedWiki covers the startup git-init step: a
// pre-existing wiki with no repository becomes versioned (with the same
// .gitignore the scaffold writes), and an already-versioned wiki is left
// alone.
func TestEnsureGitRepoInitsUnversionedWiki(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureGitRepo(dir); err != nil {
		t.Fatalf("EnsureGitRepo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected a git repo: %v", err)
	}
	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".DS_Store", "*.db"} {
		if !strings.Contains(string(gitignore), want) {
			t.Fatalf(".gitignore missing %q: %q", want, gitignore)
		}
	}
	// Idempotent on the repo it just created.
	if err := EnsureGitRepo(dir); err != nil {
		t.Fatalf("second EnsureGitRepo: %v", err)
	}
	// Idempotent on a shell-git-initialized wiki too.
	other := t.TempDir()
	if err := Scaffold(other); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if err := EnsureGitRepo(other); err != nil {
		t.Fatalf("EnsureGitRepo on versioned wiki: %v", err)
	}
}

func TestRulebookDefaultFolderMap(t *testing.T) {
	for _, want := range []string{
		"- inbox/ — unfiled quick captures.",
		"- meetings/ — one file per meeting: YYYY-MM-DD-<topic>.md",
		"- projects/<name>/ — one folder per project.",
		"- todos/ — TODO.md is the ONLY task list: sections Now / Next / Someday.",
	} {
		if !strings.Contains(Rulebook(), want) {
			t.Fatalf("default rulebook missing folder map line %q", want)
		}
	}
}

func TestRulebookCustomFolderMap(t *testing.T) {
	got := RulebookFor([]string{"recipes"})
	if !strings.Contains(got, "- recipes/ — custom workspace folder.") {
		t.Fatalf("custom rulebook missing generic folder line:\n%s", got)
	}
	if strings.Contains(got, "- inbox/ —") {
		t.Fatalf("custom rulebook must not contain the default folder map:\n%s", got)
	}
}

func TestRulebookFrontmatterTypes(t *testing.T) {
	rulebook := Rulebook()
	want := "type: <" + strings.Join(NoteTypes(), "|") + ">"
	if !strings.Contains(rulebook, want) {
		t.Fatalf("rulebook must render the type list from NoteTypes(), got:\n%s", rulebook)
	}
	for _, legacy := range []string{"<meeting|project|link|setup|knowledge|todo|daily|note>", "type: note"} {
		if strings.Contains(rulebook, legacy) {
			t.Fatalf("rulebook must not contain legacy type list %q", legacy)
		}
	}
}

func TestNoteTypesDeriveFromFolders(t *testing.T) {
	want := []string{"inbox", "meeting", "project", "link", "setup", "knowledge", "todo", "daily"}
	if got := NoteTypes(); !slices.Equal(got, want) {
		t.Fatalf("NoteTypes() = %v, want %v", got, want)
	}
	// The type list is derived from the folder list, so it can never drift:
	// the only folder without a type is the reserved attachments dir.
	if got := NoteTypesFor(defaultFolders); len(got) != len(defaultFolders)-1 {
		t.Fatalf("expected one type per folder except attachments, got %d types for %d folders", len(got), len(defaultFolders))
	}
	for _, tp := range want {
		found := false
		for _, f := range defaultFolders {
			if f == AttachmentsDir {
				continue
			}
			if noteType(f) == tp {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no default folder derives the type %q", tp)
		}
	}
}

func TestNoteTypesForCustomFolders(t *testing.T) {
	if got := NoteTypesFor([]string{"journal", "recipes"}); !slices.Equal(got, []string{"journal", "recipe"}) {
		t.Fatalf("NoteTypesFor(custom) = %v, want [journal recipe]", got)
	}
	// Attachments is never a note type, even when listed.
	if got := NoteTypesFor([]string{"recipes", AttachmentsDir}); !slices.Equal(got, []string{"recipe"}) {
		t.Fatalf("NoteTypesFor with attachments = %v, want [recipe]", got)
	}
}

func TestRulebookCustomFolderTypes(t *testing.T) {
	rulebook := RulebookFor([]string{"journal", "recipes"})
	if !strings.Contains(rulebook, "type: <journal|recipe>") {
		t.Fatalf("custom rulebook must render types derived from its folders:\n%s", rulebook)
	}
	for _, leftover := range []string{"inbox|meeting", "type: note"} {
		if strings.Contains(rulebook, leftover) {
			t.Fatalf("custom rulebook must not carry default types %q:\n%s", leftover, rulebook)
		}
	}
}

func TestEnsureReservedDirCreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureReservedDir(dir); err != nil {
		t.Fatalf("EnsureReservedDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, AttachmentsDir)); err != nil {
		t.Fatalf("attachments dir not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, AttachmentsDir, ".gitkeep")); err != nil {
		t.Fatalf("attachments dir missing .gitkeep: %v", err)
	}
	// Idempotent: a second call keeps everything in place.
	if err := EnsureReservedDir(dir); err != nil {
		t.Fatalf("second EnsureReservedDir: %v", err)
	}
}

func TestEnsureReservedDirReportsBlockedByFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, AttachmentsDir)
	if err := os.WriteFile(blocker, []byte("oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := EnsureReservedDir(dir)
	if err == nil {
		t.Fatal("expected error when a file blocks the reserved attachments dir")
	}
	if !strings.Contains(err.Error(), "blocked by a file") || !strings.Contains(err.Error(), blocker) {
		t.Fatalf("error must name the conflict and the blocking path, got: %v", err)
	}
}

func TestEnsureReservedDirKeepsExistingContents(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, AttachmentsDir)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(p, "install.sh")
	if err := os.WriteFile(asset, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureReservedDir(dir); err != nil {
		t.Fatalf("EnsureReservedDir: %v", err)
	}
	if _, err := os.Stat(asset); err != nil {
		t.Fatalf("existing attachment was removed: %v", err)
	}
}
