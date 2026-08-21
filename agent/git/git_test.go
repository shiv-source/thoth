package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

var testIdentity = Identity{Name: "Test User", Email: "test@example.com"}

// initTestRepo returns a fresh repo plus its path.
func initTestRepo(t *testing.T) (*Repo, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return r, dir
}

// commitFile writes a file and commits it, returning the repo.
func commitFile(t *testing.T, r *Repo, dir, rel, content, msg string) {
	t.Helper()
	writeFile(t, dir, rel, content)
	committed, err := r.CommitAll(msg, testIdentity)
	if err != nil {
		t.Fatalf("CommitAll(%q): %v", msg, err)
	}
	if !committed {
		t.Fatalf("CommitAll(%q): nothing committed", msg)
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInitIdempotentAndDefaultsToMain(t *testing.T) {
	dir := t.TempDir()
	r, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected .git: %v", err)
	}
	// A second Init is a no-op, not an error.
	again, err := Init(dir)
	if err != nil {
		t.Fatalf("second Init: %v", err)
	}
	head, err := again.Head()
	if !errors.Is(err, ErrNoCommits) {
		t.Fatalf("Head on empty repo = %q, %v, want ErrNoCommits", head, err)
	}
	// Committing on a fresh repo lands on main.
	writeFile(t, dir, "a.md", "hi")
	if _, err := r.CommitAll("first", testIdentity); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	br, err := rawBranch(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if br != "refs/heads/main" {
		t.Fatalf("HEAD branch = %q, want main", br)
	}
}

func rawBranch(t *testing.T, dir string) (string, error) {
	t.Helper()
	rr, err := git.PlainOpen(dir)
	if err != nil {
		return "", err
	}
	ref, err := rr.Head()
	if err != nil {
		return "", err
	}
	return ref.Name().String(), nil
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	if _, err := Open(dir); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("Open(non-repo) = %v, want ErrNotRepository", err)
	}
	_, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err != nil {
		t.Fatalf("Open(repo): %v", err)
	}
	// A path that is not a directory cannot be opened or initialized.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(blocker); err == nil || errors.Is(err, ErrNotRepository) {
		t.Fatalf("Open(file) = %v, want a non-not-repository error", err)
	}
	if _, err := Init(blocker); err == nil {
		t.Fatal("Init on a file should error")
	}
	// Init creates the parent directories it needs.
	deep := filepath.Join(t.TempDir(), "no", "parent")
	if _, err := Init(deep); err != nil {
		t.Fatalf("Init under missing parents: %v", err)
	}
	if _, err := os.Stat(filepath.Join(deep, ".git")); err != nil {
		t.Fatalf("expected .git under created parents: %v", err)
	}
}

func TestHead(t *testing.T) {
	r, dir := initTestRepo(t)
	if _, err := r.Head(); !errors.Is(err, ErrNoCommits) {
		t.Fatalf("Head on empty = %v, want ErrNoCommits", err)
	}
	commitFile(t, r, dir, "a.md", "x", "first")
	head, err := r.Head()
	if err != nil {
		t.Fatal(err)
	}
	if len(head) != 40 {
		t.Fatalf("Head hash = %q, want 40 hex chars", head)
	}
}

func TestStatus(t *testing.T) {
	r, dir := initTestRepo(t)
	commitFile(t, r, dir, "a.md", "x", "first")
	out, err := r.Status()
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("clean status = %q, want empty", out)
	}
	writeFile(t, dir, "a.md", "y")
	writeFile(t, dir, "b.md", "z")
	out, err = r.Status()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("status lines = %d, want 2: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], " M a.md") && !strings.HasPrefix(lines[1], " M a.md") {
		t.Fatalf("status = %q, want a modified a.md", out)
	}
	if !strings.HasPrefix(lines[0], "?? b.md") && !strings.HasPrefix(lines[1], "?? b.md") {
		t.Fatalf("status = %q, want untracked b.md", out)
	}
}

func TestLog(t *testing.T) {
	r, dir := initTestRepo(t)
	if _, err := r.Log(10); !errors.Is(err, ErrNoCommits) {
		t.Fatalf("Log on empty = %v, want ErrNoCommits", err)
	}
	commitFile(t, r, dir, "a.md", "x", "first")
	commitFile(t, r, dir, "b.md", "y", "second")
	commitFile(t, r, dir, "c.md", "z", "third")

	out, err := r.Log(2)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("Log(2) lines = %d, want 2: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "third") || !strings.Contains(lines[1], "second") {
		t.Fatalf("Log(2) = %q, want newest first with subjects", out)
	}
	if !strings.Contains(lines[0], testIdentity.Name) || !strings.Contains(lines[0], testIdentity.Email) {
		t.Fatalf("Log line lacks identity: %q", lines[0])
	}
	if _, err := r.Log(0); err == nil {
		t.Fatal("Log(0) should error")
	}
}

func TestCommitAll(t *testing.T) {
	r, dir := initTestRepo(t)
	// Identity is required.
	if _, err := r.CommitAll("first", Identity{}); err == nil {
		t.Fatal("CommitAll with empty identity should error")
	}
	if _, err := r.CommitAll("", testIdentity); err == nil {
		t.Fatal("CommitAll with empty message should error")
	}
	// Clean tree reports false, not an error.
	committed, err := r.CommitAll("first", testIdentity)
	if err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	if committed {
		t.Fatal("CommitAll on a clean tree should report nothing committed")
	}
	// With changes it commits.
	writeFile(t, dir, "a.md", "x")
	committed, err = r.CommitAll("first", testIdentity)
	if err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	if !committed {
		t.Fatal("CommitAll should report a commit")
	}
	// The commit is on the log with the message.
	out, err := r.Log(1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "first") {
		t.Fatalf("Log = %q, want commit message", out)
	}
}

func TestDiff(t *testing.T) {
	r, dir := initTestRepo(t)
	commitFile(t, r, dir, "a.md", "line one\n", "first")
	// Untracked files do not appear, matching git diff.
	writeFile(t, dir, "untracked.md", "never tracked\n")
	out, err := r.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("diff with only untracked = %q, want empty", out)
	}
	// A modification shows as a unified diff.
	writeFile(t, dir, "a.md", "line one\nline two\n")
	out, err = r.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "diff --git a/a.md b/a.md") {
		t.Fatalf("diff missing header: %q", out)
	}
	if !strings.Contains(out, "+line two") {
		t.Fatalf("diff missing added line: %q", out)
	}
	// A deleted tracked file appears as a deletion.
	if err := os.Remove(filepath.Join(dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	out, err = r.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "-line one") {
		t.Fatalf("diff missing removed line: %q", out)
	}
	// A repo with no commits diffs empty (everything untracked).
	empty, emptyDir := initTestRepo(t)
	writeFile(t, emptyDir, "x.md", "x")
	out, err = empty.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("diff on empty repo = %q, want empty", out)
	}
}

func TestPush(t *testing.T) {
	r, dir := initTestRepo(t)
	commitFile(t, r, dir, "a.md", "x", "first")
	// No origin remote -> clean error.
	if err := r.Push(Auth{}); err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("Push without remote = %v, want clean origin error", err)
	}

	bare := initBare(t)
	rr, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rr.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{bare}}); err != nil {
		t.Fatalf("CreateRemote: %v", err)
	}
	if err := r.Push(Auth{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	// A second push is already-up-to-date, not an error.
	if err := r.Push(Auth{Username: "u", Token: "t"}); err != nil {
		t.Fatalf("second Push: %v", err)
	}
	// The bare remote saw the commit.
	if got := rawLog(t, bare); !strings.Contains(got, "first") {
		t.Fatalf("bare log = %q, want pushed commit", got)
	}
}

func initBare(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, true); err != nil {
		t.Fatalf("init bare: %v", err)
	}
	return dir
}

func rawLog(t *testing.T, dir string) string {
	t.Helper()
	rr, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := rr.Reference(plumbing.NewBranchReferenceName("main"), false)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := rr.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}
	return commit.Message
}
