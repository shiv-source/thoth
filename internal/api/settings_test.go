package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/settings"
)

func putSettingsReq(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func getSettingsReq(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}) settingsDTO {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status %d: %s", rec.Code, rec.Body.String())
	}
	var got settingsDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSettingsRoundTripAndCallback(t *testing.T) {
	d := testDeps(t)
	called := false
	var savedPath string
	d.OnSettingsSaved = func(wikiPath string) error { called = true; savedPath = wikiPath; return nil }
	e := New(d)

	// GET returns the migration-seeded defaults.
	got := getSettingsReq(t, e)
	if got.WikiPath != "~/.thoth/wiki" || got.Model != "" || got.RepoURL != "" || got.SyncEnabled {
		t.Fatalf("seeded settings = %+v", got)
	}

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/other/wiki","model":"claude-sonnet-5","repo_url":"https://github.com/x/w.git","sync_enabled":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if !called || savedPath != "/tmp/other/wiki" {
		t.Fatalf("callback not fired with the wiki path: %v %q", called, savedPath)
	}

	got = getSettingsReq(t, e)
	if got.WikiPath != "/tmp/other/wiki" || got.Model != "claude-sonnet-5" || got.RepoURL != "https://github.com/x/w.git" || !got.SyncEnabled {
		t.Fatalf("after PUT: %+v", got)
	}
	// The values live in the settings table.
	if v, found, err := d.Settings.Setting(settings.KeyWikiPath); err != nil || !found || v != "/tmp/other/wiki" {
		t.Fatalf("stored wiki_path = %q/%v/%v", v, found, err)
	}
	if v, found, err := d.Settings.Setting(settings.KeyModel); err != nil || !found || v != "claude-sonnet-5" {
		t.Fatalf("stored model = %q/%v/%v", v, found, err)
	}
	if v, found, err := d.Settings.Setting(settings.KeyRepoURL); err != nil || !found || v != "https://github.com/x/w.git" {
		t.Fatalf("stored repo_url = %q/%v/%v", v, found, err)
	}
	if on, err := d.Settings.SyncEnabled(); err != nil || !on {
		t.Fatalf("stored sync_enabled = %v/%v, want true/nil", on, err)
	}
}

func TestSettingsRejectsMalformedBody(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	if rec := putSettingsReq(t, e, `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed body, got %d", rec.Code)
	}
}

func TestSettingsRejectsMissingFields(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	rec := putSettingsReq(t, e, `{"wiki_path":"","repo_url":"","sync_enabled":false}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "wiki_path is required") {
		t.Fatalf("expected 400 for empty wiki_path, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSettingsCallbackError(t *testing.T) {
	d := testDeps(t)
	d.OnSettingsSaved = func(string) error { return errors.New("callback boom") }
	e := New(d)

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/other/wiki","repo_url":"","sync_enabled":false}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when callback fails, got %d: %s", rec.Code, rec.Body.String())
	}
	// A failed callback must not persist the new wiki path.
	if v, _, err := d.Settings.Setting(settings.KeyWikiPath); err != nil || v != "~/.thoth/wiki" {
		t.Fatalf("wiki_path mutated after failed callback: %q/%v", v, err)
	}
}

func TestSettingsSetErrorCallbackStillRan(t *testing.T) {
	d := testDeps(t)
	called := false
	d.OnSettingsSaved = func(string) error { called = true; return nil }
	if err := d.Settings.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/other/wiki","repo_url":"","sync_enabled":false}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when the settings write fails, got %d", rec.Code)
	}
	// The callback runs before the writes (self-heal on retry).
	if !called {
		t.Fatal("callback must run before the settings write")
	}
}

func TestSettingsAPIKeyRoundTrip(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// Seeded: no key saved yet.
	if got := getSettingsReq(t, e); got.HasAPIKey {
		t.Fatalf("seeded has_api_key = true, want false")
	}

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","model":"","repo_url":"","sync_enabled":false,"api_key":"sk-secret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.KeyAPIKey); err != nil || v != "sk-secret" {
		t.Fatalf("stored api_key = %q/%v", v, err)
	}
	if got := getSettingsReq(t, e); !got.HasAPIKey {
		t.Fatalf("has_api_key = false after saving a key")
	}

	// An empty api_key leaves the stored key untouched (the UI omits it
	// unless the user typed a new one).
	rec = putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","model":"","repo_url":"","sync_enabled":false,"api_key":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.KeyAPIKey); err != nil || v != "sk-secret" {
		t.Fatalf("api_key changed on empty PUT: %q/%v", v, err)
	}
}

func TestSettingsAPIKeyNeverEchoed(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	if rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","model":"","repo_url":"","sync_enabled":false,"api_key":"sk-echo-check"}`); rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "sk-echo-check") {
		t.Fatalf("GET /api/settings echoed the api key: %s", rec.Body.String())
	}
}

func TestSettingsRepoURLWithoutAuthStored(t *testing.T) {
	// The old "connect GitHub first" gate is gone: the KV table accepts the
	// URL regardless of a github_auth row.
	d := testDeps(t)
	e := New(d)
	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"https://github.com/x/w.git","sync_enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT without auth: %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.KeyRepoURL); err != nil || v != "https://github.com/x/w.git" {
		t.Fatalf("stored repo_url = %q/%v", v, err)
	}
}

func TestSettingsSyncEnabledRoundTrip(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	for _, want := range []bool{true, false} {
		body := `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":` + map[bool]string{true: "true", false: "false"}[want] + `}`
		if rec := putSettingsReq(t, e, body); rec.Code != http.StatusOK {
			t.Fatalf("PUT sync_enabled=%v: %d: %s", want, rec.Code, rec.Body.String())
		}
		if got := getSettingsReq(t, e); got.SyncEnabled != want {
			t.Fatalf("sync_enabled = %v, want %v", got.SyncEnabled, want)
		}
	}
}

func TestSettingsRepoURLClears(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	if rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"https://github.com/x/w.git","sync_enabled":false}`); rec.Code != http.StatusOK {
		t.Fatalf("seed PUT: %d: %s", rec.Code, rec.Body.String())
	}
	if rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":false}`); rec.Code != http.StatusOK {
		t.Fatalf("clear PUT: %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.KeyRepoURL); err != nil || v != "" {
		t.Fatalf("repo_url not cleared: %q/%v", v, err)
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

func TestDeleteConversationEndpoint(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	id, err := d.Store.CreateConversation("bye")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if convs, err := d.Store.ListConversations(); err != nil || len(convs) != 0 {
		t.Fatalf("conversation not deleted: %v %+v", err, convs)
	}
	// Idempotent.
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("second delete status %d", rec.Code)
	}
}
