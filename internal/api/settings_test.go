package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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
