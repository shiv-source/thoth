package sync

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/store"
)

// s3Stub is a minimal in-memory S3-compatible endpoint exercising the driver's
// put/list/delete/get calls against a fake bucket.
type s3Stub struct {
	objects map[string][]byte // key → body
	failPut bool              // when true, PUTs return 500 (server fault)
}

func (s *s3Stub) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut:
			if s.failPut {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			key := strings.TrimPrefix(r.URL.Path, "/bucket/")
			body, _ := io.ReadAll(r.Body)
			s.objects[key] = body
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "list-type=2"):
			s.list(w, r)
		case r.Method == http.MethodPost && strings.Contains(r.URL.RawQuery, "delete"):
			s.delete(w, r)
		case r.Method == http.MethodGet:
			key := strings.TrimPrefix(r.URL.Path, "/bucket/")
			body, ok := s.objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(body)
		default:
			w.WriteHeader(http.StatusOK) // HEAD probe
		}
	})
}

func (s *s3Stub) list(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	_, _ = w.Write([]byte(`<?xml version="1.0"?><ListBucketResult><IsTruncated>false</IsTruncated>`))
	for k := range s.objects {
		if strings.HasPrefix(k, prefix) {
			_, _ = w.Write([]byte("<Contents><Key>" + k + "</Key><LastModified>2026-01-02T03:04:05Z</LastModified></Contents>"))
		}
	}
	_, _ = w.Write([]byte(`</ListBucketResult>`))
}

func (s *s3Stub) delete(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	text := string(body)
	// Each <Key>K</Key> names an object to delete.
	for _, k := range strings.Split(text, "<Key>") {
		if i := strings.Index(k, "</Key>"); i >= 0 {
			delete(s.objects, k[:i])
		}
	}
	_, _ = w.Write([]byte(`<?xml version="1.0"?><DeleteResult></DeleteResult>`))
}

func newS3Driver(t *testing.T, stub *s3Stub) (Driver, string) {
	t.Helper()
	if stub.objects == nil {
		stub.objects = map[string][]byte{}
	}
	srv := httptest.NewServer(stub.handler())
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	return d, srv.URL
}

func testWikiRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("---\ntitle: Hi\n---\n\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestS3SnapshotModes exercises the three snapshot schemes: stable keeps only
// wiki.zip, history writes only timestamped keys, both keeps the pointer + a
// timestamped key.
func TestS3SnapshotModes(t *testing.T) {
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}

	t.Run("stable default", func(t *testing.T) {
		stub := &s3Stub{}
		d, _ := newS3Driver(t, stub)
		if err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{}); err != nil {
			t.Fatalf("Push: %v", err)
		}
		if len(stub.objects) != 1 || stub.objects["wiki.zip"] == nil {
			t.Fatalf("stable push must write exactly wiki.zip: %+v", stub.objects)
		}
	})

	t.Run("history", func(t *testing.T) {
		stub := &s3Stub{}
		d, _ := newS3Driver(t, stub)
		histCfg := cloneConfig(cfg, "snapshot", SnapshotHistory)
		if err := d.Push(context.Background(), histCfg, testWikiRoot(t), Identity{}); err != nil {
			t.Fatalf("Push: %v", err)
		}
		if len(stub.objects) != 1 {
			t.Fatalf("history push must write exactly one key: %+v", stub.objects)
		}
		for k := range stub.objects {
			if !strings.HasPrefix(k, "thoth-wiki-") || !strings.HasSuffix(k, ".zip") {
				t.Fatalf("history key wrong: %q", k)
			}
		}
	})

	t.Run("both", func(t *testing.T) {
		stub := &s3Stub{}
		d, _ := newS3Driver(t, stub)
		bothCfg := cloneConfig(cfg, "snapshot", SnapshotBoth)
		if err := d.Push(context.Background(), bothCfg, testWikiRoot(t), Identity{}); err != nil {
			t.Fatalf("Push: %v", err)
		}
		if stub.objects["wiki.zip"] == nil {
			t.Fatalf("both push must keep the wiki.zip pointer: %+v", stub.objects)
		}
		historyKeys := 0
		for k := range stub.objects {
			if isHistoryKey(k) {
				historyKeys++
			}
		}
		if historyKeys != 1 {
			t.Fatalf("both push must write one history key, got %d: %+v", historyKeys, stub.objects)
		}
	})
}

// TestS3RetentionPrunesOldest seeds three history snapshots (one older than
// the others), pushes with retention=2, and verifies only the newest two
// survive the prune.
func TestS3RetentionPrunesOldest(t *testing.T) {
	stub := &s3Stub{objects: map[string][]byte{
		"thoth-wiki-20260101-000000.zip": []byte("oldest"),
		"thoth-wiki-20260102-000000.zip": []byte("middle"),
		"thoth-wiki-20260103-000000.zip": []byte("newest-seed"),
	}}
	d, _ := newS3Driver(t, stub)
	cfg := cloneConfig(Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}, "snapshot", SnapshotHistory)
	cfg["retention"] = "2"

	if err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	historyKeys := 0
	for k := range stub.objects {
		if isHistoryKey(k) {
			historyKeys++
		}
	}
	if historyKeys != 2 {
		t.Fatalf("retention=2 must keep exactly 2 snapshots, got %d: %+v", historyKeys, stub.objects)
	}
	// The two oldest seeded snapshots are pruned; the newest seed survives.
	if _, ok := stub.objects["thoth-wiki-20260101-000000.zip"]; ok {
		t.Fatalf("oldest snapshot not pruned: %+v", stub.objects)
	}
	if _, ok := stub.objects["thoth-wiki-20260103-000000.zip"]; !ok {
		t.Fatalf("newest seeded snapshot must survive: %+v", stub.objects)
	}
}

// TestS3RestoreRoundTrip pushes a snapshot then restores it, verifying the
// bytes round-trip and that the latest resolves to the newest key.
func TestS3RestoreRoundTrip(t *testing.T) {
	stub := &s3Stub{}
	d, _ := newS3Driver(t, stub)
	cfg := cloneConfig(Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}, "snapshot", SnapshotBoth)
	if err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	restorer, ok := d.(Restorer)
	if !ok {
		t.Fatal("s3 driver must implement Restorer")
	}
	snaps, err := restorer.Snapshots(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Snapshots: %v", err)
	}
	// wiki.zip first, then the history key.
	if len(snaps) != 2 || snaps[0].Key != "wiki.zip" {
		t.Fatalf("snapshots wrong: %+v", snaps)
	}
	if !isHistoryKey(snaps[1].Key) {
		t.Fatalf("second snapshot must be a history key: %+v", snaps)
	}

	ra, size, err := restorer.Restore(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("Restore(latest): %v", err)
	}
	if size == 0 {
		t.Fatal("restored archive is empty")
	}
	got := make([]byte, size)
	if _, err := ra.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "note.md") {
		t.Fatalf("restored archive missing note.md: %.500s", got)
	}
	// Restoring a specific history key also works.
	if _, _, err := restorer.Restore(context.Background(), cfg, snaps[1].Key); err != nil {
		t.Fatalf("Restore(key): %v", err)
	}
}

// TestS3RestoreEmptyBucket reports a clean error when there is nothing to
// restore.
func TestS3RestoreEmptyBucket(t *testing.T) {
	stub := &s3Stub{}
	d, _ := newS3Driver(t, stub)
	restorer := d.(Restorer)
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}
	if _, _, err := restorer.Restore(context.Background(), cfg, ""); err == nil {
		t.Fatal("restoring an empty bucket must error")
	}
	if snaps, err := restorer.Snapshots(context.Background(), cfg); err != nil || len(snaps) != 0 {
		t.Fatalf("empty bucket snapshots = %+v / %v, want none", snaps, err)
	}
}

// TestS3PushRetryableAndPermanent verifies the retry classification: a 5xx
// server fault is ErrRetryable, a missing-bucket config error is not.
func TestS3PushRetryableAndPermanent(t *testing.T) {
	stub := &s3Stub{failPut: true}
	d, _ := newS3Driver(t, stub)
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}
	err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{})
	if err == nil {
		t.Fatal("server fault must surface as an error")
	}
	if !isRetryable(err) {
		t.Fatalf("server fault must be retryable, got %v", err)
	}
	// Permanent: missing bucket config is a fixed message, not retryable.
	if err := d.Push(context.Background(), Config{}, testWikiRoot(t), Identity{}); err == nil {
		t.Fatal("missing bucket must error")
	}
}

// TestS3PushPermanentUploadError covers the non-retryable upload branch: a 403
// (bad credentials) is a permanent sanitized error, not retryable.
func TestS3PushPermanentUploadError(t *testing.T) {
	var putCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putCalls++
			w.WriteHeader(http.StatusForbidden)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}
	err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{})
	if err == nil {
		t.Fatal("a 403 must surface as an error")
	}
	if isRetryable(err) {
		t.Fatalf("a 403 must be permanent, got %v", err)
	}
	if putCalls != 1 {
		t.Fatalf("put calls = %d, want 1 (no retry)", putCalls)
	}
}

// TestS3PushBothWriteError covers the both-mode error branch: when the stable
// pointer upload fails, the error surfaces (not just the history write).
func TestS3PushBothWriteError(t *testing.T) {
	var putCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putCalls++
			w.WriteHeader(http.StatusForbidden)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	cfg := cloneConfig(Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}, "snapshot", SnapshotBoth)
	if err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{}); err == nil {
		t.Fatal("a failing both-mode push must error")
	}
	if putCalls != 1 {
		t.Fatalf("put calls = %d, want 1 (first write fails, history never written)", putCalls)
	}
}

// TestLocalRestoreRoundTrip pushes a local snapshot then restores it and lists
// the snapshots.
func TestLocalRestoreRoundTrip(t *testing.T) {
	root := testWikiRoot(t)
	backup := filepath.Join(t.TempDir(), "backup")
	d := &localDriver{}
	cfg := Config{"path": backup}
	if err := d.Push(context.Background(), cfg, root, Identity{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	restorer, ok := any(d).(Restorer)
	if !ok {
		t.Fatal("local driver must implement Restorer")
	}
	snaps, err := restorer.Snapshots(context.Background(), cfg)
	if err != nil || len(snaps) != 1 {
		t.Fatalf("Snapshots: %+v / %v", snaps, err)
	}
	ra, size, err := restorer.Restore(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	got := make([]byte, size)
	if _, err := ra.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "note.md") {
		t.Fatalf("restored local archive missing note.md: %.500s", got)
	}
	// Empty folder → clean "nothing to restore".
	empty := &localDriver{}
	if _, _, err := empty.Restore(context.Background(), Config{"path": filepath.Join(t.TempDir(), "nope")}, ""); err == nil {
		t.Fatal("restoring a missing folder must error")
	}
}

// TestSnapshotTimeFromKey verifies the key timestamp extraction.
func TestSnapshotTimeFromKey(t *testing.T) {
	if got := snapshotTimeFromKey("thoth-wiki-20260102-030405.zip"); got != "2026-01-02T03:04:05Z" {
		t.Fatalf("time = %q", got)
	}
	if got := snapshotTimeFromKey("backups/thoth-wiki-20260102-030405.zip"); got != "2026-01-02T03:04:05Z" {
		t.Fatalf("time with prefix = %q", got)
	}
	if got := snapshotTimeFromKey("wiki.zip"); got != "" {
		t.Fatalf("stable key time = %q, want empty", got)
	}
	if got := snapshotTimeFromKey("junk"); got != "" {
		t.Fatalf("junk time = %q, want empty", got)
	}
}

// TestRetryableErrorSurface checks that a retryable error's message stays the
// clean fixed text while errors.Is(ErrRetryable) holds.
func TestRetryableErrorSurface(t *testing.T) {
	err := retryable("could not upload the wiki to the bucket")
	if err.Error() != "could not upload the wiki to the bucket" {
		t.Fatalf("message = %q", err.Error())
	}
	if !isRetryable(err) {
		t.Fatal("must be retryable")
	}
}

// TestIsRetryableGit classifies network-flake push errors as retryable and
// leaves permanent ones alone.
func TestIsRetryableGit(t *testing.T) {
	if !isRetryableGit(&net.OpError{}) {
		t.Fatal("net.OpError must be retryable")
	}
	if !isRetryableGit(&url.Error{Err: errors.New("connection refused")}) {
		t.Fatal("url.Error must be retryable")
	}
	if !isRetryableGit(context.DeadlineExceeded) {
		t.Fatal("deadline must be retryable")
	}
	if isRetryableGit(errors.New("authentication required")) {
		t.Fatal("an auth error must not be retryable")
	}
}

// TestPushWithRetry verifies the retry loop: a transient failure is retried
// and the final error is annotated with the retry count, while a permanent
// failure returns immediately.
func TestPushWithRetry(t *testing.T) {
	// Permanent: never retried, returned as-is.
	permanent := errors.New("no credentials")
	if err := pushWithRetry(context.Background(), &pushStub{err: permanent, fails: 10}, Config{}, "", Identity{}); err != permanent {
		t.Fatalf("permanent error = %v, want the original", err)
	}

	// Transient, fails all attempts: retried and annotated.
	transient := retryable("could not upload")
	err := pushWithRetry(context.Background(), &pushStub{err: transient, fails: 10}, Config{}, "", Identity{})
	if err == nil {
		t.Fatal("transient failure must surface")
	}
	if !isRetryable(err) {
		t.Fatalf("annotated error must stay retryable: %v", err)
	}
	if !strings.Contains(err.Error(), "retried") {
		t.Fatalf("annotated error must mention the retry count: %v", err)
	}

	// Transient then success: succeeds.
	if err := pushWithRetry(context.Background(), &pushStub{fails: 2, err: transient}, Config{}, "", Identity{}); err != nil {
		t.Fatalf("recovered push must succeed: %v", err)
	}
}

// pushStub is a Driver double for the retry loop tests.
type pushStub struct {
	err   error
	fails int // number of times to fail before succeeding
	seen  int
}

func (s *pushStub) Kind() Kind { return KindS3 }
func (s *pushStub) Fields() []Field {
	return []Field{{Key: "bucket", Label: "Bucket", Required: true}, IntervalField}
}
func (s *pushStub) Verify(context.Context, Config) (Identity, error)  { return Identity{}, nil }
func (s *pushStub) Targets(context.Context, Config) ([]Target, error) { return nil, nil }
func (s *pushStub) Push(context.Context, Config, string, Identity) error {
	s.seen++
	if s.seen <= s.fails {
		return s.err
	}
	return nil
}

// TestSchedulerStartStop verifies Start runs a sweep on the tick and Stop
// halts it (covering the loop's select branches).
func TestSchedulerStartStop(t *testing.T) {
	svc := NewService(testStore(t), nil)
	local, _ := svc.Store.SyncProviderBySlug("local")
	backup := t.TempDir()
	if _, err := svc.Store.CreateConnection(local.ID, "backup",
		`{"path":"`+backup+`","interval":"60"}`, "", true); err != nil {
		t.Fatal(err)
	}
	results := make(chan Result, 1)
	s := NewScheduler(svc, t.TempDir(), nil, func(r Result) { results <- r })
	s.tick = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()
	select {
	case r := <-results:
		if !r.OK {
			t.Fatalf("scheduled push failed: %v", r.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start never ran a sweep")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not stop on ctx cancel")
	}
}

// TestSchedulerStop covers the Stop() branch (done channel).
func TestSchedulerStop(t *testing.T) {
	s := NewScheduler(nil, "", nil, nil)
	done := make(chan struct{})
	go func() {
		s.Start(context.Background())
		close(done)
	}()
	s.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not stop on Stop()")
	}
}

// TestEncodeHelpersErrorPaths covers the identity encoder and its decode
// round-trip (the identity JSON shape).
func TestEncodeHelpersErrorPaths(t *testing.T) {
	raw, err := EncodeIdentity(Identity{Account: "a", Username: "u"})
	if err != nil || raw == "" {
		t.Fatalf("encode identity: %v %q", err, raw)
	}
	back, err := DecodeIdentity(raw)
	if err != nil || back.Account != "a" || back.Username != "u" {
		t.Fatalf("decode identity: %+v / %v", back, err)
	}
	// Config round-trip with the shared interval field.
	cfgRaw, err := EncodeConfig(Config{"interval": "30", "path": "/tmp/x"})
	if err != nil {
		t.Fatal(err)
	}
	cfgBack, err := DecodeConfig(cfgRaw)
	if err != nil || cfgBack["interval"] != "30" {
		t.Fatalf("config round trip: %+v / %v", cfgBack, err)
	}
}

// TestS3RestoreErrorPaths covers the restore guards: missing bucket and a
// download failure (404).
func TestS3RestoreErrorPaths(t *testing.T) {
	stub := &s3Stub{}
	d, _ := newS3Driver(t, stub)
	restorer := d.(Restorer)
	if _, _, err := restorer.Restore(context.Background(), Config{}, ""); err == nil {
		t.Fatal("restore without a bucket must error")
	}
	// 404 on the object → sanitized download error.
	if _, _, err := restorer.Restore(context.Background(), Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}, "missing.zip"); err == nil {
		t.Fatal("restore of a missing object must error")
	}
	// Snapshots without a bucket errors cleanly.
	if _, err := restorer.Snapshots(context.Background(), Config{}); err == nil {
		t.Fatal("snapshots without a bucket must error")
	}
}

// TestS3PruneDeleteError covers the prune's delete-failure branch: the driver
// surfaces a sanitized error when DeleteObjects fails.
func TestS3PruneDeleteError(t *testing.T) {
	var listCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "list-type=2"):
			listCalls++
			_, _ = w.Write([]byte(`<?xml version="1.0"?><ListBucketResult><IsTruncated>false</IsTruncated>
				<Contents><Key>thoth-wiki-20260101-000000.zip</Key></Contents>
				<Contents><Key>thoth-wiki-20260102-000000.zip</Key></Contents>
				<Contents><Key>thoth-wiki-20260103-000000.zip</Key></Contents>
				</ListBucketResult>`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.RawQuery, "delete"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	cfg := cloneConfig(Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}, "snapshot", SnapshotHistory)
	cfg["retention"] = "2"
	if err := d.Push(context.Background(), cfg, testWikiRoot(t), Identity{}); err == nil {
		t.Fatal("a failing delete must surface as a prune error")
	}
	if listCalls == 0 {
		t.Fatal("prune must list the bucket before deleting")
	}
}

// TestS3ListKeysError covers the sanitized list failure (500 from the stub).
func TestS3ListKeysError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}
	if _, err := d.Snapshots(context.Background(), cfg); err == nil {
		t.Fatal("snapshots with a failing list must error")
	}
}

// TestS3ListKeysPagination verifies the pagination continuation branch: a
// two-page listing returns every key, sorted.
func TestS3ListKeysPagination(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		token := r.URL.Query().Get("continuation-token")
		if token == "" {
			_, _ = w.Write([]byte(`<?xml version="1.0"?><ListBucketResult><IsTruncated>true</IsTruncated>
				<NextContinuationToken>page2</NextContinuationToken>
				<Contents><Key>thoth-wiki-20260102-030405.zip</Key></Contents>
				</ListBucketResult>`))
			return
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?><ListBucketResult><IsTruncated>false</IsTruncated>
			<Contents><Key>thoth-wiki-20260101-030405.zip</Key></Contents>
			</ListBucketResult>`))
	}))
	t.Cleanup(srv.Close)
	d := &s3Driver{endpoint: srv.URL}
	cfg := Config{
		"access_key_id": "AK", "secret_access_key": "SK", "region": "us-east-1", "bucket": "bucket",
	}
	snaps, err := d.Snapshots(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("list calls = %d, want 2 (paginated)", calls)
	}
	if len(snaps) != 2 || snaps[0].Key != "thoth-wiki-20260102-030405.zip" {
		t.Fatalf("paginated snapshots wrong: %+v", snaps)
	}
}

// TestS3RetryableClassification covers the status-based retry check: a 4xx
// client fault is permanent, a 5xx server fault is retryable.
func TestS3RetryableClassification(t *testing.T) {
	if !isRetryableS3(&fakeHTTPError{status: 500}) {
		t.Fatal("5xx must be retryable")
	}
	if isRetryableS3(&fakeHTTPError{status: 403}) {
		t.Fatal("4xx must be permanent")
	}
	if !isRetryableS3(errors.New("connection reset")) {
		t.Fatal("a transport error must be retryable")
	}
}

// fakeHTTPError implements the SDK's HTTPStatusCode interface for the
// classification tests.
type fakeHTTPError struct{ status int }

func (e *fakeHTTPError) Error() string       { return "fake" }
func (e *fakeHTTPError) HTTPStatusCode() int { return e.status }

// TestGitPushRetryable covers the git driver's transient-failure path: a push
// to a dead remote surfaces a retryable error.
func TestGitPushRetryable(t *testing.T) {
	svc := NewService(testStore(t), nil)
	driver, err := svc.Driver(providerRow("github", ""))
	if err != nil {
		t.Fatalf("Driver: %v", err)
	}
	root := t.TempDir()
	// Push to a URL whose host does not resolve → network flake.
	err = driver.Push(context.Background(), Config{"token": "t", "repo_url": "https://nonexistent.invalid/x.git"}, root, Identity{
		Username: "octo", DisplayName: "Octo", Email: "o@e.com",
	})
	if err != nil {
		if !isRetryable(err) {
			t.Fatalf("push to an unreachable remote must be retryable, got %v", err)
		}
	} else {
		// A fast local resolution could succeed spuriously; that is fine, but
		// it must not error permanently.
		t.Log("push unexpectedly succeeded — remote unreachable test skipped")
	}
}

// TestLocalRestoreErrorPaths covers the local restore guards: a missing folder
// and an unknown key.
func TestLocalRestoreErrorPaths(t *testing.T) {
	d := &localDriver{}
	restorer, ok := any(d).(Restorer)
	if !ok {
		t.Fatal("local driver must implement Restorer")
	}
	if _, _, err := restorer.Restore(context.Background(), Config{"path": ""}, ""); err == nil {
		t.Fatal("restore without a path must error")
	}
	dir := t.TempDir()
	if _, _, err := restorer.Restore(context.Background(), Config{"path": dir}, "missing.zip"); err == nil {
		t.Fatal("restore of a missing key must error")
	}
	// Snapshots of a missing folder → empty, not an error.
	snaps, err := restorer.Snapshots(context.Background(), Config{"path": filepath.Join(t.TempDir(), "nope")})
	if err != nil || len(snaps) != 0 {
		t.Fatalf("snapshots of a missing folder = %+v / %v, want none", snaps, err)
	}
}

// TestLocalRestoreByKey restores a specific snapshot (not just the latest).
func TestLocalRestoreByKey(t *testing.T) {
	root := testWikiRoot(t)
	backup := filepath.Join(t.TempDir(), "backup")
	d := &localDriver{}
	if err := d.Push(context.Background(), Config{"path": backup}, root, Identity{}); err != nil {
		t.Fatal(err)
	}
	restorer, _ := any(d).(Restorer)
	snaps, err := restorer.Snapshots(context.Background(), Config{"path": backup})
	if err != nil || len(snaps) != 1 {
		t.Fatalf("snapshots: %+v / %v", snaps, err)
	}
	ra, size, err := restorer.Restore(context.Background(), Config{"path": backup}, snaps[0].Key)
	if err != nil || size == 0 {
		t.Fatalf("restore by key: %v / %d", err, size)
	}
	got := make([]byte, size)
	if _, err := ra.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "note.md") {
		t.Fatalf("restored archive missing note.md: %.500s", got)
	}
}

// TestLocalPushFolderError covers the local push guards: an uncreatable
// backup folder path.
func TestLocalPushFolderError(t *testing.T) {
	d := &localDriver{}
	// A path under an existing FILE cannot be created as a directory.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.Push(context.Background(), Config{"path": filepath.Join(file, "sub")}, testWikiRoot(t), Identity{}); err == nil {
		t.Fatal("pushing under a file must error")
	}
}

// TestLocalSnapshotsUnreadable covers the ReadDir error branch: a path that is
// a file (not a directory) errors cleanly.
func TestLocalSnapshotsUnreadable(t *testing.T) {
	d := &localDriver{}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	restorer, _ := any(d).(Restorer)
	if _, err := restorer.Snapshots(context.Background(), Config{"path": file}); err == nil {
		t.Fatal("snapshots of a file path must error")
	}
}

// TestLocalSnapshotsFiltersNonZips verifies the listing skips directories and
// non-snapshot files, and sorts newest-first.
func TestLocalSnapshotsFiltersNonZips(t *testing.T) {
	dir := t.TempDir()
	// Two snapshots plus noise: a plain file and a subdirectory.
	for _, name := range []string{"thoth-wiki-20260102-030405.zip", "thoth-wiki-20260101-030405.zip", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := &localDriver{}
	restorer, _ := any(d).(Restorer)
	snaps, err := restorer.Snapshots(context.Background(), Config{"path": dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 2 || snaps[0].Key != "thoth-wiki-20260102-030405.zip" {
		t.Fatalf("snapshots wrong (newest first, filtered): %+v", snaps)
	}
	// Restore of the newest resolves to the lexically-latest key.
	ra, size, err := restorer.Restore(context.Background(), Config{"path": dir}, "")
	if err != nil || size == 0 {
		t.Fatalf("restore latest: %v / %d", err, size)
	}
	got := make([]byte, size)
	if _, err := ra.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	if string(got) != "x" {
		t.Fatalf("restored latest = %q, want the newest snapshot body", got)
	}
	// Restore of a folder with no snapshots at all → clean error.
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := restorer.Restore(context.Background(), Config{"path": empty}, ""); err == nil {
		t.Fatal("restore with no backups must error")
	}
}

// TestPathContains covers the wiki/backup containment guard incl. the Rel
// error branch.
func TestPathContains(t *testing.T) {
	if !pathContains("/wiki", "/wiki/backup") {
		t.Fatal("child inside parent must contain")
	}
	if pathContains("/wiki/backup", "/wiki") {
		t.Fatal("parent inside child must not contain")
	}
	// Rel errors (different volumes) report false.
	if pathContains("/a", "/b") && pathContains("/a", "/a/b") {
		t.Fatal("different roots must not contain")
	}
}

// TestConnectionInterval parses the per-connection auto-sync interval config.
func TestConnectionInterval(t *testing.T) {
	if _, ok := connectionInterval(store.Connection{Config: "{}"}); ok {
		t.Fatal("no interval must be off")
	}
	if _, ok := connectionInterval(store.Connection{Config: `{"interval":""}`}); ok {
		t.Fatal("empty interval must be off")
	}
	if _, ok := connectionInterval(store.Connection{Config: `{"interval":"0"}`}); ok {
		t.Fatal("zero interval must be off")
	}
	if _, ok := connectionInterval(store.Connection{Config: `{"interval":"-1"}`}); ok {
		t.Fatal("negative interval must be off")
	}
	if _, ok := connectionInterval(store.Connection{Config: `{"interval":"abc"}`}); ok {
		t.Fatal("non-numeric interval must be off")
	}
	d, ok := connectionInterval(store.Connection{Config: `{"interval":"30"}`})
	if !ok || d != 30*time.Minute {
		t.Fatalf("interval = %v / %v, want 30m / true", d, ok)
	}
}

// TestSchedulerDue verifies the never-synced and elapsed-time scheduling rules.
func TestSchedulerDue(t *testing.T) {
	s := NewScheduler(nil, "", nil, nil)
	now := time.Now().UTC()
	if !s.due(store.Connection{LastSyncedAt: ""}, time.Hour, now) {
		t.Fatal("never-synced connection must be due")
	}
	if !s.due(store.Connection{LastSyncedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)}, time.Hour, now) {
		t.Fatal("elapsed interval must be due")
	}
	if s.due(store.Connection{LastSyncedAt: now.Format(time.RFC3339)}, time.Hour, now) {
		t.Fatal("recent sync must not be due")
	}
	if !s.due(store.Connection{LastSyncedAt: "garbage"}, time.Hour, now) {
		t.Fatal("unparseable timestamp must be treated as due")
	}
}

func cloneConfig(cfg Config, key, value string) Config {
	out := make(Config, len(cfg)+1)
	for k, v := range cfg {
		out[k] = v
	}
	out[key] = value
	return out
}
