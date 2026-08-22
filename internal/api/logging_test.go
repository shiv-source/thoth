package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// newLoggingServer builds a server whose request logs land in buf.
func newLoggingServer(t *testing.T, buf *bytes.Buffer) *echo.Echo {
	t.Helper()
	d := testDeps(t)
	d.Log = slog.New(slog.NewTextHandler(buf, nil))
	return New(d)
}

func TestRequestLogsAPIPaths(t *testing.T) {
	var buf bytes.Buffer
	e := newLoggingServer(t, &buf)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line := buf.String()
	for _, want := range []string{"msg=request", "method=GET", "path=/api/v1/health", "status=200", "dur="} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q missing %q", line, want)
		}
	}
	if strings.Contains(line, "err=") {
		t.Errorf("clean request should not log err: %q", line)
	}
}

func TestRequestLogsFailureWithErr(t *testing.T) {
	var buf bytes.Buffer
	e := newLoggingServer(t, &buf)

	// Missing q is a 400 with the handler's own error body.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}

	line := buf.String()
	if !strings.Contains(line, "path=/api/v1/search") || !strings.Contains(line, "status=400") {
		t.Errorf("log line %q missing search failure fields", line)
	}
}

func TestRequestLogSkipsNonAPIPaths(t *testing.T) {
	var buf bytes.Buffer
	e := newLoggingServer(t, &buf)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if buf.Len() != 0 {
		t.Errorf("non-API request logged: %q", buf.String())
	}
}
