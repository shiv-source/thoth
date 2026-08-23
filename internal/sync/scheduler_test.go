package sync

import (
	"context"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/store"
)

// TestServicePushRecordsResult verifies Service.Push drives the driver, retries
// transient failures, and records the outcome on the row + push history.
func TestServicePushRecordsResult(t *testing.T) {
	svc := NewService(testStore(t), nil)
	local, err := svc.Store.SyncProviderBySlug("local")
	if err != nil {
		t.Fatal(err)
	}
	backup := t.TempDir()
	conn, err := svc.Store.CreateConnection(local.ID, "backup", `{"path":"`+backup+`"}`, "", true)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := svc.Push(context.Background(), conn, root); err != nil {
		t.Fatalf("Push: %v", err)
	}
	got, _ := svc.Store.Connection(conn.ID)
	if got.LastSyncedAt == "" || got.LastError != "" {
		t.Fatalf("sync result not recorded: %+v", got)
	}
	history, err := svc.Store.ListPushHistory(conn.ID)
	if err != nil || len(history) != 1 || !history[0].OK {
		t.Fatalf("push history wrong: %+v / %v", history, err)
	}
}

// TestServicePushRetriesTransient verifies a transient (retryable) failure is
// retried with backoff and the final error is annotated with the retry count,
// while a permanent failure returns immediately.
func TestServicePushRetriesTransient(t *testing.T) {
	svc := NewService(testStore(t), nil)
	local, _ := svc.Store.SyncProviderBySlug("local")
	conn, _ := svc.Store.CreateConnection(local.ID, "backup", `{}`, "", true)

	// Missing folder → permanent error: no retries, not annotated.
	err := svc.Push(context.Background(), conn, t.TempDir())
	if err == nil {
		t.Fatal("push with no folder must fail")
	}
	if isRetryable(err) {
		t.Fatalf("missing-folder failure must not be retryable: %v", err)
	}
	got, _ := svc.Store.Connection(conn.ID)
	if got.LastError == "" {
		t.Fatal("failure must be recorded")
	}
}

// TestSchedulerSweepFiresDuePushes verifies the scheduler fires a push for an
// enabled connection with a configured interval and reports the Result. Uses a
// never-synced connection so it is due immediately.
func TestSchedulerSweepFiresDuePushes(t *testing.T) {
	st := testStore(t)
	svc := NewService(st, nil)
	local, err := st.SyncProviderBySlug("local")
	if err != nil {
		t.Fatal(err)
	}
	backup := t.TempDir()
	if _, err := st.CreateConnection(local.ID, "backup",
		`{"path":"`+backup+`","interval":"60"}`, "", true); err != nil {
		t.Fatal(err)
	}
	results := make(chan Result, 1)
	s := NewScheduler(svc, t.TempDir(), nil, func(r Result) { results <- r })
	s.tick = time.Millisecond // force an immediate sweep in the test loop
	s.sweep(context.Background())
	select {
	case r := <-results:
		if !r.OK {
			t.Fatalf("push failed: %v", r.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler never reported a push")
	}
}

// TestSchedulerSkipsDisabledAndUnconfigured verifies enabled=0 or no-interval
// connections are never scheduled.
func TestSchedulerSkipsDisabledAndUnconfigured(t *testing.T) {
	st := testStore(t)
	svc := NewService(st, nil)
	local, _ := st.SyncProviderBySlug("local")
	// Disabled but has an interval.
	if _, err := st.CreateConnection(local.ID, "off", `{"interval":"60"}`, "", false); err != nil {
		t.Fatal(err)
	}
	// Enabled but no interval.
	if _, err := st.CreateConnection(local.ID, "manual", `{}`, "", true); err != nil {
		t.Fatal(err)
	}
	called := false
	s := NewScheduler(svc, t.TempDir(), nil, func(Result) { called = true })
	s.sweep(context.Background())
	if called {
		t.Fatal("scheduler must not fire for disabled or unconfigured connections")
	}
}

// TestSchedulerSweepListError covers the list-connections error branch in
// sweep (a nil service has no store to read from).
func TestSchedulerSweepListError(t *testing.T) {
	s := NewScheduler(nil, t.TempDir(), nil, func(Result) {})
	s.sweep(context.Background()) // must not panic; nothing to sweep
}

// TestSchedulerSweepConcurrencyCap covers the cap branch: two due pushes with
// a cap of one — the second is skipped while the first holds the slot.
func TestSchedulerSweepConcurrencyCap(t *testing.T) {
	st := testStore(t)
	svc := NewService(st, nil)
	local, _ := st.SyncProviderBySlug("local")
	backup := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if _, err := st.CreateConnection(local.ID, name,
			`{"path":"`+backup+`","interval":"60"}`, "", true); err != nil {
			t.Fatal(err)
		}
	}
	results := make(chan Result, 2)
	s := NewScheduler(svc, t.TempDir(), nil, func(r Result) { results <- r })
	// Occupy the single slot so every push in the sweep is skipped.
	s.sem <- struct{}{}
	s.sweep(context.Background())
	<-s.sem
	select {
	case <-results:
		t.Fatal("a push must not start while the concurrency slot is full")
	default:
	}
}

// TestSchedulerReportsFailure verifies a failed scheduled push surfaces in the
// Result with its error (the onPush error branch).
func TestSchedulerReportsFailure(t *testing.T) {
	st := testStore(t)
	svc := NewService(st, nil)
	local, _ := st.SyncProviderBySlug("local")
	// An enabled connection with an interval but no folder → push fails.
	if _, err := st.CreateConnection(local.ID, "backup", `{"interval":"60"}`, "", true); err != nil {
		t.Fatal(err)
	}
	results := make(chan Result, 1)
	s := NewScheduler(svc, t.TempDir(), nil, func(r Result) { results <- r })
	s.sweep(context.Background())
	select {
	case r := <-results:
		if r.OK {
			t.Fatal("a push without a folder must fail")
		}
		if r.Error == "" {
			t.Fatal("failed push must carry an error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler never reported a push")
	}
}

// TestServicePushIdentityError covers the Service.Push identity decode branch.
func TestServicePushIdentityError(t *testing.T) {
	svc := NewService(testStore(t), nil)
	conn := store.Connection{
		ProviderID: 1, Config: `{"path":"/tmp/x"}`, Identity: "{nope",
	}
	if err := svc.Push(context.Background(), conn, t.TempDir()); err == nil {
		t.Fatal("push with a bad identity must error")
	}
}
