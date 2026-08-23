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
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func getSettingsReq(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}) settingsDTO {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
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
	if got.WikiPath != "~/.thoth/wiki" || got.Model != "" || got.RepoURL != "" || got.SyncEnabled || got.ContextInjection {
		t.Fatalf("seeded settings = %+v", got)
	}

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/other/wiki","model":"claude-sonnet-5","repo_url":"https://github.com/x/w.git","sync_enabled":true,"context_injection":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if !called || savedPath != "/tmp/other/wiki" {
		t.Fatalf("callback not fired with the wiki path: %v %q", called, savedPath)
	}

	got = getSettingsReq(t, e)
	if got.WikiPath != "/tmp/other/wiki" || got.Model != "claude-sonnet-5" || got.RepoURL != "https://github.com/x/w.git" || !got.SyncEnabled || !got.ContextInjection {
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
	if on, err := d.Settings.ContextInjection(); err != nil || !on {
		t.Fatalf("stored context_injection = %v/%v, want true/nil", on, err)
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

func TestSettingsProvidersRoundTrip(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateModel("deepseek-v4-flash", "V4 Flash", "fastest", "DeepSeek"); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	// Seeded: the provider appears with no key and no base url.
	got := getSettingsReq(t, e)
	ds, ok := got.Providers["DeepSeek"]
	if !ok || ds.HasAPIKey || ds.BaseURL != "" {
		t.Fatalf("seeded DeepSeek config = %+v (present=%v)", ds, ok)
	}

	// PUT writes the per-provider key + base url; GET reports has_api_key
	// but never the key itself.
	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":false,
		"providers":{"DeepSeek":{"api_key":"ds-secret","base_url":"https://api.deepseek.com"}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.ProviderAPIKeyKey("DeepSeek")); err != nil || v != "ds-secret" {
		t.Fatalf("stored provider api key = %q/%v", v, err)
	}
	if v, _, err := d.Settings.Setting(settings.ProviderBaseURLKey("DeepSeek")); err != nil || v != "https://api.deepseek.com" {
		t.Fatalf("stored provider base url = %q/%v", v, err)
	}
	got = getSettingsReq(t, e)
	ds, ok = got.Providers["DeepSeek"]
	if !ok || !ds.HasAPIKey || ds.BaseURL != "https://api.deepseek.com" || ds.APIKey != "" {
		t.Fatalf("after PUT DeepSeek config = %+v (present=%v)", ds, ok)
	}

	// An empty api key leaves the stored key; an empty base url clears back
	// to the default endpoint.
	rec = putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":false,
		"providers":{"DeepSeek":{"api_key":"","base_url":""}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.ProviderAPIKeyKey("DeepSeek")); err != nil || v != "ds-secret" {
		t.Fatalf("provider api key changed on empty PUT: %q/%v", v, err)
	}
	if v, _, err := d.Settings.Setting(settings.ProviderBaseURLKey("DeepSeek")); err != nil || v != "" {
		t.Fatalf("provider base url not cleared: %q/%v", v, err)
	}
}

func TestSettingsProviderKeyNeverEchoed(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateModel("gpt-5.6-mini", "Mini", "fast", "OpenAI"); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	if rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":false,
		"providers":{"OpenAI":{"api_key":"sk-openai-secret","base_url":""}}}`); rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "sk-openai-secret") {
		t.Fatalf("GET /api/v1/settings echoed the provider api key: %s", rec.Body.String())
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

func TestSettingsWikiFoldersRoundTrip(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// Seeded: no custom set.
	if got := getSettingsReq(t, e); len(got.WikiFolders) != 0 {
		t.Fatalf("seeded wiki_folders = %v, want empty", got.WikiFolders)
	}

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":false,"wiki_folders":["journal","recipes"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if v, _, err := d.Settings.Setting(settings.KeyWikiFolders); err != nil || v != "journal,recipes" {
		t.Fatalf("stored wiki_folders = %q/%v, want journal,recipes", v, err)
	}
	if got := getSettingsReq(t, e); len(got.WikiFolders) != 2 || got.WikiFolders[0] != "journal" || got.WikiFolders[1] != "recipes" {
		t.Fatalf("GET wiki_folders = %v, want [journal recipes]", got.WikiFolders)
	}

	// An empty set clears the key back to the default behavior.
	rec = putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","repo_url":"","sync_enabled":false,"wiki_folders":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if got := getSettingsReq(t, e); len(got.WikiFolders) != 0 {
		t.Fatalf("cleared wiki_folders = %v, want empty", got.WikiFolders)
	}
}

func TestConversationsEndpoints(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader([]byte(`{"title":"first"}`)))
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

	req = httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
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
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/conversations/"+id, nil)
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
