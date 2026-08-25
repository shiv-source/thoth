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
	if got.WikiPath != "~/.thoth/wiki" || got.Model != "" || got.ContextInjection {
		t.Fatalf("seeded settings = %+v", got)
	}

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/other/wiki","model":"claude-sonnet-5","context_injection":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if !called || savedPath != "/tmp/other/wiki" {
		t.Fatalf("callback not fired with the wiki path: %v %q", called, savedPath)
	}

	got = getSettingsReq(t, e)
	if got.WikiPath != "/tmp/other/wiki" || got.Model != "claude-sonnet-5" || !got.ContextInjection {
		t.Fatalf("after PUT: %+v", got)
	}
	// The values live in the settings table.
	if v, found, err := d.Settings.Setting(settings.KeyWikiPath); err != nil || !found || v != "/tmp/other/wiki" {
		t.Fatalf("stored wiki_path = %q/%v/%v", v, found, err)
	}
	if v, found, err := d.Settings.Setting(settings.KeyModel); err != nil || !found || v != "claude-sonnet-5" {
		t.Fatalf("stored model = %q/%v/%v", v, found, err)
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

func TestSettingsRepoURLWithoutAuthStored(t *testing.T) {
	// The old "connect GitHub first" gate is gone: sync state lives on
	// connections, so the settings payload carries no repo_url at all.
	d := testDeps(t)
	e := New(d)
	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT: %d: %s", rec.Code, rec.Body.String())
	}
	if got := getSettingsReq(t, e); got.WikiPath != "/tmp/wiki" {
		t.Fatalf("wiki_path = %q, want /tmp/wiki", got.WikiPath)
	}
}

func TestSettingsRepoURLClears(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	if rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki"}`); rec.Code != http.StatusOK {
		t.Fatalf("PUT: %d: %s", rec.Code, rec.Body.String())
	}
	if got := getSettingsReq(t, e); got.WikiPath != "/tmp/wiki" {
		t.Fatalf("wiki_path = %q, want /tmp/wiki", got.WikiPath)
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

func TestSettingsConversationRetentionDays(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// Seeded: no stored key, so the documented default is reported.
	if got := getSettingsReq(t, e); got.ConversationRetentionDays != settings.DefaultRetentionDays {
		t.Fatalf("seeded retention = %d, want %d", got.ConversationRetentionDays, settings.DefaultRetentionDays)
	}

	rec := putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","conversation_retention_days":30}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if got := getSettingsReq(t, e); got.ConversationRetentionDays != 30 {
		t.Fatalf("stored retention = %d, want 30", got.ConversationRetentionDays)
	}
	if v, _, err := d.Settings.Setting(settings.KeyConversationRetentionDays); err != nil || v != "30" {
		t.Fatalf("stored key = %q/%v, want 30/nil", v, err)
	}

	// Zero disables auto-delete.
	rec = putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","conversation_retention_days":0}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("zero PUT status %d: %s", rec.Code, rec.Body.String())
	}
	if got := getSettingsReq(t, e); got.ConversationRetentionDays != 0 {
		t.Fatalf("disabled retention = %d, want 0", got.ConversationRetentionDays)
	}

	// Negative values are rejected at the boundary.
	rec = putSettingsReq(t, e, `{"wiki_path":"/tmp/wiki","conversation_retention_days":-1}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "conversation_retention_days cannot be negative") {
		t.Fatalf("negative retention: %d: %s", rec.Code, rec.Body.String())
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
