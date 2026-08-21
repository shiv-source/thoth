package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/wiki"
)

func doReq(t *testing.T, e http.Handler, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// ---- settings ----

func TestGetSettingsStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodGet, "/api/settings", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestGetSettingsProviderReadError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodGet, "/api/settings", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500 (provider config read)", rec.Code)
	}
}

// ---- models ----

func TestModelsListStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodGet, "/api/models", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsCreateStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodPost, "/api/models", `{"value":"x","name":"X"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsCreateMalformedBody(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodPost, "/api/models", `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestModelsUpdateMalformedID(t *testing.T) {
	d := testDeps(t)
	rec := doReq(t, New(d), http.MethodPut, "/api/models/abc", `{"value":"x","name":"X"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestModelsUpdateMalformedBody(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodPut, "/api/models/1", `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestModelsUpdateStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodPut, "/api/models/1", `{"value":"x","name":"X"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsUpdateRenamesSelectedSettingError(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("old-value", "Old", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodPut, "/api/models/"+strconv.FormatInt(m.ID, 10), `{"value":"new-value","name":"New"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500 (model setting read)", rec.Code)
	}
}

func TestModelsDeleteMalformedID(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodDelete, "/api/models/abc", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestModelsDeleteStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodDelete, "/api/models/1", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsDeleteClearsSelectedSettingError(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("doomed", "Doomed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodDelete, "/api/models/"+strconv.FormatInt(m.ID, 10), "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500 (model setting read)", rec.Code)
	}
}

// ---- git ----

// writeFailGit writes a git stub that records invocations and fails on a
// command line matching FAKE_GIT_FAIL (regex) or, for FAKE_GIT_COMMIT_EMPTY,
// simulates an empty-tree commit.
func writeFailGit(t *testing.T) (binDir, logPath string) {
	t.Helper()
	logPath = filepath.Join(t.TempDir(), "git.log")
	script := `#!/bin/sh
echo "$@" >> "$FAKE_GIT_LOG"
if [ -n "$FAKE_GIT_COMMIT_EMPTY" ] && [ "$1" = "commit" ]; then
  echo "nothing to commit" >&2
  exit 1
fi
if [ -n "$FAKE_GIT_FAIL" ] && echo "$*" | grep -qE "$FAKE_GIT_FAIL"; then
  echo "simulated failure" >&2
  exit 1
fi
exit 0
`
	binDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir, logPath
}

func setupGitDeps(t *testing.T, fail, commitEmpty string) (Deps, *httptest.ResponseRecorder) {
	t.Helper()
	d := testDeps(t)
	if err := d.Store.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	binDir, logPath := writeFailGit(t)
	// Prepend, don't replace: the fake script shells out to grep.
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GIT_LOG", logPath)
	if fail != "" {
		t.Setenv("FAKE_GIT_FAIL", fail)
	}
	if commitEmpty != "" {
		t.Setenv("FAKE_GIT_COMMIT_EMPTY", commitEmpty)
	}
	rec := doReq(t, New(d), http.MethodPost, "/api/git/setup", `{"url":"https://example.com/wiki.git"}`)
	return d, rec
}

func TestGitSetupInitFailure(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	// A wiki root that does not exist makes `git init` fail.
	d.Wiki = wiki.New(filepath.Join(t.TempDir(), "missing"))
	binDir, logPath := writeFailGit(t)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GIT_LOG", logPath)
	rec := doReq(t, New(d), http.MethodPost, "/api/git/setup", `{"url":"https://example.com/wiki.git"}`)
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.OK {
		t.Fatalf("expected ok:false from a failing git init, got %+v %s", body, rec.Body.String())
	}
}

func TestGitSetRemoteAddPath(t *testing.T) {
	// get-url fails (no origin yet) and `remote add` succeeds: the remote is
	// added and the flow continues to a successful sync.
	_, rec := setupGitDeps(t, "remote get-url", "")
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true after remote add, got %+v %s", body, rec.Body.String())
	}
}

func TestGitSetRemoteAddFailure(t *testing.T) {
	_, rec := setupGitDeps(t, "remote get-url origin|remote add", "")
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Fatalf("expected ok:false from a failing remote add: %s", rec.Body.String())
	}
}

func TestGitSetRemoteSetURLFailure(t *testing.T) {
	_, rec := setupGitDeps(t, "remote set-url", "")
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Fatalf("expected ok:false from a failing set-url: %s", rec.Body.String())
	}
}

func TestGitCommitAddFailure(t *testing.T) {
	_, rec := setupGitDeps(t, "add -A", "")
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Fatalf("expected ok:false from a failing git add: %s", rec.Body.String())
	}
}

func TestGitCommitFailure(t *testing.T) {
	_, rec := setupGitDeps(t, "commit -m", "")
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Fatalf("expected ok:false from a failing git commit: %s", rec.Body.String())
	}
}

func TestGitCommitNothingToCommit(t *testing.T) {
	// An empty tree is not an error: the commit step reports nothing-to-commit
	// and the sync still completes.
	_, rec := setupGitDeps(t, "", "1")
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true for an empty-tree commit, got %+v %s", body, rec.Body.String())
	}
}

func TestGitPushFailure(t *testing.T) {
	_, rec := setupGitDeps(t, "push -u", "")
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Fatalf("expected ok:false from a failing git push: %s", rec.Body.String())
	}
}

// ---- github ----

func TestConnectGitHubNotConfigured(t *testing.T) {
	d := testDeps(t)
	d.GitHub = nil
	if rec := doReq(t, New(d), http.MethodPost, "/api/github/auth", `{"token":"t"}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestConnectGitHubSaveError(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Close(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"login":"octo","name":"Octo"}`)
	}))
	defer srv.Close()
	d.GitHub.Client = github.New(http.DefaultClient).WithBaseURL(srv.URL)
	if rec := doReq(t, New(d), http.MethodPost, "/api/github/auth", `{"token":"t"}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestGetGitHubAuthError(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodGet, "/api/github/auth", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestListGitHubReposNotConfigured(t *testing.T) {
	d := testDeps(t)
	d.GitHub = nil
	if rec := doReq(t, New(d), http.MethodGet, "/api/github/repos", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestListGitHubReposGetError(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodGet, "/api/github/repos", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestListGitHubReposClientNotConfigured(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Save(github.Auth{Token: "t", Username: "octo"}); err != nil {
		t.Fatal(err)
	}
	d.GitHub.Client = nil
	if rec := doReq(t, New(d), http.MethodGet, "/api/github/repos", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestListGitHubReposFetchError(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Save(github.Auth{Token: "t", Username: "octo"}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	d.GitHub.Client = github.New(http.DefaultClient).WithBaseURL(srv.URL)
	if rec := doReq(t, New(d), http.MethodGet, "/api/github/repos", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestDisconnectGitHubError(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodDelete, "/api/github/auth", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

// ---- notes / fs ----

func TestNoteUnknownExtensionFallsBackToOctetStream(t *testing.T) {
	d := testDeps(t)
	p := filepath.Join(d.Wiki.Root(), "asset.qzy")
	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/notes?path=asset.qzy", nil)
	rec := httptest.NewRecorder()
	New(d).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type = %q, want application/octet-stream", ct)
	}
	if rec.Header().Get("Content-Disposition") == "" {
		t.Fatal("non-image attachment must be served as a download")
	}
}

// ---- chat / hub ----

func TestNewHubNilCtx(t *testing.T) {
	d := testDeps(t)
	h := NewHub(&FakeClient{}, d.Store, d.Log, nil)
	if h.ctx == nil {
		t.Fatal("nil ctx must default to background")
	}
}

func TestHubRecordCapsAt500(t *testing.T) {
	d := testDeps(t)
	h := NewHub(&FakeClient{}, d.Store, d.Log, context.Background())
	for i := 0; i < 600; i++ {
		h.record("c1", serverMsg{Type: "assistant_delta", Text: "x"})
	}
	if got := len(h.replay("c1")); got != 500 {
		t.Fatalf("replay length = %d, want 500", got)
	}
}

func TestAllowLocalOriginRejectsMalformedOrigin(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	u := wsURL(t, e)
	// url.Parse fails on this Origin, so the upgrade must be refused.
	if _, _, err := websocket.DefaultDialer.Dial(u, http.Header{"Origin": []string{"http://exa mple.com"}}); err == nil {
		t.Fatal("expected handshake failure for a malformed Origin")
	}
}

func TestChatSendConversationCreateError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" {
		t.Fatalf("expected error frame, got %+v", m)
	}
}

func TestChatOpenConversationExistsError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.WriteJSON(map[string]string{"type": "open", "conversation_id": "any"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" || m["message"] != "unknown conversation" {
		t.Fatalf("expected unknown-conversation error, got %+v", m)
	}
}

func TestChatTurnWriterSurfacesAgentErrorEvent(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventDelta, Text: "partial"},
		{Type: agentlib.EventError, Detail: "model exploded"},
		{Type: agentlib.EventDone},
	}}
	e := New(d)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	var sawErr bool
	for {
		m := readMsg(t, conn)
		if m["type"] == "error" && m["message"] == "model exploded" {
			sawErr = true
			break
		}
	}
	if !sawErr {
		t.Fatal("agent error event did not surface as an error frame")
	}
}

// ---- logging ----

func TestRequestLogNilIsIdentity(t *testing.T) {
	e := echo.New()
	e.Use(requestLog(nil))
	e.GET("/api/health", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("nil-log middleware must pass requests through, got %d", rec.Code)
	}
}

func TestRequestLogLogsErrors(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	e := echo.New()
	e.Use(requestLog(log))
	e.GET("/api/boom", func(c echo.Context) error { return errors.New("kaboom") })
	req := httptest.NewRequest(http.MethodGet, "/api/boom", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if !strings.Contains(buf.String(), "kaboom") {
		t.Fatalf("request log missing error attr: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "path=/api/boom") {
		t.Fatalf("request log missing path attr: %s", buf.String())
	}
}

// ---- health ----

func TestHealthProviderNameDegrades(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if got := providerName(d.Store, "claude-sonnet-5"); got != "" {
		t.Fatalf("providerName = %q, want \"\" when the store is closed", got)
	}
	if got := providerName(nil, "claude-sonnet-5"); got != "" {
		t.Fatalf("providerName(nil store) = %q, want \"\"", got)
	}
	if got := providerName(d.Store, ""); got != "" {
		t.Fatalf("providerName(empty model) = %q, want \"\"", got)
	}
}

func TestHealthProviderNameUnknownModel(t *testing.T) {
	d := testDeps(t)
	if got := providerName(d.Store, "no-such-model"); got != "" {
		t.Fatalf("providerName(unknown) = %q, want \"\"", got)
	}
	if _, err := d.Store.CreateModel("m", "M", "", "Vendor"); err != nil {
		t.Fatal(err)
	}
	if got := providerName(d.Store, "m"); got != "Vendor" {
		t.Fatalf("providerName = %q, want Vendor", got)
	}
}

// ---- fake client ----

func TestFakeClientSurfacesWriterError(t *testing.T) {
	f := &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDelta, Text: "x"}}}
	boom := errors.New("writer boom")
	_, err := f.Start(context.Background(), "c1", "p", agentlib.WriterFunc(func(agentlib.Event) error { return boom }))
	if !errors.Is(err, boom) {
		t.Fatalf("Start error = %v, want writer boom", err)
	}
}

// ---- settings KV ----

func TestSettingsKVError(t *testing.T) {
	d := testDeps(t)
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.SetSetting(settings.KeyModel, "x"); err == nil {
		t.Fatal("SetSetting on a closed repo must error")
	}
}
