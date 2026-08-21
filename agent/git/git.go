// Package git is the pure-Go git backend the agent's git tools run on. It
// wraps go-git (no shell, no cgo, keeps all cross-compile targets green) and
// stays wiki-agnostic and dependency-free of internal/*. Identity and auth
// are always parameters — never stored here: hosts inject the committer
// Identity for commits and a lazy Auth for pushes.
package git

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// ErrNotRepository reports a path that is not a git repository. Tools surface
// it as a clean "not a git repository" message instead of a raw go-git error.
var ErrNotRepository = errors.New("not a git repository")

// ErrNoCommits reports a repository that has no commits yet, so operations
// that need a HEAD cannot run.
var ErrNoCommits = errors.New("repository has no commits yet")

// Identity is the committer identity for commits. The host injects it per
// call; the package never persists or guesses one.
type Identity struct {
	Name  string
	Email string
}

// Auth carries the credentials a push needs. The token is held only for the
// duration of the push call.
type Auth struct {
	Username string
	Token    string
}

// Repo is a handle on one git repository. It is safe for sequential use; a
// single call never runs concurrently.
type Repo struct {
	repo *git.Repository
}

// Init opens the repository at path, initializing a fresh one (default branch
// main, matching modern `git init`) only when path is not already a
// repository. It is idempotent: a second Init on the same path is a no-op.
func Init(path string) (*Repo, error) {
	r, err := git.PlainOpen(path)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		r, err = git.PlainInitWithOptions(path, &git.PlainInitOptions{
			InitOptions: git.InitOptions{DefaultBranch: plumbing.Main},
		})
		if err != nil {
			return nil, fmt.Errorf("git: init %s: %w", path, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("git: open %s: %w", path, err)
	}
	return &Repo{repo: r}, nil
}

// Open opens the repository at path, erroring with ErrNotRepository when path
// is not one. Read-only tools use Open so an unversioned wiki is reported
// cleanly instead of being silently initialized.
func Open(path string) (*Repo, error) {
	r, err := git.PlainOpen(path)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		return nil, fmt.Errorf("git: %w: %s", ErrNotRepository, path)
	}
	if err != nil {
		return nil, fmt.Errorf("git: open %s: %w", path, err)
	}
	return &Repo{repo: r}, nil
}

// Head returns the current HEAD commit hash.
func (r *Repo) Head() (string, error) {
	ref, err := r.repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return "", ErrNoCommits
	}
	if err != nil {
		return "", fmt.Errorf("git: head: %w", err)
	}
	return ref.Hash().String(), nil
}

// Status returns a git-status-short style report of the working tree, one
// "XY path" line per change (X staged, Y worktree), sorted by path. It is
// empty when the tree is clean.
func (r *Repo) Status() (string, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("git: worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("git: status: %w", err)
	}
	paths := make([]string, 0, len(status))
	for path := range status {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var sb strings.Builder
	for _, path := range paths {
		fs := status[path]
		sb.WriteByte(byte(fs.Staging))
		sb.WriteByte(byte(fs.Worktree))
		sb.WriteByte(' ')
		sb.WriteString(path)
		sb.WriteByte('\n')
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// Log returns the n most recent commits, newest first, one per line:
// "short-hash author <email> date subject". Fewer than n are shown when the
// repository has fewer commits.
func (r *Repo) Log(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("git: log count must be positive, got %d", n)
	}
	head, err := r.repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return "", ErrNoCommits
	}
	if err != nil {
		return "", fmt.Errorf("git: head: %w", err)
	}
	iter, err := r.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return "", fmt.Errorf("git: log: %w", err)
	}
	defer iter.Close()
	var lines []string
	for i := 0; i < n; i++ {
		c, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("git: log: %w", err)
		}
		subject, _, _ := strings.Cut(c.Message, "\n")
		lines = append(lines, fmt.Sprintf("%s %s <%s> %s %s",
			c.Hash.String()[:7], c.Author.Name, c.Author.Email,
			c.Author.When.Format("2006-01-02 15:04"), subject))
	}
	if len(lines) == 0 {
		return "", ErrNoCommits
	}
	return strings.Join(lines, "\n"), nil
}

// Diff returns the working-tree diff against HEAD as a unified diff. Untracked
// files are excluded, matching `git diff`; the result is empty when nothing
// tracked changed.
func (r *Repo) Diff() (string, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("git: worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("git: status: %w", err)
	}
	if status.IsClean() {
		return "", nil
	}
	head, err := r.repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return "", nil // no commits: everything is untracked, git diff shows nothing
	}
	if err != nil {
		return "", fmt.Errorf("git: head: %w", err)
	}
	commit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("git: resolve head: %w", err)
	}
	headTree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("git: head tree: %w", err)
	}
	var changes object.Changes
	paths := make([]string, 0, len(status))
	for path := range status {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fs := status[path]
		if fs.Staging == git.Untracked && fs.Worktree == git.Untracked {
			continue
		}
		change, err := worktreeChange(wt, headTree, path, fs, r.repo.Storer)
		if err != nil {
			return "", fmt.Errorf("git: diff %s: %w", path, err)
		}
		if change != nil {
			changes = append(changes, change)
		}
	}
	if len(changes) == 0 {
		return "", nil
	}
	patch, err := changes.Patch()
	if err != nil {
		return "", fmt.Errorf("git: patch: %w", err)
	}
	var sb strings.Builder
	if err := patch.Encode(&sb); err != nil {
		return "", fmt.Errorf("git: encode diff: %w", err)
	}
	return sb.String(), nil
}

// CommitAll stages every change and commits it under Identity id. A clean
// tree after staging is not an error: it reports false so the caller can say
// "nothing to commit".
func (r *Repo) CommitAll(msg string, id Identity) (bool, error) {
	if strings.TrimSpace(msg) == "" {
		return false, errors.New("git: commit message must not be empty")
	}
	if id.Name == "" || id.Email == "" {
		return false, errors.New("git: committer identity (name and email) is required")
	}
	wt, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("git: worktree: %w", err)
	}
	if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return false, fmt.Errorf("git: add: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("git: status: %w", err)
	}
	if status.IsClean() {
		return false, nil
	}
	if _, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{Name: id.Name, Email: id.Email, When: time.Now()},
	}); err != nil {
		return false, fmt.Errorf("git: commit: %w", err)
	}
	return true, nil
}

// SetRemote points the origin remote at url, adding it when it does not exist
// yet and replacing it when it does (set-url semantics). A second call with a
// different URL overrides the first.
func (r *Repo) SetRemote(url string) error {
	if _, err := r.repo.Remote("origin"); err != nil {
		if _, err := r.repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}}); err != nil {
			return fmt.Errorf("git: add remote: %w", err)
		}
		return nil
	}
	// go-git has no set-url; replacing the remote is delete + create.
	if err := r.repo.DeleteRemote("origin"); err != nil {
		return fmt.Errorf("git: replace remote: %w", err)
	}
	if _, err := r.repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}}); err != nil {
		return fmt.Errorf("git: replace remote: %w", err)
	}
	return nil
}

// Push pushes the current branch to the origin remote. A missing origin
// remote is a clean error; auth is applied only when a token is provided.
func (r *Repo) Push(a Auth) error {
	if _, err := r.repo.Remote("origin"); err != nil {
		return errors.New("git: no origin remote configured — connect a sync repository in Settings")
	}
	opts := &git.PushOptions{}
	if a.Token != "" {
		opts.Auth = &http.BasicAuth{Username: a.Username, Password: a.Token}
	}
	if err := r.repo.Push(opts); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("git: push: %w", err)
	}
	return nil
}
