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

	"github.com/go-git/go-git/v5"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/settings"
	syncsvc "github.com/shiv-source/thoth/internal/sync"
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
	if rec := doReq(t, New(d), http.MethodGet, "/api/v1/settings", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

// ---- models ----

func TestModelsListStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodGet, "/api/v1/models", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsCreateStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodPost, "/api/v1/models", `{"value":"x","name":"X"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsCreateMalformedBody(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodPost, "/api/v1/models", `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestModelsUpdateMalformedID(t *testing.T) {
	d := testDeps(t)
	rec := doReq(t, New(d), http.MethodPut, "/api/v1/models/abc", `{"value":"x","name":"X"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestModelsUpdateMalformedBody(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodPut, "/api/v1/models/1", `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestModelsUpdateStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodPut, "/api/v1/models/1", `{"value":"x","name":"X"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsUpdateRenamesSelectedSettingError(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("old-value", "Old", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodPut, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), `{"value":"new-value","name":"New"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500 (model setting read)", rec.Code)
	}
}

func TestModelsDeleteMalformedID(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodDelete, "/api/v1/models/abc", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestModelsDeleteStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := doReq(t, New(d), http.MethodDelete, "/api/v1/models/1", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestModelsDeleteClearsSelectedSettingError(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("doomed", "Doomed", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doReq(t, New(d), http.MethodDelete, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500 (model setting read)", rec.Code)
	}
}

// ---- git ----

// pushGitConn creates a git connection under the github provider targeting
// repoURL and pushes the wiki through it.
func pushGitConn(t *testing.T, d Deps, repoURL string) *httptest.ResponseRecorder {
	t.Helper()
	p, err := d.Store.SyncProviderBySlug("github")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := syncsvc.EncodeConfig(syncsvc.Config{"token": "t", "repo_url": repoURL})
	ident, _ := syncsvc.EncodeIdentity(syncsvc.Identity{Username: "octo", DisplayName: "Octo", Email: "octo@example.com"})
	conn, err := d.Store.CreateConnection(p.ID, "work", cfg, ident, true)
	if err != nil {
		t.Fatal(err)
	}
	return apiJSON(New(d), http.MethodPost, "/api/v1/sync/connections/"+itoa(conn.ID)+"/push", nil)
}

func TestPushGitInitFailure(t *testing.T) {
	d := testDeps(t)
	// A file squatting on the wiki root makes repo init fail.
	root := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(root, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	d.Wiki = wiki.New(root)
	rec := pushGitConn(t, d, "https://example.com/wiki.git")
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.OK {
		t.Fatalf("expected ok:false from a failing git init, got %+v %s", body, rec.Body.String())
	}
}

func TestPushGitAddsRemote(t *testing.T) {
	// The first push adds the origin remote (none exists yet) and completes.
	d := testDeps(t)
	bare := initBare(t)
	if rec := pushGitConn(t, d, bare); rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	repo, err := git.PlainOpen(d.Wiki.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Remote("origin"); err != nil {
		t.Fatalf("origin remote should exist after the first sync: %v", err)
	}
}

func TestPushGitReplacesRemote(t *testing.T) {
	// A second push with a different URL replaces the origin (set-url path).
	d := testDeps(t)
	bare := initBare(t)
	if rec := pushGitConn(t, d, bare); rec.Code != http.StatusOK {
		t.Fatalf("first status %d: %s", rec.Code, rec.Body.String())
	}
	other := initBare(t)
	if rec := pushGitConn(t, d, other); rec.Code != http.StatusOK {
		t.Fatalf("second status %d: %s", rec.Code, rec.Body.String())
	}
	repo, err := git.PlainOpen(d.Wiki.Root())
	if err != nil {
		t.Fatal(err)
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatal(err)
	}
	urls := remote.Config().URLs
	if len(urls) != 1 || urls[0] != other {
		t.Fatalf("origin URLs after set-url = %v, want [%s]", urls, other)
	}
}

// ---- sync ----

func TestSyncStoreErrors(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	for _, tc := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/sync/providers", ""},
		{http.MethodPost, "/api/v1/sync/providers", `{"name":"X","driver":"github"}`},
		{http.MethodGet, "/api/v1/sync/connections", ""},
		{http.MethodPost, "/api/v1/sync/connections", `{"provider_id":1,"name":"x","config":{}}`},
		{http.MethodGet, "/api/v1/sync/connections/1", ""},
		{http.MethodGet, "/api/v1/sync/connections/1/targets", ""},
		{http.MethodPost, "/api/v1/sync/connections/1/push", ""},
	} {
		if rec := doReq(t, e, tc.method, tc.path, tc.body); rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s on a closed store = %d, want 500", tc.method, tc.path, rec.Code)
		}
	}
}

func TestConnectSyncNotConfigured(t *testing.T) {
	d := testDeps(t)
	d.Sync = nil
	if rec := doReq(t, New(d), http.MethodPost, "/api/v1/sync/connections", `{"provider_id":1,"name":"x","config":{"token":"t"}}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("connect without sync service = %d, want 500", rec.Code)
	}
}

func TestConnectUnknownDriver(t *testing.T) {
	d := testDeps(t)
	if rec := doReq(t, New(d), http.MethodPost, "/api/v1/sync/providers", `{"name":"Bogus","driver":"bogus"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown driver = %d, want 400", rec.Code)
	}
}

// ---- notes / fs ----

func TestNoteUnknownExtensionFallsBackToOctetStream(t *testing.T) {
	d := testDeps(t)
	p := filepath.Join(d.Wiki.Root(), "asset.qzy")
	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes?path=asset.qzy", nil)
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
	e.GET("/api/v1/health", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
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
	e.GET("/api/v1/boom", func(c echo.Context) error { return errors.New("kaboom") })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/boom", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if !strings.Contains(buf.String(), "kaboom") {
		t.Fatalf("request log missing error attr: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "path=/api/v1/boom") {
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
	pid := seedProvider(t, d, "Vendor")
	if _, err := d.Store.CreateModel("m", "M", "", pid); err != nil {
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
