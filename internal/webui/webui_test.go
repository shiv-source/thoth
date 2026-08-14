package webui

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	Register(e, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return e
}

func TestRegisterServesIndexAtRoot(t *testing.T) {
	e := newTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("expected index.html at root, got %q", rec.Body.String())
	}
}

func TestRegisterReturns404ForUnknownAPIPaths(t *testing.T) {
	e := newTestEcho()
	// Unknown API paths are a client bug or a stale URL — they must stay a
	// JSON 404, never index.html.
	req := httptest.NewRequest(http.MethodGet, "/api/nope", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("API 404 must not serve index.html, got %q", rec.Body.String())
	}
}

func TestRegisterServesExistingAsset(t *testing.T) {
	e := newTestEcho()
	// index.html is a real embedded file; the asset dir holds real files.
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestRegisterFallsBackToIndexForMissingPaths(t *testing.T) {
	e := newTestEcho()
	// Unknown deep links (client-side routes) must resolve to index.html.
	for _, p := range []string{"/notes/some/note", "/assets/missing.js", "/api-adjacent-path", "/chat/123e4567-e89b-4122-a456-426614174000"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", p, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "<!doctype html>") {
			t.Fatalf("%s: expected index.html fallback, got %q", p, rec.Body.String())
		}
	}
}
