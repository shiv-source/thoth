package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/github"
)

func TestSettingsRoundTripAndCallback(t *testing.T) {
	d := testDeps(t)
	d.ConfigPath = filepath.Join(t.TempDir(), "config.toml")

	called := false
	var saved config.Config
	d.OnSettingsSaved = func(c config.Config) error { called = true; saved = c; return nil }

	e := New(d)

	// GET returns defaults
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status %d", rec.Code)
	}
	var got config.Config
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Port != config.Default().Port {
		t.Fatalf("unexpected defaults: %+v", got)
	}

	// PUT saves and fires the callback
	body := `{"wiki_path":"/tmp/other/wiki","host":"127.0.0.1","port":9444,"claude_bin":"","permission_mode":"acceptEdits","model":""}`
	req = httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if !called || saved.WikiPath != "/tmp/other/wiki" || saved.Port != 9444 {
		t.Fatalf("callback not fired with saved config: %v %+v", called, saved)
	}

	// reloaded from disk matches
	loaded, err := config.Load(d.ConfigPath)
	if err != nil || loaded.Port != 9444 {
		t.Fatalf("config not persisted: %v %+v", err, loaded)
	}
}

func TestSettingsRejectsBadPort(t *testing.T) {
	d := testDeps(t)
	d.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	e := New(d)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(`{"port":70000}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSettingsRejectsMalformedBody(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(`{not json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed body, got %d", rec.Code)
	}
}

func TestSettingsRejectsMissingFields(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(`{"wiki_path":"","host":"","port":8333}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty wiki_path/host, got %d", rec.Code)
	}
}

func TestSettingsSaveError(t *testing.T) {
	d := testDeps(t)
	// ConfigPath whose parent is a regular file: config.Save must fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	d.ConfigPath = filepath.Join(blocker, "config.toml")
	e := New(d)

	body := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when save fails, got %d: %s", rec.Code, rec.Body.String())
	}
	// A failed save must not apply the new settings to the live config.
	if got := d.Config.WikiPath; got != config.Default().WikiPath {
		t.Fatalf("in-memory config mutated after failed save: wiki_path = %q", got)
	}
}

func TestSettingsCallbackError(t *testing.T) {
	d := testDeps(t)
	d.OnSettingsSaved = func(config.Config) error { return errors.New("callback boom") }
	e := New(d)

	body := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when callback fails, got %d: %s", rec.Code, rec.Body.String())
	}
	// A failed callback must not apply the new settings to the live config.
	if got := d.Config.WikiPath; got != config.Default().WikiPath {
		t.Fatalf("in-memory config mutated after failed callback: wiki_path = %q, want %q", got, config.Default().WikiPath)
	}
	if got := d.Config.Port; got != config.Default().Port {
		t.Fatalf("in-memory config mutated after failed callback: port = %d", got)
	}
}

func TestSettingsCallbackRunsBeforeSave(t *testing.T) {
	d := testDeps(t)
	d.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	// The save fails (parent path is a regular file) but the callback must
	// still have run first, so a wiki-path change is applied even when the
	// disk write fails; a retry then self-heals.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	d.ConfigPath = filepath.Join(blocker, "config.toml")

	called := false
	d.OnSettingsSaved = func(c config.Config) error { called = true; return nil }

	e := New(d)
	body := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when save fails, got %d", rec.Code)
	}
	if !called {
		t.Fatal("callback must run before the save")
	}
	// The in-memory config stays on the old value: the swap happens only
	// after a successful save.
	if got := d.Config.WikiPath; got != config.Default().WikiPath {
		t.Fatalf("in-memory config mutated after failed save: wiki_path = %q", got)
	}
}

func TestSettingsWithoutConfigPathSkipsSave(t *testing.T) {
	d := testDeps(t) // ConfigPath is "" by default
	e := New(d)

	body := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestConversationsEndpoints(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	req := httptest.NewRequest(http.MethodPost, "/api/conversations", bytes.NewReader([]byte(`{"title":"first"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("bad create response: %v %s", err, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(created.ID)) {
		t.Fatalf("list missing conversation: %d %s", rec.Code, rec.Body.String())
	}
}

func seedGitHubAuth(t *testing.T, d Deps, repoURL string) {
	t.Helper()
	if err := d.GitHub.Repo.Save(github.Auth{Token: "ghp_x", Username: "octo"}); err != nil {
		t.Fatal(err)
	}
	if err := d.GitHub.Repo.SetRepoURL(repoURL); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsRepoURLRoundTrip(t *testing.T) {
	d := testDeps(t)
	d.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	seedGitHubAuth(t, d, "")
	e := New(d)

	// GET reads repo_url from the DB.
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var got settingsDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.RepoURL != "" {
		t.Fatalf("repo_url = %q, want empty before save", got.RepoURL)
	}

	// PUT persists it to the DB — never to config.toml.
	body := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333,"repo_url":"https://github.com/x/w.git"}`
	req = httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.RepoURL != "https://github.com/x/w.git" {
		t.Fatalf("repo_url = %q after save", got.RepoURL)
	}
	raw, err := os.ReadFile(d.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "repo_url") || strings.Contains(string(raw), "github.com") {
		t.Fatalf("config.toml must not store the repo URL:\n%s", raw)
	}
}

func TestSettingsRepoURLWithoutAuth(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	base := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333`

	// An empty repo_url is nothing to store: ordinary saves keep working.
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(base+`,"repo_url":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("empty repo_url without auth: status %d: %s", rec.Code, rec.Body.String())
	}

	// A non-empty repo_url without a connected account is a client error.
	req = httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(base+`,"repo_url":"https://github.com/x/w.git"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "connect GitHub first") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestSettingsRepoURLClears(t *testing.T) {
	d := testDeps(t)
	seedGitHubAuth(t, d, "https://github.com/x/w.git")
	e := New(d)

	body := `{"wiki_path":"/tmp/wiki","host":"127.0.0.1","port":8333,"repo_url":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	a, ok, err := d.GitHub.Repo.Get()
	if err != nil || !ok || a.RepoURL != "" {
		t.Fatalf("repo_url not cleared: %+v %v %v", a, ok, err)
	}
}
