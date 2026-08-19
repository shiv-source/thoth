// Package gitutil holds the shared git commands both the wiki scaffold and
// the git-sync API use. Keeping the command in one place means behavior (the
// 15s timeout, the sanitized failure message) never diverges between them.
package gitutil

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// stepTimeout bounds every git command; a hung git must not wedge the caller.
const stepTimeout = 15 * time.Second

// Init runs `git init` in dir unless it is already a repository. The error is
// a fixed message: the raw exec error carries the command line, never shown.
func Init(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), stepTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = dir
	if _, err := cmd.CombinedOutput(); err != nil {
		return errors.New("could not initialize a git repository — check that git is installed and the directory is writable")
	}
	return nil
}
