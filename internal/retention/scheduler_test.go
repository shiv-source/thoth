package retention

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
)

// newTestScheduler builds a scheduler over a temp db (returning the db path
// for raw seeding) with the retention setting pre-seeded to days.
func newTestScheduler(t *testing.T, days int) (*Scheduler, *store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	stg, err := settings.OpenRepo(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stg.Close() })
	if err := stg.SetSetting(settings.KeyConversationRetentionDays, strconv.Itoa(days)); err != nil {
		t.Fatal(err)
	}
	return NewScheduler(st, stg, nil), st, path
}

// seedConversation writes a conversation row with an explicit age so sweep
// outcomes are deterministic (CreateConversation stamps created_at = now).
func seedConversation(t *testing.T, path, id string, age time.Duration) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	created := time.Now().UTC().Add(-age).Format(time.RFC3339)
	if _, err := db.Exec(
		`INSERT INTO conversations(id, title, created_at) VALUES (?, ?, ?)`,
		id, id, created); err != nil {
		t.Fatalf("seed conversation %s: %v", id, err)
	}
}

func TestSweepDeletesExpiredConversations(t *testing.T) {
	s, st, path := newTestScheduler(t, 7)
	seedConversation(t, path, "old", 30*24*time.Hour)
	seedConversation(t, path, "fresh", 0)
	if err := st.AddMessage("old", "user", "expired"); err != nil {
		t.Fatal(err)
	}

	s.sweep(context.Background())

	convs, err := st.ListConversations()
	if err != nil || len(convs) != 1 || convs[0].ID != "fresh" {
		t.Fatalf("after sweep: %v %+v", err, convs)
	}
	if msgs, err := st.Messages("old"); err != nil || len(msgs) != 0 {
		t.Fatalf("expired messages not purged: %v %+v", err, msgs)
	}
}

func TestSweepKeepsConversationsWithinWindow(t *testing.T) {
	s, st, path := newTestScheduler(t, 7)
	seedConversation(t, path, "recent", 6*24*time.Hour)

	s.sweep(context.Background())

	convs, err := st.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("after sweep: %v %+v", err, convs)
	}
}

func TestSweepDisabledWhenRetentionZero(t *testing.T) {
	s, st, path := newTestScheduler(t, 0)
	seedConversation(t, path, "old", 365*24*time.Hour)

	s.sweep(context.Background())

	if convs, err := st.ListConversations(); err != nil || len(convs) != 1 {
		t.Fatalf("retention 0 must keep conversations: %v %+v", err, convs)
	}
}

func TestStartStopsOnCancel(t *testing.T) {
	s, _, _ := newTestScheduler(t, 7)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not stop on ctx cancellation")
	}
}
