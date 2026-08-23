package sync

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestConfigHelpers(t *testing.T) {
	raw, err := EncodeConfig(Config{"token": "t", "repo_url": "https://x"})
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeConfig(raw)
	if err != nil || back["token"] != "t" || back["repo_url"] != "https://x" {
		t.Fatalf("round trip = %+v / %v", back, err)
	}
	if empty, err := DecodeConfig(""); err != nil || len(empty) != 0 {
		t.Fatalf("empty config = %+v / %v", empty, err)
	}
	if _, err := DecodeConfig("{nope"); err == nil {
		t.Fatal("bad config JSON must error")
	}

	idRaw, err := EncodeIdentity(Identity{Username: "u", Email: "e@x"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := DecodeIdentity(idRaw)
	if err != nil || id.Username != "u" || id.Email != "e@x" {
		t.Fatalf("identity round trip = %+v / %v", id, err)
	}
	if idEmpty, err := DecodeIdentity(""); err != nil || idEmpty != (Identity{}) {
		t.Fatalf("empty identity = %+v / %v", idEmpty, err)
	}
	if _, err := DecodeIdentity("{"); err == nil {
		t.Fatal("bad identity JSON must error")
	}
}

// TestGitDriverPush covers the sanitized failure paths and the happy path
// against a local bare remote (no network, no git binary).
func TestGitDriverPush(t *testing.T) {
	svc := NewService(testStore(t), nil)
	driver, err := svc.Driver(providerRow("github", ""))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	wikiRoot := t.TempDir()
	ident := Identity{Username: "octo", DisplayName: "Octo", Email: "o@e.com"}

	// Missing repo_url / token / identity are clean errors.
	if err := driver.Push(context.Background(), Config{"token": "t"}, wikiRoot, ident); err == nil {
		t.Fatal("push without a repo must fail")
	}
	if err := driver.Push(context.Background(), Config{"repo_url": "https://x"}, wikiRoot, ident); err == nil {
		t.Fatal("push without a token must fail")
	}
	if err := driver.Push(context.Background(), Config{"token": "t", "repo_url": "https://x"}, wikiRoot, Identity{}); err == nil {
		t.Fatal("push without an identity must fail")
	}

	// Happy path: init, set remote, commit, push to a local bare remote.
	if err := os.WriteFile(filepath.Join(wikiRoot, "note.md"), []byte("---\ntitle: Hi\n---\n\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bare := t.TempDir()
	if _, err := git.PlainInit(bare, true); err != nil {
		t.Fatal(err)
	}
	if err := driver.Push(context.Background(), Config{"token": "t", "repo_url": bare}, wikiRoot, ident); err != nil {
		t.Fatalf("Push: %v", err)
	}
	bareRepo, err := git.PlainOpen(bare)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bareRepo.Reference(plumbing.NewBranchReferenceName("main"), false); err != nil {
		t.Fatalf("bare remote missing main: %v", err)
	}
}

func TestLocalVerify(t *testing.T) {
	d := &localDriver{}
	if _, err := d.Verify(context.Background(), Config{}); err == nil {
		t.Fatal("verify without a path must fail")
	}
	if _, err := d.Verify(context.Background(), Config{"path": "/tmp/backup"}); err != nil {
		t.Fatalf("verify with a path: %v", err)
	}
	if targets, err := d.Targets(context.Background(), Config{}); err != nil || targets != nil {
		t.Fatalf("local targets = %+v / %v, want none", targets, err)
	}
}

func TestS3VerifyErrors(t *testing.T) {
	d := &s3Driver{}
	if _, err := d.Verify(context.Background(), Config{"bucket": "b"}); err == nil {
		t.Fatal("verify without keys must fail")
	}
	if _, err := d.Verify(context.Background(), Config{"access_key_id": "AK", "secret_access_key": "SK"}); err == nil {
		t.Fatal("verify without a bucket must fail")
	}
	if targets, err := d.Targets(context.Background(), Config{}); err != nil || targets != nil {
		t.Fatalf("s3 targets = %+v / %v, want none", targets, err)
	}
}

// TestS3VerifyAwsViaSts exercises the real-AWS verify path (no endpoint
// override) by pointing the STS client at a stub through the standard
// AWS_ENDPOINT_URL_STS env var.
func TestS3VerifyAwsViaSts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w,
			`<GetCallerIdentityResponse><GetCallerIdentityResult>`+
				`<Account>123456789012</Account><Arn>arn:aws:iam::123456789012:user/u</Arn><UserId>AIDAX</UserId>`+
				`</GetCallerIdentityResult></GetCallerIdentityResponse>`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "AK")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SK")
	t.Setenv("AWS_REGION", "us-east-1")

	d := &s3Driver{}
	ident, err := d.Verify(context.Background(), Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "b",
	})
	if err != nil {
		t.Fatalf("Verify via sts: %v", err)
	}
	if ident.Account != "123456789012" {
		t.Fatalf("identity account = %q", ident.Account)
	}
}

func TestRegistry(t *testing.T) {
	svc := NewService(testStore(t), nil)
	if _, err := svc.Driver(providerRow("bogus", "")); err == nil {
		t.Fatal("unknown driver must error")
	}
	for _, d := range []string{"github", "gitlab", "s3", "local"} {
		if !svc.KnownDriver(d) {
			t.Fatalf("KnownDriver(%q) = false", d)
		}
	}
	if svc.KnownDriver("bogus") {
		t.Fatal("KnownDriver(bogus) = true")
	}
}

// TestGitTokenGuards covers the git driver's "token is required" guards on
// Verify and Targets (the connection flow surfaces them as a 400).
func TestGitTokenGuards(t *testing.T) {
	svc := NewService(testStore(t), nil)
	driver, err := svc.Driver(providerRow("github", ""))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	if _, err := driver.Verify(context.Background(), Config{}); err == nil {
		t.Fatal("verify without a token must fail")
	}
	if _, err := driver.Targets(context.Background(), Config{}); err == nil {
		t.Fatal("targets without a token must fail")
	}
	if len(driver.Fields()) == 0 {
		t.Fatal("git driver must describe its fields")
	}
}

// TestS3VerifyStsDeniedFallsBackToBucket covers the AWS path where the keys
// are valid but sts is denied: the bucket probe succeeds and identity is
// empty rather than rejected.
func TestS3VerifyStsDeniedFallsBackToBucket(t *testing.T) {
	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(sts.Close)
	s3srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(s3srv.Close)
	t.Setenv("AWS_ENDPOINT_URL_STS", sts.URL)
	t.Setenv("AWS_ENDPOINT_URL_S3", s3srv.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "AK")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SK")
	t.Setenv("AWS_REGION", "us-east-1")

	d := &s3Driver{}
	ident, err := d.Verify(context.Background(), Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "b",
	})
	if err != nil {
		t.Fatalf("Verify with bucket fallback: %v", err)
	}
	if ident != (Identity{}) {
		t.Fatalf("identity = %+v, want empty", ident)
	}
}

// TestS3VerifyBothFailRejected covers both sts and the bucket probe failing:
// the keys are treated as rejected.
func TestS3VerifyBothFailRejected(t *testing.T) {
	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(sts.Close)
	s3srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(s3srv.Close)
	t.Setenv("AWS_ENDPOINT_URL_STS", sts.URL)
	t.Setenv("AWS_ENDPOINT_URL_S3", s3srv.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "AK")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SK")
	t.Setenv("AWS_REGION", "us-east-1")

	d := &s3Driver{}
	if _, err := d.Verify(context.Background(), Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "b",
	}); err != ErrRejected {
		t.Fatalf("Verify both-fail = %v, want ErrRejected", err)
	}
}

// TestS3PushErrors covers the sanitized push guards (bucket and keys) and the
// field descriptor.
func TestS3PushErrors(t *testing.T) {
	d := &s3Driver{}
	root := t.TempDir()
	if err := d.Push(context.Background(), Config{}, root, Identity{}); err == nil {
		t.Fatal("push without a bucket must fail")
	}
	if err := d.Push(context.Background(), Config{"bucket": "b"}, root, Identity{}); err == nil {
		t.Fatal("push without keys must fail")
	}
	if len(d.Fields()) == 0 {
		t.Fatal("s3 driver must describe its fields")
	}
}

// TestS3VerifyEndpointRejected covers the S3-compatible endpoint path where
// the bucket probe fails: the keys are treated as rejected.
func TestS3VerifyEndpointRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	if _, err := d.Verify(context.Background(), Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "b",
	}); err != ErrRejected {
		t.Fatalf("Verify = %v, want ErrRejected", err)
	}
}

// TestS3PushUploadError covers the sanitized upload failure and the
// region-default branch (no region in the config) on a stub endpoint.
func TestS3PushUploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	wikiRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(wikiRoot, "note.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := d.Push(context.Background(), Config{
		"access_key_id": "AK", "secret_access_key": "SK", "bucket": "b", // no region → default
	}, wikiRoot, Identity{})
	if err == nil {
		t.Fatal("upload failure must surface as an error")
	}
}

// TestLocalFields pins the local driver's field descriptor.
func TestLocalFields(t *testing.T) {
	d := &localDriver{}
	fields := d.Fields()
	if len(fields) != 2 || fields[0].Key != "path" || !fields[0].Required {
		t.Fatalf("local fields wrong: %+v", fields)
	}
	if fields[1].Key != "interval" {
		t.Fatalf("local interval field missing: %+v", fields)
	}
}

func TestGitlabAPIDefaultBase(t *testing.T) {
	if s := newGitlabService(nil, ""); s.apiBase() != gitlabAPIBase {
		t.Fatalf("default api base = %q, want %q", s.apiBase(), gitlabAPIBase)
	}
	if s := newGitlabService(nil, "https://gitlab.example.com"); s.apiBase() != "https://gitlab.example.com" {
		t.Fatalf("custom api base = %q", s.apiBase())
	}
}

// TestGitlabTransportError covers the sanitized unreachable error — the
// message must not embed the request URL.
func TestGitlabTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close() // closed before the call → transport error
	s := newGitlabService(srv.Client(), url)
	_, err := s.Profile(context.Background(), "t")
	if err == nil {
		t.Fatal("closed server must error")
	}
	if err.Error() != "fetch gitlab: could not reach gitlab" {
		t.Fatalf("transport error = %q", err.Error())
	}
}
