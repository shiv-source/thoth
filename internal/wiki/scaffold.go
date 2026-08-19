package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shiv-source/thoth/internal/gitutil"
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
// init is best-effort: a missing git binary skips the repository step rather
// than failing the scaffold, since the folder skeleton is the real contract.
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

// gitInit writes the .gitignore (unless present) and initializes a repository
// so the wiki is versioned from day one. Failures are ignored: versioning is
// additive to the scaffold, and a machine without git still gets a valid wiki.
func gitInit(dir string) {
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			return
		}
	}
	_ = gitutil.Init(dir)
}
