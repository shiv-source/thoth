package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/shiv-source/thoth/internal/github"
)

// gitTestDeps returns test deps wired for a successful sync: a stored GitHub
// connection (committer identity + push auth) and a fresh bare remote to push
// to. The bare path is a local filesystem remote, so no git binary is needed.
func gitTestDeps(t *testing.T) (Deps, string) {
	t.Helper()
	d := testDeps(t)
	if err := d.Store.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	if err := d.GitHub.Repo.Save(github.Auth{Token: "t", Username: "octo", DisplayName: "Octo", Email: "octo@example.com"}); err != nil {
		t.Fatal(err)
	}
	return d, initBare(t)
}

// initBare initializes an empty bare repository and returns its path.
func initBare(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, true); err != nil {
		t.Fatalf("init bare: %v", err)
	}
	return dir
}

func gitSetupReq(e http.Handler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/git/setup",
		bytes.NewReader([]byte(`{"url":"`+url+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestGitSetupRunsAgainstWiki(t *testing.T) {
	d, bare := gitTestDeps(t)
	// A note so the commit has something to stage.
	if err := os.WriteFile(filepath.Join(d.Wiki.Root(), "hello.md"),
		[]byte("---\ntitle: Hi\n---\n\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := gitSetupReq(New(d), bare)
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true, got %v %s", err, rec.Body.Bytes())
	}

	// The wiki's origin points at the remote and the bare remote received the
	// commit on main.
	repo, err := git.PlainOpen(d.Wiki.Root())
	if err != nil {
		t.Fatal(err)
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatalf("origin remote: %v", err)
	}
	urls := remote.Config().URLs
	if len(urls) != 1 || urls[0] != bare {
		t.Fatalf("origin URLs = %v, want [%s]", urls, bare)
	}
	bareRepo, err := git.PlainOpen(bare)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bareRepo.Reference(plumbing.NewBranchReferenceName("main"), false); err != nil {
		t.Fatalf("bare remote missing main: %v", err)
	}

	// A successful push records the sync outcome.
	last, syncErr, err := d.Settings.SyncState()
	if err != nil {
		t.Fatal(err)
	}
	if last == "" || !strings.HasSuffix(last, "Z") || syncErr != "" {
		t.Fatalf("sync state after success: last=%q err=%q", last, syncErr)
	}
}

func TestGitSetupEmptyTree(t *testing.T) {
	d, bare := gitTestDeps(t)
	e := New(d)
	// An empty tree is not an error: a repo with nothing to stage reports
	// nothing-to-commit and the sync still completes (the push is
	// already-up-to-date).
	if rec := gitSetupReq(e, bare); rec.Code != http.StatusOK {
		t.Fatalf("first status %d: %s", rec.Code, rec.Body.String())
	}
	rec := gitSetupReq(e, bare)
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true for an empty-tree commit, got %+v %s", body, rec.Body.String())
	}
}

func TestGitSetupReportsSanitizedFailure(t *testing.T) {
	d, _ := gitTestDeps(t)
	// A URL that is not a git repository makes the push fail.
	url := filepath.Join(t.TempDir(), "not-a-repo")
	rec := gitSetupReq(New(d), url)
	var body struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.OK || body.Error == "" {
		t.Fatalf("expected ok:false with a message, got %+v", body)
	}
	if strings.Contains(body.Error, url) || strings.Contains(body.Error, "token") {
		t.Fatalf("error must not echo the remote URL or credentials: %q", body.Error)
	}

	// A failed push records the sanitized error; last_synced_at stays empty.
	last, syncErr, err := d.Settings.SyncState()
	if err != nil {
		t.Fatal(err)
	}
	if last != "" || syncErr == "" || strings.Contains(syncErr, url) {
		t.Fatalf("sync state after failure: last=%q err=%q", last, syncErr)
	}
}

func TestGitSetupRequiresURL(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/git/setup", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGitSetupRequiresConnection(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	// No github_auth row: the sync needs the stored token to commit and push.
	rec := gitSetupReq(New(d), initBare(t))
	var body struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.OK || !strings.Contains(body.Error, "GitHub") {
		t.Fatalf("expected ok:false with a connection hint, got %+v", body)
	}
}
