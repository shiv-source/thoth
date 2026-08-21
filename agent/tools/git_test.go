package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	googit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/shiv-source/thoth/agent/git"
)

var testCtx = context.Background()

// committedDir builds a fresh repo with one committed file and returns its
// path.
func committedDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	r, err := git.Init(dir)
	if err != nil {
		t.Fatalf("git.Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CommitAll("first", git.Identity{Name: "Test", Email: "t@example.com"}); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	return dir
}

func gitOpts(dir string) GitOptions {
	return GitOptions{RepoPath: func() string { return dir }}
}

func TestGitToolSchemas(t *testing.T) {
	opts := gitOpts(t.TempDir())
	tools := []Tool{
		NewGitInit(opts),
		NewGitStatus(opts),
		NewGitLog(opts),
		NewGitDiff(opts),
		NewGitCommit(opts),
		NewGitPush(opts),
	}
	want := map[string][]string{
		"git_init":   nil,
		"git_status": nil,
		"git_log":    nil,
		"git_diff":   nil,
		"git_commit": {"message"},
		"git_push":   {"message"},
	}
	for _, tl := range tools {
		name := tl.Name()
		if name == "" || tl.Description() == "" {
			t.Fatalf("tool %q missing name or description", name)
		}
		if schema := tl.Schema(); schema["type"] != "object" {
			t.Fatalf("%s schema type = %v, want object", name, schema["type"])
		} else if reqs := want[name]; reqs == nil {
			if _, present := schema["required"]; present {
				t.Fatalf("%s schema should not require arguments", name)
			}
		} else if got, ok := schema["required"].([]string); !ok || !reflect.DeepEqual(got, reqs) {
			t.Fatalf("%s required = %v, want %v", name, schema["required"], reqs)
		}
	}
}

func TestGitInit(t *testing.T) {
	dir := t.TempDir() // not a repository yet
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatal("precondition: dir must not be a repo")
	}
	init := NewGitInit(gitOpts(dir))
	out, err := init.Run(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("git_init: %v", err)
	}
	if !strings.Contains(out, "initialized") {
		t.Fatalf("git_init = %q, want initialized", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("git_init should create a repository: %v", err)
	}
	// Idempotent on an existing repository.
	if _, err := init.Run(testCtx, map[string]any{}); err != nil {
		t.Fatalf("second git_init: %v", err)
	}
}

func TestGitStatusNotARepository(t *testing.T) {
	st := NewGitStatus(gitOpts(t.TempDir()))
	_, err := st.Run(testCtx, map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("git_status error = %v, want clean not-a-repo message", err)
	}
}

func TestGitStatusCleanAndDirty(t *testing.T) {
	dir := committedDir(t)
	st := NewGitStatus(gitOpts(dir))
	out, err := st.Run(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("git_status: %v", err)
	}
	if out != "working tree clean" {
		t.Fatalf("clean git_status = %q", out)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = st.Run(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("git_status: %v", err)
	}
	if !strings.Contains(out, "a.md") {
		t.Fatalf("dirty git_status = %q, want a.md", out)
	}
}

func TestGitLog(t *testing.T) {
	dir := committedDir(t)
	r, err := git.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i, msg := range []string{"second", "third"} {
		if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte(msg+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := r.CommitAll(msg, git.Identity{Name: "Test", Email: "t@example.com"}); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
				t.Fatal(err)
			}
		}
	}

	l := NewGitLog(gitOpts(dir))
	out, err := l.Run(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("git_log: %v", err)
	}
	if !strings.Contains(out, "third") || !strings.Contains(out, "first") {
		t.Fatalf("git_log = %q, want all commits", out)
	}
	if n := len(strings.Split(out, "\n")); n != 3 {
		t.Fatalf("git_log lines = %d, want 3", n)
	}
	out, err = l.Run(testCtx, map[string]any{"n": 1.0})
	if err != nil {
		t.Fatalf("git_log n=1: %v", err)
	}
	if n := len(strings.Split(out, "\n")); n != 1 {
		t.Fatalf("git_log n=1 lines = %d, want 1: %q", n, out)
	}
	if _, err := l.Run(testCtx, map[string]any{"n": 0.0}); err == nil {
		t.Fatal("git_log n=0 should error")
	}
}

func TestGitDiff(t *testing.T) {
	dir := committedDir(t)
	d := NewGitDiff(gitOpts(dir))
	out, err := d.Run(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("git_diff: %v", err)
	}
	if out != "no uncommitted changes" {
		t.Fatalf("clean git_diff = %q", out)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("x\ny\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = d.Run(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("git_diff: %v", err)
	}
	if !strings.Contains(out, "+y") {
		t.Fatalf("git_diff missing added line: %q", out)
	}
}

func TestGitCommitGuard(t *testing.T) {
	opts := GitOptions{
		RepoPath: func() string { return t.TempDir() },
		Guard:    func() error { return errors.New("git sync is disabled") },
	}
	c := NewGitCommit(opts)
	_, err := c.Run(testCtx, map[string]any{"message": "x"})
	if err == nil || !strings.Contains(err.Error(), "git sync is disabled") {
		t.Fatalf("git_commit error = %v, want guard message", err)
	}
}

func TestGitCommitRequiresIdentity(t *testing.T) {
	c := NewGitCommit(gitOpts(t.TempDir()))
	_, err := c.Run(testCtx, map[string]any{"message": "x"})
	if err == nil || !strings.Contains(err.Error(), "no git identity") {
		t.Fatalf("git_commit error = %v, want identity message", err)
	}
}

func TestGitCommitAutoInit(t *testing.T) {
	dir := t.TempDir() // not a repository yet
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := GitOptions{
		RepoPath: func() string { return dir },
		Identity: func() (GitIdentity, error) {
			return GitIdentity{Name: "Test", Email: "t@example.com"}, nil
		},
	}
	c := NewGitCommit(opts)
	out, err := c.Run(testCtx, map[string]any{"message": "first note"})
	if err != nil {
		t.Fatalf("git_commit: %v", err)
	}
	if !strings.Contains(out, "committed") {
		t.Fatalf("git_commit = %q, want committed", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("git_commit should have auto-initialized a repo: %v", err)
	}
}

func TestGitCommitCleanTree(t *testing.T) {
	opts := GitOptions{
		RepoPath: func() string { return committedDir(t) },
		Identity: func() (GitIdentity, error) {
			return GitIdentity{Name: "Test", Email: "t@example.com"}, nil
		},
	}
	out, err := NewGitCommit(opts).Run(testCtx, map[string]any{"message": "x"})
	if err != nil {
		t.Fatalf("git_commit: %v", err)
	}
	if !strings.Contains(out, "nothing to commit") {
		t.Fatalf("git_commit = %q, want nothing to commit", out)
	}
}

func TestGitPushRequiresAuthAndRemote(t *testing.T) {
	dir := committedDir(t)
	// No auth configured -> clean error before any push.
	opts := GitOptions{
		RepoPath: func() string { return dir },
		Identity: func() (GitIdentity, error) {
			return GitIdentity{Name: "Test", Email: "t@example.com"}, nil
		},
	}
	p := NewGitPush(opts)
	_, err := p.Run(testCtx, map[string]any{"message": "x"})
	if err == nil || !strings.Contains(err.Error(), "no GitHub connection") {
		t.Fatalf("git_push error = %v, want auth message", err)
	}
	// Auth present but no origin remote -> clean error.
	opts = GitOptions{
		RepoPath: func() string { return dir },
		Identity: func() (GitIdentity, error) {
			return GitIdentity{Name: "Test", Email: "t@example.com"}, nil
		},
		Auth: func() (GitAuth, error) { return GitAuth{Username: "u", Token: "t"}, nil },
	}
	p = NewGitPush(opts)
	_, err = p.Run(testCtx, map[string]any{"message": "x"})
	if err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("git_push error = %v, want missing-remote message", err)
	}
}

func TestGitPushToBareRemote(t *testing.T) {
	dir := committedDir(t)
	bare := t.TempDir()
	if _, err := googit.PlainInit(bare, true); err != nil {
		t.Fatal(err)
	}
	rr, err := googit.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rr.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{bare}}); err != nil {
		t.Fatal(err)
	}

	opts := GitOptions{
		RepoPath: func() string { return dir },
		Identity: func() (GitIdentity, error) {
			return GitIdentity{Name: "Test", Email: "t@example.com"}, nil
		},
		Auth: func() (GitAuth, error) { return GitAuth{Username: "u", Token: "t"}, nil },
	}
	p := NewGitPush(opts)
	out, err := p.Run(testCtx, map[string]any{"message": "sync"})
	if err != nil {
		t.Fatalf("git_push: %v", err)
	}
	if !strings.Contains(out, "pushed") {
		t.Fatalf("git_push = %q, want pushed", out)
	}

	// Pending changes are committed before pushing.
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = p.Run(testCtx, map[string]any{"message": "second"})
	if err != nil {
		t.Fatalf("git_push with changes: %v", err)
	}
	if !strings.Contains(out, "committed and pushed") {
		t.Fatalf("git_push with changes = %q, want committed and pushed", out)
	}
	// The bare remote has both commits.
	bareRepo, err := googit.PlainOpen(bare)
	if err != nil {
		t.Fatal(err)
	}
	mainRef, err := bareRepo.Reference(plumbing.NewBranchReferenceName("main"), false)
	if err != nil {
		t.Fatal(err)
	}
	iter, err := bareRepo.Log(&googit.LogOptions{From: mainRef.Hash()})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for {
		if _, err := iter.Next(); err != nil {
			break
		}
		count++
	}
	if count != 2 {
		t.Fatalf("bare repo has %d commits, want 2", count)
	}
}
