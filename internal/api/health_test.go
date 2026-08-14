package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/shiv-source/thoth/internal/claude"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	// The schema lives in the store's migrations; the index opens the same
	// file afterwards and issues no DDL of its own.
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ix, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	return Deps{
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config:   func() *config.Config { c := config.Default(); return &c }(),
		ConfigMu: &sync.RWMutex{},
		Store:    st,
		Claude:   &claude.FakeClient{},
		Wiki:     wiki.New(t.TempDir()),
		Index:    ix,
	}
}

func TestHealth(t *testing.T) {
	e := New(testDeps(t))
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Status != "ok" {
		t.Fatalf("body: %v %s", err, rec.Body.String())
	}
}
