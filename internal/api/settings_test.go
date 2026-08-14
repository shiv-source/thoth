package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/config"
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
