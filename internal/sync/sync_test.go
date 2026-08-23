package sync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func providerRow(driver, baseURL string) store.SyncProvider {
	return store.SyncProvider{Slug: "p", Name: "Provider", Driver: driver, BaseURL: baseURL}
}

// TestGitHubDriver exercises Verify + Targets against a stub github server and
// maps a rejected token to ErrRejected.
func TestGitHubDriver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo,user")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"login": "octo", "name": "Octo Cat", "email": "o@e.com",
				"avatar_url": "https://a", "html_url": "https://github.com/octo",
			})
		case "/user/emails":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"email": "o@e.com", "primary": true, "verified": true}})
		case "/user/repos":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"full_name": "octo/wiki", "clone_url": "https://github.com/octo/wiki.git", "private": true, "description": ""},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := NewService(testStore(t), srv.Client())
	driver, err := svc.Driver(providerRow("github", srv.URL))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	if driver.Kind() != KindGit {
		t.Fatalf("kind = %v", driver.Kind())
	}
	if len(driver.Fields()) == 0 || !driver.Fields()[0].Secret {
		t.Fatalf("fields wrong: %+v", driver.Fields())
	}

	// Rejected token surfaces as ErrRejected.
	if _, err := driver.Verify(context.Background(), Config{"token": "bad"}); !errors.Is(err, ErrRejected) {
		t.Fatalf("Verify(rejected) = %v, want ErrRejected", err)
	}

	ident, err := driver.Verify(context.Background(), Config{"token": "good"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ident.Username != "octo" || ident.Email != "o@e.com" || ident.Scopes != "repo,user" {
		t.Fatalf("identity wrong: %+v", ident)
	}

	targets, err := driver.Targets(context.Background(), Config{"token": "good"})
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if len(targets) != 1 || targets[0].FullName != "octo/wiki" || !targets[0].Private {
		t.Fatalf("targets wrong: %+v", targets)
	}
}

// TestGitLabDriver exercises the gitlab service against a stub server.
func TestGitLabDriver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v4/user":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"username": "gitguy", "name": "Git Guy", "email": "g@g.com",
				"avatar_url": "https://a", "web_url": "https://gitlab.com/gitguy",
			})
		case "/api/v4/projects":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"path_with_namespace": "gitguy/wiki", "http_url_to_repo": "https://gitlab.com/gitguy/wiki.git", "visibility": "private", "description": ""},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := NewService(testStore(t), srv.Client())
	driver, err := svc.Driver(providerRow("gitlab", srv.URL))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	ident, err := driver.Verify(context.Background(), Config{"token": "good"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ident.Username != "gitguy" || ident.Email != "g@g.com" {
		t.Fatalf("identity wrong: %+v", ident)
	}
	targets, err := driver.Targets(context.Background(), Config{"token": "good"})
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if len(targets) != 1 || targets[0].FullName != "gitguy/wiki" {
		t.Fatalf("targets wrong: %+v", targets)
	}
	if _, err := driver.Verify(context.Background(), Config{"token": "bad"}); !errors.Is(err, ErrRejected) {
		t.Fatalf("Verify(rejected) = %v, want ErrRejected", err)
	}
}

// TestLocalDriver verifies the path guard and writes a snapshot zip.
func TestLocalDriver(t *testing.T) {
	wikiRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(wikiRoot, "note.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(testStore(t), nil)
	driver, err := svc.Driver(providerRow("local", ""))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	if driver.Kind() != KindLocal {
		t.Fatalf("kind = %v", driver.Kind())
	}

	backup := filepath.Join(t.TempDir(), "backup")
	if err := driver.Push(context.Background(), Config{"path": backup}, wikiRoot, Identity{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	entries, err := os.ReadDir(backup)
	if err != nil || len(entries) != 1 {
		t.Fatalf("backup entries = %+v / %v", entries, err)
	}
	if !strings.HasPrefix(entries[0].Name(), "thoth-wiki-") || !strings.HasSuffix(entries[0].Name(), ".zip") {
		t.Fatalf("snapshot name wrong: %s", entries[0].Name())
	}

	// A folder inside the wiki is rejected — it would recurse forever.
	if err := driver.Push(context.Background(), Config{"path": filepath.Join(wikiRoot, "sub")}, wikiRoot, Identity{}); err == nil {
		t.Fatal("pushing into the wiki must fail")
	}
	// Missing folder config is a clean error.
	if err := driver.Push(context.Background(), Config{}, wikiRoot, Identity{}); err == nil {
		t.Fatal("pushing without a folder must fail")
	}
}

// TestS3Driver exercises the s3-compatible path (provider endpoint override):
// HeadBucket verifies, PutObject pushes wiki.zip.
func TestS3Driver(t *testing.T) {
	var gotPut string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			gotPut = r.URL.Path
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK) // HEAD bucket probe
	}))
	defer srv.Close()

	wikiRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(wikiRoot, "note.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1",
		"bucket": "wiki", "prefix": "backups",
	}
	svc := NewService(testStore(t), nil)
	driver, err := svc.Driver(providerRow("s3", srv.URL))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	if driver.Kind() != KindS3 {
		t.Fatalf("kind = %v", driver.Kind())
	}
	ident, err := driver.Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ident.Account != "wiki" {
		t.Fatalf("identity account = %q", ident.Account)
	}
	if err := driver.Push(context.Background(), cfg, wikiRoot, Identity{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	// Path-style addressing: the S3-compatible endpoint receives the bucket
	// as the first path segment, so the object lands at /bucket/backups/wiki.zip.
	if gotPut != "/wiki/backups/wiki.zip" {
		t.Fatalf("put path = %q, want /wiki/backups/wiki.zip", gotPut)
	}
	// Missing bucket is a clean error.
	if err := driver.Push(context.Background(), Config{"access_key_id": "AK", "secret_access_key": "SK"}, wikiRoot, Identity{}); err == nil {
		t.Fatal("pushing without a bucket must fail")
	}
}
