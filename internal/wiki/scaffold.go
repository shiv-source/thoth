package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	agentgit "github.com/shiv-source/thoth/agent/git"
)

// ScaffoldOptions tunes ScaffoldWithOptions. Folders replaces the default set
// when non-empty; GitInit defaults to true through Scaffold.
type ScaffoldOptions struct {
	Folders []string
	GitInit bool
}

// gitignore protects editor cruft and stray database files in the wiki repo.
const gitignore = ".DS_Store\n*.db\n"

// Scaffold creates the wiki folder skeleton and CLAUDE.md under dir, and
// initializes a local git repo. It never overwrites an existing CLAUDE.md.
func Scaffold(dir string) error {
	return ScaffoldWithOptions(dir, ScaffoldOptions{GitInit: true})
}

// ScaffoldWithOptions creates the wiki skeleton with the given options. Git
// init is best-effort: a failure to version-control skips the repository step
// rather than failing the scaffold, since the folder skeleton is the real
// contract.
func ScaffoldWithOptions(dir string, opts ScaffoldOptions) error {
	folders := opts.Folders
	if len(folders) == 0 {
		folders = defaultFolders
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	for _, f := range folders {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("scaffold: %w", err)
		}
		// .gitkeep so empty folders survive the wiki's git repo.
		if _, err := os.Stat(filepath.Join(p, ".gitkeep")); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(filepath.Join(p, ".gitkeep"), nil, 0o644); err != nil {
				return fmt.Errorf("scaffold: %w", err)
			}
		}
	}
	rp := filepath.Join(dir, "CLAUDE.md")
	if _, err := os.Stat(rp); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(rp, []byte(RulebookFor(folders)), 0o644); err != nil {
			return fmt.Errorf("scaffold: %w", err)
		}
	}
	if opts.GitInit {
		gitInit(dir)
	}
	return nil
}

// EnsureReservedDir creates the reserved attachments directory (plus its
// .gitkeep so it survives the wiki's git repo) when missing. It runs on every
// startup so the wiki always has a home for non-markdown assets, even under a
// custom folder set or after the directory was deleted.
func EnsureReservedDir(root string) error {
	p := filepath.Join(root, AttachmentsDir)
	// A file squatting on the reserved name blocks the directory. Fail fast
	// with a hint instead of letting MkdirAll surface a bare "not a
	// directory", since the check runs on every startup.
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return fmt.Errorf("reserved directory %q is blocked by a file; move or remove %s", AttachmentsDir, p)
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return fmt.Errorf("ensure %s: %w", AttachmentsDir, err)
	}
	if _, err := os.Stat(filepath.Join(p, ".gitkeep")); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(p, ".gitkeep"), nil, 0o644); err != nil {
			return fmt.Errorf("ensure %s: %w", AttachmentsDir, err)
		}
	}
	return nil
}

// gitInit writes the .gitignore (unless present) and initializes a repository
// so the wiki is versioned from day one. Failures are ignored: versioning is
// additive to the scaffold, and a machine without a usable path still gets a
// valid wiki.
func gitInit(dir string) {
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			return
		}
	}
	_, _ = agentgit.Init(dir)
}

// EnsureGitRepo initializes a pure-Go git repository in root when it is not
// already one, writing the same .gitignore the scaffold uses. It runs on every
// startup so a pre-existing-but-unversioned wiki becomes versioned without a
// shell git dependency; it is a no-op when root is already a repository.
func EnsureGitRepo(root string) error {
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			return fmt.Errorf("ensure git: %w", err)
		}
	}
	if _, err := agentgit.Init(root); err != nil {
		return fmt.Errorf("ensure git: %w", err)
	}
	return nil
}
