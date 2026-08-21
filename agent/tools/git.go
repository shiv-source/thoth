package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/shiv-source/thoth/agent/git"
)

// GitAuth is the credential pair a push uses. The token is never logged or
// stored by the tools: hosts hand it over through a lazy func per push.
type GitAuth = git.Auth

// GitIdentity is the committer identity a commit uses.
type GitIdentity = git.Identity

// GitOptions configures the git tools. RepoPath is evaluated per call so the
// tools follow a live root (e.g. a wiki that can move). Guard runs before the
// mutating tools (git_commit, git_push) and its error blocks them — commit and
// push only after a host says the user opted into sync. Auth and Identity are
// lazy: called only when a push or commit actually needs them, so a token is
// held for the shortest possible time.
type GitOptions struct {
	// RepoPath returns the absolute path of the repository the tools operate
	// on. Required; evaluated on every call.
	RepoPath func() string
	// Guard blocks the mutating tools (git_commit, git_push) when it errors.
	// Optional; nil means no guard.
	Guard func() error
	// Auth returns the credentials for a push. Optional; a nil func makes
	// git_push fail with a clean "not connected" error.
	Auth func() (GitAuth, error)
	// Identity returns the committer for a commit. Optional; a nil func makes
	// git_commit/git_push fail with a clean error.
	Identity func() (GitIdentity, error)
}

// openRepo opens the repository for the read-only tools, translating a
// missing repository into a clean, actionable error.
func openRepo(opts GitOptions) (*git.Repo, error) {
	path, err := repoPath(opts)
	if err != nil {
		return nil, err
	}
	r, err := git.Open(path)
	if errors.Is(err, git.ErrNotRepository) {
		return nil, errors.New("the workspace is not a git repository — nothing has been committed yet")
	}
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}
	return r, nil
}

// repoPath resolves and validates the configured repository path.
func repoPath(opts GitOptions) (string, error) {
	if opts.RepoPath == nil {
		return "", errors.New("no repository path is configured")
	}
	return opts.RepoPath(), nil
}

// ensureRepo opens the repository, initializing it (default branch main) when
// it does not exist yet — the mutating tools auto-init.
func ensureRepo(opts GitOptions) (*git.Repo, error) {
	path, err := repoPath(opts)
	if err != nil {
		return nil, err
	}
	r, err := git.Init(path)
	if err != nil {
		return nil, fmt.Errorf("initialize repository: %w", err)
	}
	return r, nil
}

// guardErr runs the tool's guard; nil when unguarded.
func guardErr(opts GitOptions) error {
	if opts.Guard == nil {
		return nil
	}
	return opts.Guard()
}

// identity returns the committer identity, with a clean error when the host
// has not configured one.
func identity(opts GitOptions) (GitIdentity, error) {
	if opts.Identity == nil {
		return GitIdentity{}, errors.New("no git identity is configured — connect GitHub in Settings to commit")
	}
	return opts.Identity()
}

// auth returns the push credentials, with a clean error when the host has not
// connected a GitHub account.
func auth(opts GitOptions) (GitAuth, error) {
	if opts.Auth == nil {
		return GitAuth{}, errors.New("no GitHub connection is configured — connect one in Settings to push")
	}
	return opts.Auth()
}

// GitInit is the "git_init" tool: it initializes a git repository in the
// workspace root when it is not already one. It is purely local (like the
// write tools), so it is not guarded by the sync switch.
type GitInit struct {
	opts GitOptions
}

// NewGitInit returns the git_init tool configured by opts.
func NewGitInit(opts GitOptions) *GitInit { return &GitInit{opts: opts} }

// Name implements Tool.
func (t *GitInit) Name() string { return "git_init" }

// Description implements Tool.
func (t *GitInit) Description() string {
	return "Initialize a git repository in the workspace if it is not already one, so status, log, commit and push work. Idempotent and purely local."
}

// Schema implements Tool.
func (t *GitInit) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Run implements Tool.
func (t *GitInit) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := ensureRepo(t.opts); err != nil {
		return "", fmt.Errorf("git_init: %w", err)
	}
	return "initialized a git repository in the workspace", nil
}

// GitStatus is the "git_status" tool: it reports the working-tree status.
type GitStatus struct {
	opts GitOptions
}

// NewGitStatus returns the git_status tool configured by opts.
func NewGitStatus(opts GitOptions) *GitStatus { return &GitStatus{opts: opts} }

// Name implements Tool.
func (t *GitStatus) Name() string { return "git_status" }

// Description implements Tool.
func (t *GitStatus) Description() string {
	return "Show the git working-tree status: which files are staged or changed, and whether the tree is clean. Read-only."
}

// Schema implements Tool.
func (t *GitStatus) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Run implements Tool.
func (t *GitStatus) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r, err := openRepo(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_status: %w", err)
	}
	out, err := r.Status()
	if err != nil {
		return "", fmt.Errorf("git_status: %w", err)
	}
	if out == "" {
		return "working tree clean", nil
	}
	return out, nil
}

// GitLog is the "git_log" tool: it lists the most recent commits.
type GitLog struct {
	opts GitOptions
}

// NewGitLog returns the git_log tool configured by opts.
func NewGitLog(opts GitOptions) *GitLog { return &GitLog{opts: opts} }

// Name implements Tool.
func (t *GitLog) Name() string { return "git_log" }

// Description implements Tool.
func (t *GitLog) Description() string {
	return "Show the most recent commits of the repository, newest first. Read-only."
}

// Schema implements Tool.
func (t *GitLog) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"n": map[string]any{
				"type":        "integer",
				"description": "Number of commits to show. Defaults to 10.",
			},
		},
	}
}

// Run implements Tool.
func (t *GitLog) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	n, err := IntArgDefault(args, "n", 10)
	if err != nil {
		return "", fmt.Errorf("git_log: %w", err)
	}
	if n <= 0 {
		return "", errors.New("git_log: n must be a positive number")
	}
	r, err := openRepo(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_log: %w", err)
	}
	out, err := r.Log(n)
	if err != nil {
		return "", fmt.Errorf("git_log: %w", err)
	}
	return out, nil
}

// GitDiff is the "git_diff" tool: it shows the working-tree diff.
type GitDiff struct {
	opts GitOptions
}

// NewGitDiff returns the git_diff tool configured by opts.
func NewGitDiff(opts GitOptions) *GitDiff { return &GitDiff{opts: opts} }

// Name implements Tool.
func (t *GitDiff) Name() string { return "git_diff" }

// Description implements Tool.
func (t *GitDiff) Description() string {
	return "Show the working-tree diff against the last commit as a unified diff. Read-only."
}

// Schema implements Tool.
func (t *GitDiff) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Run implements Tool.
func (t *GitDiff) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r, err := openRepo(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_diff: %w", err)
	}
	out, err := r.Diff()
	if err != nil {
		return "", fmt.Errorf("git_diff: %w", err)
	}
	if out == "" {
		return "no uncommitted changes", nil
	}
	return out, nil
}

// GitCommit is the "git_commit" tool: it stages everything and commits.
type GitCommit struct {
	opts GitOptions
}

// NewGitCommit returns the git_commit tool configured by opts.
func NewGitCommit(opts GitOptions) *GitCommit { return &GitCommit{opts: opts} }

// Name implements Tool.
func (t *GitCommit) Name() string { return "git_commit" }

// Description implements Tool.
func (t *GitCommit) Description() string {
	return "Stage all changes and commit them with the given message. Initializes the repository if needed. Requires git sync to be enabled in Settings."
}

// Schema implements Tool.
func (t *GitCommit) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The commit message.",
			},
		},
		"required": []string{"message"},
	}
}

// Run implements Tool.
func (t *GitCommit) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := guardErr(t.opts); err != nil {
		return "", fmt.Errorf("git_commit: %w", err)
	}
	msg, err := StringArg(args, "message")
	if err != nil {
		return "", fmt.Errorf("git_commit: %w", err)
	}
	id, err := identity(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_commit: %w", err)
	}
	r, err := ensureRepo(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_commit: %w", err)
	}
	committed, err := r.CommitAll(msg, id)
	if err != nil {
		return "", fmt.Errorf("git_commit: %w", err)
	}
	if !committed {
		return "nothing to commit — working tree clean", nil
	}
	return fmt.Sprintf("committed %q", msg), nil
}

// GitPush is the "git_push" tool: it commits pending changes and pushes.
type GitPush struct {
	opts GitOptions
}

// NewGitPush returns the git_push tool configured by opts.
func NewGitPush(opts GitOptions) *GitPush { return &GitPush{opts: opts} }

// Name implements Tool.
func (t *GitPush) Name() string { return "git_push" }

// Description implements Tool.
func (t *GitPush) Description() string {
	return "Commit any pending changes with the given message, then push to the origin remote. Initializes the repository if needed. Requires git sync to be enabled and a GitHub connection in Settings."
}

// Schema implements Tool.
func (t *GitPush) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Commit message used for any pending changes.",
			},
		},
		"required": []string{"message"},
	}
}

// Run implements Tool.
func (t *GitPush) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := guardErr(t.opts); err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	msg, err := StringArg(args, "message")
	if err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	id, err := identity(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	a, err := auth(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	r, err := ensureRepo(t.opts)
	if err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	committed, err := r.CommitAll(msg, id)
	if err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	if err := r.Push(a); err != nil {
		return "", fmt.Errorf("git_push: %w", err)
	}
	if committed {
		return "committed and pushed", nil
	}
	return "pushed (nothing new to commit)", nil
}
