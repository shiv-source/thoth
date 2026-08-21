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
)

// writeFakeGit writes an executable git script that records every invocation
// (arguments joined by spaces, one per line) to logPath and exits 1 whenever
// the arguments contain the sentinel "fail-remote".
func writeFakeGit(t *testing.T) (binDir, logPath string) {
	t.Helper()
	logPath = filepath.Join(t.TempDir(), "git.log")
	script := `#!/bin/sh
echo "$@" >> "$FAKE_GIT_LOG"
case "$*" in
  *"fail-remote"*) echo "remote rejected" >&2; exit 1 ;;
esac
exit 0
`
	binDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir, logPath
}

func TestGitSetupRunsAgainstWiki(t *testing.T) {
	d := testDeps(t)
	// Boot seeds the metadata row; sync outcomes are recorded against it.
	if err := d.Store.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	binDir, logPath := writeFakeGit(t)
	t.Setenv("PATH", binDir)
	t.Setenv("FAKE_GIT_LOG", logPath)
	// A note so the commit has something to stage.
	if err := os.WriteFile(filepath.Join(d.Wiki.Root(), "hello.md"),
		[]byte("---\ntitle: Hi\n---\n\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	req := httptest.NewRequest(http.MethodPost, "/api/git/setup", bytes.NewReader([]byte(`{"url":"https://example.com/wiki.git"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true, got %v %s", err, rec.Body.Bytes())
	}

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"init",
		"remote set-url origin https://example.com/wiki.git",
		"add -A",
		"commit -m chore: sync wiki",
		"push -u origin HEAD",
	} {
		if !strings.Contains(string(log), want) {
			t.Fatalf("git log missing %q:\n%s", want, log)
		}
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

func TestGitSetupReportsSanitizedFailure(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	binDir, logPath := writeFakeGit(t)
	t.Setenv("PATH", binDir)
	t.Setenv("FAKE_GIT_LOG", logPath)
	e := New(d)

	url := "https://fail-remote.example.com/wiki.git"
	req := httptest.NewRequest(http.MethodPost, "/api/git/setup", bytes.NewReader([]byte(`{"url":"`+url+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
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
	if strings.Contains(body.Error, url) {
		t.Fatalf("error must not echo the remote URL: %q", body.Error)
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
	req := httptest.NewRequest(http.MethodPost, "/api/git/setup", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
