package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/github"
)

// githubStub serves /user and /user/emails for the connect endpoint.
func githubStub(t *testing.T) (*httptest.Server, func(*github.Client) *github.Client) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo,user")
			_, _ = w.Write([]byte(`{"login":"octo","name":"Octo Cat","avatar_url":"https://avatars/u/1","html_url":"https://github.com/octo","created_at":"2018-05-01T10:00:00Z","updated_at":"2026-08-01T10:00:00Z"}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[{"email":"octo@example.com","primary":true,"verified":true}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	base := func(c *github.Client) *github.Client { return c.WithBaseURL(ts.URL) }
	return ts, base
}

func doJSON(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestConnectGitHub(t *testing.T) {
	d := testDeps(t)
	ts, base := githubStub(t)
	d.GitHub.Client = base(github.New(ts.Client()))
	e := New(d)

	rec := doJSON(t, e, http.MethodPost, "/api/github/auth", `{"token":"ghp_secret123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var id githubIdentity
	if err := json.Unmarshal(rec.Body.Bytes(), &id); err != nil {
		t.Fatal(err)
	}
	want := githubIdentity{
		Username: "octo", DisplayName: "Octo Cat", Email: "octo@example.com",
		AvatarURL: "https://avatars/u/1", ProfileURL: "https://github.com/octo",
		Scopes:           "repo,user",
		AccountCreatedAt: "2018-05-01T10:00:00Z", AccountUpdatedAt: "2026-08-01T10:00:00Z",
	}
	if id != want {
		t.Fatalf("identity = %+v, want %+v", id, want)
	}
	if strings.Contains(rec.Body.String(), "ghp_secret123") {
		t.Fatal("response must never contain the token")
	}
	// The token and the account facts are stored.
	a, ok, err := d.GitHub.Repo.Get()
	if err != nil || !ok || a.Token != "ghp_secret123" || a.Username != "octo" ||
		a.ProfileURL != "https://github.com/octo" ||
		a.AccountCreatedAt != "2018-05-01T10:00:00Z" || a.AccountUpdatedAt != "2026-08-01T10:00:00Z" {
		t.Fatalf("stored auth = %+v %v %v", a, ok, err)
	}
}

func TestConnectGitHubRequiresToken(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	for _, body := range []string{`{}`, `{"token":""}`} {
		if rec := doJSON(t, e, http.MethodPost, "/api/github/auth", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status %d, want 400", body, rec.Code)
		}
	}
}

func TestConnectGitHubRejectedToken(t *testing.T) {
	d := testDeps(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(ts.Close)
	d.GitHub.Client = github.New(ts.Client()).WithBaseURL(ts.URL)
	e := New(d)

	rec := doJSON(t, e, http.MethodPost, "/api/github/auth", `{"token":"bad"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "github rejected the token") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestConnectGitHubUpstreamError(t *testing.T) {
	d := testDeps(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(ts.Close)
	d.GitHub.Client = github.New(ts.Client()).WithBaseURL(ts.URL)
	e := New(d)

	rec := doJSON(t, e, http.MethodPost, "/api/github/auth", `{"token":"t"}`)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "internal error") {
		t.Fatalf("status %d body %s, want a generic 500", rec.Code, rec.Body.String())
	}
}

func TestGetGitHubAuth(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// Not connected: 200 with empty strings.
	rec := doJSON(t, e, http.MethodGet, "/api/github/auth", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var id githubIdentity
	if err := json.Unmarshal(rec.Body.Bytes(), &id); err != nil {
		t.Fatal(err)
	}
	if id != (githubIdentity{}) {
		t.Fatalf("identity = %+v, want empty", id)
	}

	// After connect: the identity comes back without the token.
	if err := d.GitHub.Repo.Save(github.Auth{Token: "ghp_x", Username: "octo"}); err != nil {
		t.Fatal(err)
	}
	rec = doJSON(t, e, http.MethodGet, "/api/github/auth", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &id); err != nil {
		t.Fatal(err)
	}
	if id.Username != "octo" || strings.Contains(rec.Body.String(), "ghp_x") {
		t.Fatalf("identity = %+v, body %s", id, rec.Body.String())
	}
}

func TestDisconnectGitHub(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	if err := d.GitHub.Repo.Save(github.Auth{Token: "ghp_x", Username: "octo"}); err != nil {
		t.Fatal(err)
	}

	if rec := doJSON(t, e, http.MethodDelete, "/api/github/auth", ""); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if _, ok, err := d.GitHub.Repo.Get(); err != nil || ok {
		t.Fatalf("auth still present after disconnect: %v %v", ok, err)
	}
	// Disconnecting again is fine.
	if rec := doJSON(t, e, http.MethodDelete, "/api/github/auth", ""); rec.Code != http.StatusOK {
		t.Fatalf("second disconnect status %d", rec.Code)
	}
}

func TestListGitHubRepos(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// Not connected: an empty list.
	rec := doJSON(t, e, http.MethodGet, "/api/github/repos", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"repos":[]`) {
		t.Fatalf("not connected: status %d body %s", rec.Code, rec.Body.String())
	}

	// Connected: repos are fetched with the stored token, never echoed back.
	if err := d.GitHub.Repo.Save(github.Auth{Token: "ghp_x", Username: "octo"}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repos" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer ghp_x" {
			t.Errorf("repos request must carry the stored token")
		}
		_, _ = w.Write([]byte(`[{"full_name":"octo/wiki","clone_url":"https://github.com/octo/wiki.git","private":true}]`))
	}))
	t.Cleanup(ts.Close)
	d.GitHub.Client = github.New(ts.Client()).WithBaseURL(ts.URL)

	rec = doJSON(t, e, http.MethodGet, "/api/github/repos", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "octo/wiki") ||
		strings.Contains(rec.Body.String(), "ghp_x") {
		t.Fatalf("connected: status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestListGitHubReposRejectedToken(t *testing.T) {
	d := testDeps(t)
	if err := d.GitHub.Repo.Save(github.Auth{Token: "ghp_x", Username: "octo"}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(ts.Close)
	d.GitHub.Client = github.New(ts.Client()).WithBaseURL(ts.URL)
	e := New(d)

	rec := doJSON(t, e, http.MethodGet, "/api/github/repos", "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "github rejected the token") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
