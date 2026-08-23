package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	syncsvc "github.com/shiv-source/thoth/internal/sync"
)

// apiJSON issues a JSON request against the server and returns the recorder.
func apiJSON(e http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// githubStub serves the github identity + repos endpoints the driver calls.
func githubStub(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"login": "octo", "name": "Octo Cat", "email": "o@e.com",
				"avatar_url": "https://a", "html_url": "https://github.com/octo",
			})
		case "/user/emails":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"email": "o@e.com", "primary": true, "verified": true}})
		case "/user/repos":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"full_name": "octo/wiki", "clone_url": "https://github.com/octo/wiki.git", "private": true, "description": ""},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSyncProvidersList(t *testing.T) {
	d := testDeps(t)
	rec := apiJSON(New(d), http.MethodGet, "/api/v1/sync/providers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Providers []syncProviderDTO `json:"providers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Providers) != 4 {
		t.Fatalf("expected 4 built-in providers, got %+v", body.Providers)
	}
	local := body.Providers[len(body.Providers)-1]
	if local.Slug != "local" || !local.Protected || local.Kind != syncsvc.KindLocal {
		t.Fatalf("local provider wrong: %+v", local)
	}
	if len(local.Fields) != 1 || local.Fields[0].Key != "path" {
		t.Fatalf("local fields wrong: %+v", local.Fields)
	}
	github := body.Providers[0]
	if github.Slug != "github" || github.Kind != syncsvc.KindGit || len(github.Fields) == 0 {
		t.Fatalf("github provider wrong: %+v", github)
	}
}

func TestSyncProviderCRUD(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	// Create a custom provider; the slug derives from the name.
	rec := apiJSON(e, http.MethodPost, "/api/v1/sync/providers", syncProviderInput{Name: "GitHub Enterprise", Driver: "github", BaseURL: "https://ghe.example.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}
	var created syncProviderDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Slug != "githubenterprise" || created.Driver != "github" {
		t.Fatalf("created provider wrong: %+v", created)
	}
	// Update.
	rec = apiJSON(e, http.MethodPut, "/api/v1/sync/providers/"+itoa(created.ID), syncProviderInput{Name: "GH Ent", Driver: "github", BaseURL: "https://ghe2.example.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status %d: %s", rec.Code, rec.Body.String())
	}
	// Delete.
	rec = apiJSON(e, http.MethodDelete, "/api/v1/sync/providers/"+itoa(created.ID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSyncProviderProtected(t *testing.T) {
	d := testDeps(t)
	local, err := d.Store.SyncProviderBySlug("local")
	if err != nil {
		t.Fatal(err)
	}
	rec := apiJSON(New(d), http.MethodDelete, "/api/v1/sync/providers/"+itoa(local.ID), nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("delete protected provider = %d, want 403", rec.Code)
	}
}

// TestSyncAPIErrorPaths covers the sentinel error branches: 400 bad input,
// 403 protected, 404 missing, 409 in-use.
func TestSyncAPIErrorPaths(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// 400: create without a name / driver.
	if rec := apiJSON(e, http.MethodPost, "/api/v1/sync/providers", syncProviderInput{Driver: "github"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("create without name = %d, want 400", rec.Code)
	}
	// 404: update/delete a missing provider.
	if rec := apiJSON(e, http.MethodPut, "/api/v1/sync/providers/999999", syncProviderInput{Name: "X", Driver: "github"}); rec.Code != http.StatusNotFound {
		t.Fatalf("update missing provider = %d, want 404", rec.Code)
	}
	if rec := apiJSON(e, http.MethodDelete, "/api/v1/sync/providers/999999", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing provider = %d, want 404", rec.Code)
	}
	// 403: update a protected provider.
	local, err := d.Store.SyncProviderBySlug("local")
	if err != nil {
		t.Fatal(err)
	}
	if rec := apiJSON(e, http.MethodPut, "/api/v1/sync/providers/"+itoa(local.ID), syncProviderInput{Name: "Renamed", Driver: "local"}); rec.Code != http.StatusForbidden {
		t.Fatalf("update protected provider = %d, want 403", rec.Code)
	}
	// 409: delete a provider that still has connections.
	p, err := d.Store.CreateSyncProvider("inuse", "In Use", "github", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Store.CreateConnection(p.ID, "mine", `{"token":"t"}`, `{"username":"u"}`, true); err != nil {
		t.Fatal(err)
	}
	if rec := apiJSON(e, http.MethodDelete, "/api/v1/sync/providers/"+itoa(p.ID), nil); rec.Code != http.StatusConflict {
		t.Fatalf("delete provider in use = %d, want 409", rec.Code)
	}

	// 400: connect with a missing required field.
	ghProvider, err := d.Store.SyncProviderBySlug("github")
	if err != nil {
		t.Fatal(err)
	}
	if rec := apiJSON(e, http.MethodPost, "/api/v1/sync/connections",
		connectInput{ProviderID: ghProvider.ID, Name: "x", Config: map[string]string{}}); rec.Code != http.StatusBadRequest {
		t.Fatalf("connect missing token = %d, want 400", rec.Code)
	}
	// 404: connect to a missing provider.
	if rec := apiJSON(e, http.MethodPost, "/api/v1/sync/connections",
		connectInput{ProviderID: 999999, Name: "x", Config: map[string]string{"token": "t"}}); rec.Code != http.StatusNotFound {
		t.Fatalf("connect missing provider = %d, want 404", rec.Code)
	}
	// 404: get/update/targets/push/active on a missing connection.
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/sync/connections/999999", ""},
		{http.MethodPut, "/api/v1/sync/connections/999999", `{}`},
		{http.MethodDelete, "/api/v1/sync/connections/999999", ""},
		{http.MethodGet, "/api/v1/sync/connections/999999/targets", ""},
		{http.MethodPost, "/api/v1/sync/connections/999999/push", ""},
		{http.MethodPost, "/api/v1/sync/connections/999999/active", ""},
	} {
		rec := apiJSON(e, tc.method, tc.path, tc.body)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s = %d, want 404", tc.method, tc.path, rec.Code)
		}
	}
}

func TestConnectGitHubRoundTrip(t *testing.T) {
	srv := githubStub(t)
	d := testDeps(t)
	d.Sync = syncsvc.NewService(d.Store, srv.Client())
	// A provider row bound to the stub endpoint.
	p, err := d.Store.CreateSyncProvider("ghe", "GH Ent", "github", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)

	rec := apiJSON(e, http.MethodPost, "/api/v1/sync/connections",
		connectInput{ProviderID: p.ID, Name: "work", Config: map[string]string{"token": "good"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("connect status %d: %s", rec.Code, rec.Body.String())
	}
	var conn connectionDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.Identity == nil || conn.Identity.Username != "octo" || conn.Identity.Email != "o@e.com" {
		t.Fatalf("stored identity wrong: %+v", conn.Identity)
	}
	// Secrets never echo: has_token true, no token value.
	if conn.Config["has_token"] != true {
		t.Fatalf("has_token = %v, want true", conn.Config["has_token"])
	}
	if _, leaked := conn.Config["token"]; leaked {
		t.Fatalf("token leaked on the wire: %+v", conn.Config)
	}

	// The list view agrees and targets resolve from the stored token.
	rec = apiJSON(e, http.MethodGet, "/api/v1/sync/connections", nil)
	var list struct {
		Connections []connectionDTO `json:"connections"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Connections) != 1 {
		t.Fatalf("list connections: %v %s", err, rec.Body.String())
	}
	rec = apiJSON(e, http.MethodGet, "/api/v1/sync/connections/"+itoa(conn.ID)+"/targets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("targets status %d: %s", rec.Code, rec.Body.String())
	}
	var targets struct {
		Targets []syncsvc.Target `json:"targets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &targets); err != nil || len(targets.Targets) != 1 {
		t.Fatalf("targets wrong: %v %s", err, rec.Body.String())
	}
}

func TestConnectRejectsBadToken(t *testing.T) {
	srv := githubStub(t)
	d := testDeps(t)
	d.Sync = syncsvc.NewService(d.Store, srv.Client())
	p, err := d.Store.CreateSyncProvider("ghe", "GH Ent", "github", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	rec := apiJSON(New(d), http.MethodPost, "/api/v1/sync/connections",
		connectInput{ProviderID: p.ID, Name: "work", Config: map[string]string{"token": "bad"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("rejected token status = %d, want 400", rec.Code)
	}
}

// initBare initializes an empty bare repository and returns its path — a
// local filesystem remote, so no git binary is needed.
func initBare(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, true); err != nil {
		t.Fatalf("init bare: %v", err)
	}
	return dir
}

// TestPushGit replaces the pre-cutover /api/v1/git/setup test: a git
// connection with a local bare remote pushes the wiki and records the outcome
// on the connection row.
func TestPushGit(t *testing.T) {
	d := testDeps(t)
	bare := initBare(t)
	// A connection under the github provider carrying token + target +
	// committer identity.
	p, err := d.Store.SyncProviderBySlug("github")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := syncsvc.EncodeConfig(syncsvc.Config{"token": "t", "repo_url": bare})
	ident, _ := syncsvc.EncodeIdentity(syncsvc.Identity{Username: "octo", DisplayName: "Octo", Email: "octo@example.com"})
	conn, err := d.Store.CreateConnection(p.ID, "work", cfg, ident, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.Wiki.Root(), "hello.md"),
		[]byte("---\ntitle: Hi\n---\n\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := apiJSON(e, http.MethodPost, "/api/v1/sync/connections/"+itoa(conn.ID)+"/push", nil)
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true, got %v %s", err, rec.Body.String())
	}

	repo, err := git.PlainOpen(d.Wiki.Root())
	if err != nil {
		t.Fatal(err)
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatalf("origin remote: %v", err)
	}
	if urls := remote.Config().URLs; len(urls) != 1 || urls[0] != bare {
		t.Fatalf("origin URLs = %v, want [%s]", urls, bare)
	}
	bareRepo, err := git.PlainOpen(bare)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bareRepo.Reference(plumbing.NewBranchReferenceName("main"), false); err != nil {
		t.Fatalf("bare remote missing main: %v", err)
	}

	got, err := d.Store.Connection(conn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastSyncedAt == "" || got.LastError != "" {
		t.Fatalf("sync state after success: last=%q err=%q", got.LastSyncedAt, got.LastError)
	}
}

// TestPushGitReportsSanitizedFailure verifies a failed push is ok:false with a
// sanitized message (no remote URL echoed) and records the error.
func TestPushGitReportsSanitizedFailure(t *testing.T) {
	d := testDeps(t)
	p, err := d.Store.SyncProviderBySlug("github")
	if err != nil {
		t.Fatal(err)
	}
	notARepo := filepath.Join(t.TempDir(), "not-a-repo")
	cfg, _ := syncsvc.EncodeConfig(syncsvc.Config{"token": "t", "repo_url": notARepo})
	ident, _ := syncsvc.EncodeIdentity(syncsvc.Identity{Username: "octo", DisplayName: "Octo", Email: "octo@example.com"})
	conn, err := d.Store.CreateConnection(p.ID, "work", cfg, ident, true)
	if err != nil {
		t.Fatal(err)
	}
	rec := apiJSON(New(d), http.MethodPost, "/api/v1/sync/connections/"+itoa(conn.ID)+"/push", nil)
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
	if strings.Contains(body.Error, notARepo) || strings.Contains(body.Error, "token") {
		t.Fatalf("error must not echo the remote URL or credentials: %q", body.Error)
	}
	got, _ := d.Store.Connection(conn.ID)
	if got.LastSyncedAt != "" || got.LastError == "" {
		t.Fatalf("sync state after failure: last=%q err=%q", got.LastSyncedAt, got.LastError)
	}
}

func TestPushLocal(t *testing.T) {
	d := testDeps(t)
	local, err := d.Store.EnsureLocalBackup()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.Wiki.Root(), "hello.md"),
		[]byte("---\ntitle: Hi\n---\n\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	backup := t.TempDir()
	e := New(d)
	// Set the backup folder on the protected local connection.
	rec := apiJSON(e, http.MethodPut, "/api/v1/sync/connections/"+itoa(local.ID),
		updateConnectionInput{Config: map[string]string{"path": backup}})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status %d: %s", rec.Code, rec.Body.String())
	}
	rec = apiJSON(e, http.MethodPost, "/api/v1/sync/connections/"+itoa(local.ID)+"/push", nil)
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !body.OK {
		t.Fatalf("expected ok:true, got %v %s", err, rec.Body.String())
	}
	entries, err := os.ReadDir(backup)
	if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "thoth-wiki-") {
		t.Fatalf("backup entries = %+v / %v", entries, err)
	}
	got, _ := d.Store.Connection(local.ID)
	if got.LastSyncedAt == "" {
		t.Fatalf("local sync last_synced_at not recorded: %+v", got)
	}
}

func TestLocalBackupProtectedConnection(t *testing.T) {
	d := testDeps(t)
	local, err := d.Store.EnsureLocalBackup()
	if err != nil {
		t.Fatal(err)
	}
	if !local.Protected {
		t.Fatalf("local backup must be protected: %+v", local)
	}
	rec := apiJSON(New(d), http.MethodDelete, "/api/v1/sync/connections/"+itoa(local.ID), nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("delete protected connection = %d, want 403", rec.Code)
	}
}

func TestDisconnectClearsActive(t *testing.T) {
	d := testDeps(t)
	p, err := d.Store.SyncProviderBySlug("github")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := d.Store.CreateConnection(p.ID, "work", `{"token":"t"}`, `{"username":"octo"}`, true)
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)
	if rec := apiJSON(e, http.MethodPost, "/api/v1/sync/connections/"+itoa(conn.ID)+"/active", nil); rec.Code != http.StatusOK {
		t.Fatalf("set active status %d", rec.Code)
	}
	rec := apiJSON(e, http.MethodDelete, "/api/v1/sync/connections/"+itoa(conn.ID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("disconnect status %d: %s", rec.Code, rec.Body.String())
	}
	if v, found, err := d.Settings.Setting("sync_active_connection"); err != nil || !found || v != "" {
		t.Fatalf("active connection not cleared: %q/%v/%v", v, found, err)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
