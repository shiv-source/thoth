package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConversationRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	id, err := s.CreateConversation("My first chat")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if id == "" {
		t.Fatal("empty conversation id")
	}
	if err := s.AddMessage(id, "user", "what is in my wiki?"); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := s.AddMessage(id, "assistant", "nothing yet"); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	convs, err := s.ListConversations()
	if err != nil || len(convs) != 1 || convs[0].ID != id || convs[0].Title != "My first chat" {
		t.Fatalf("ListConversations: %v %+v", err, convs)
	}
	msgs, err := s.Messages(id)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("Messages: %v %+v", err, msgs)
	}
	if msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("messages out of order: %+v", msgs)
	}
}

func TestMessagesUnknownConversation(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	msgs, err := s.Messages("nope")
	if err != nil || len(msgs) != 0 {
		t.Fatalf("expected empty, got %v %+v", err, msgs)
	}
}

func TestListConversationsOrderedNewestFirst(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// Insert with explicit timestamps so ordering is deterministic, including
	// a row whose created_at does not parse (CreatedAt must be zero, not fatal).
	for _, row := range []struct {
		id, title, created string
	}{
		{"oldest", "oldest", "2026-01-01T10:00:00Z"},
		{"newest", "newest", "2026-03-01T10:00:00Z"},
		{"middle", "middle", "2026-02-01T10:00:00Z"},
		{"garbage", "garbage", "not-a-time"},
	} {
		if _, err := s.db.Exec(
			`INSERT INTO conversations(id, title, created_at) VALUES (?, ?, ?)`,
			row.id, row.title, row.created); err != nil {
			t.Fatalf("seed conversation %s: %v", row.id, err)
		}
	}

	convs, err := s.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	var order []string
	times := map[string]bool{}
	for _, c := range convs {
		order = append(order, c.ID)
		if !c.CreatedAt.IsZero() {
			times[c.ID] = true
		}
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 conversations, got %+v", order)
	}
	// created_at is stored as text and sorted lexically DESC, so the
	// unparseable "not-a-time" row sorts first; the three RFC3339 rows must
	// still come back newest-first.
	want := []string{"garbage", "newest", "middle", "oldest"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	if times["garbage"] || !times["newest"] {
		t.Fatalf("created_at parsing wrong: %v", times)
	}
}

func TestTimestampsStoredInUTC(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	id, err := s.CreateConversation("utc check")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := s.AddMessage(id, "user", "hi"); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	// created_at is compared lexically by ORDER BY, so it must carry a fixed
	// offset: a local-offset RFC3339 string would sort against UTC rows wrong.
	var convCreated string
	if err := s.db.QueryRow(`SELECT created_at FROM conversations WHERE id = ?`, id).Scan(&convCreated); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(convCreated, "Z") {
		t.Fatalf("conversation created_at %q not UTC (want Z suffix)", convCreated)
	}
	var msgCreated string
	if err := s.db.QueryRow(`SELECT created_at FROM messages WHERE conversation_id = ?`, id).Scan(&msgCreated); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(msgCreated, "Z") {
		t.Fatalf("message created_at %q not UTC (want Z suffix)", msgCreated)
	}
}

func TestClosedStoreErrors(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := s.CreateConversation("x"); err == nil {
		t.Fatal("CreateConversation on closed store must error")
	}
	if err := s.AddMessage("c", "user", "hi"); err == nil {
		t.Fatal("AddMessage on closed store must error")
	}
	if _, err := s.ListConversations(); err == nil {
		t.Fatal("ListConversations on closed store must error")
	}
	if _, err := s.Messages("c"); err == nil {
		t.Fatal("Messages on closed store must error")
	}
}

func TestNewIDIsUUIDShaped(t *testing.T) {
	for range 20 {
		id, err := newID()
		if err != nil {
			t.Fatalf("newID: %v", err)
		}
		// 8-4-4-4-12 hex groups.
		if len(id) != 36 {
			t.Fatalf("id %q is not UUID shaped", id)
		}
		for _, r := range id {
			if r != '-' && (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				t.Fatalf("id %q has invalid character %q", id, r)
			}
		}
		u, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("id %q is not a valid UUID: %v", id, err)
		}
		if u.Version() != 4 || u.Variant() != uuid.RFC4122 {
			t.Fatalf("id %q is not v4 RFC4122 (version %v variant %v)", id, u.Version(), u.Variant())
		}
	}
}

func TestEnsureMetadataSeedsOnce(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	read := func() (string, string) {
		t.Helper()
		var id, created string
		if err := s.db.QueryRow(`SELECT installation_id, created_at FROM app_metadata`).Scan(&id, &created); err != nil {
			t.Fatal(err)
		}
		return id, created
	}

	if err := s.EnsureMetadata(); err != nil {
		t.Fatalf("EnsureMetadata: %v", err)
	}
	id, created := read()
	u, err := uuid.Parse(id)
	if err != nil || u.Version() != 4 || u.Variant() != uuid.RFC4122 {
		t.Fatalf("installation_id %q is not a valid v4 UUID", id)
	}
	if !strings.HasSuffix(created, "Z") {
		t.Fatalf("created_at %q not UTC", created)
	}

	// A second call must not reseed; reopening must keep the same values.
	if err := s.EnsureMetadata(); err != nil {
		t.Fatal(err)
	}
	if againID, againCreated := read(); againID != id || againCreated != created {
		t.Fatalf("metadata changed on second call: %q/%q -> %q/%q", id, created, againID, againCreated)
	}

	// The CHECK (id = 1) constraint keeps the table to one row.
	if _, err := s.db.Exec(`INSERT INTO app_metadata(id, installation_id, created_at) VALUES (2, 'x', 'y')`); err == nil {
		t.Fatal("second app_metadata row must violate the id = 1 constraint")
	}
}

func TestFreshOpenRunsAllMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 16 {
		t.Fatalf("user_version = %d, want 16 (one migration per table, plus the last_attempt_at, provider_headers, and duration_secs columns)", v)
	}
	// The settings seed lands with the migrations.
	var wikiPath string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'wiki_path'`).Scan(&wikiPath); err != nil {
		t.Fatal(err)
	}
	if wikiPath != "~/.thoth/wiki" {
		t.Fatalf("seeded wiki_path = %q", wikiPath)
	}
	// The sync_providers seed lands with migration 0012, including the
	// protected local backup.
	var syncProviders int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sync_providers`).Scan(&syncProviders); err != nil {
		t.Fatal(err)
	}
	if syncProviders != 4 {
		t.Fatalf("seeded sync_providers = %d, want 4 built-ins", syncProviders)
	}
	var localProtected int
	if err := s.db.QueryRow(`SELECT protected FROM sync_providers WHERE slug = 'local'`).Scan(&localProtected); err != nil {
		t.Fatal(err)
	}
	if localProtected != 1 {
		t.Fatalf("local sync provider protected = %d, want 1", localProtected)
	}
}

func TestMessageUsageRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	id, err := s.CreateConversation("usage")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddMessage(id, "user", "question"); err != nil {
		t.Fatal(err)
	}
	const usage = `{"input_tokens":10,"output_tokens":4,"cache_read_tokens":5,"cache_write_tokens":3}`
	if err := s.AddMessage(id, "assistant", "answer", MessageMeta{Usage: usage, DurationSecs: 12.5}); err != nil {
		t.Fatal(err)
	}

	msgs, err := s.Messages(id)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("Messages: %v %+v", err, msgs)
	}
	if msgs[0].Usage != nil {
		t.Fatalf("user message has usage %s, want none", msgs[0].Usage)
	}
	if string(msgs[1].Usage) != usage {
		t.Fatalf("assistant usage = %s, want %s", msgs[1].Usage, usage)
	}
	if msgs[1].DurationSecs == nil || *msgs[1].DurationSecs != 12.5 {
		t.Fatalf("assistant duration = %v, want 12.5", msgs[1].DurationSecs)
	}
	if msgs[0].DurationSecs != nil {
		t.Fatalf("user message has duration %v, want none", msgs[0].DurationSecs)
	}
}

func TestMessageDurationRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	id, err := s.CreateConversation("duration")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddMessage(id, "user", "question"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMessage(id, "assistant", "answer", MessageMeta{DurationSecs: 7.25}); err != nil {
		t.Fatal(err)
	}

	msgs, err := s.Messages(id)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("Messages: %v %+v", err, msgs)
	}
	if msgs[0].DurationSecs != nil {
		t.Fatalf("user message has duration %v, want none", msgs[0].DurationSecs)
	}
	if msgs[1].DurationSecs == nil || *msgs[1].DurationSecs != 7.25 {
		t.Fatalf("assistant duration = %v, want 7.25", msgs[1].DurationSecs)
	}
}

func TestDeleteConversation(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	id, err := s.CreateConversation("to delete")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddMessage(id, "user", "hello"); err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateConversation("to keep")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteConversation(id); err != nil {
		t.Fatal(err)
	}
	convs, err := s.ListConversations()
	if err != nil || len(convs) != 1 || convs[0].ID != other {
		t.Fatalf("after delete: %v %+v", err, convs)
	}
	if msgs, err := s.Messages(id); err != nil || len(msgs) != 0 {
		t.Fatalf("messages not deleted: %v %+v", err, msgs)
	}
	// Idempotent.
	if err := s.DeleteConversation(id); err != nil {
		t.Fatalf("second delete: %v", err)
	}
}

func TestDeleteConversationsBefore(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// Seed three conversations with explicit timestamps (created_at is stored
	// as RFC3339 text) so the retention boundary is deterministic.
	for _, row := range []struct {
		id, title, created string
	}{
		{"old", "old", "2026-01-01T10:00:00Z"},
		{"border", "border", "2026-02-01T10:00:00Z"},
		{"fresh", "fresh", "2026-03-01T10:00:00Z"},
	} {
		if _, err := s.db.Exec(
			`INSERT INTO conversations(id, title, created_at) VALUES (?, ?, ?)`,
			row.id, row.title, row.created); err != nil {
			t.Fatalf("seed conversation %s: %v", row.id, err)
		}
	}
	if err := s.AddMessage("old", "user", "expired"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMessage("fresh", "user", "kept"); err != nil {
		t.Fatal(err)
	}

	// A cutoff of 2026-02-01 00:00:00Z falls before "border", so only "old"
	// (created 2026-01-01) is strictly older.
	cutoff, err := time.Parse(time.RFC3339, "2026-02-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.DeleteConversationsBefore(cutoff)
	if err != nil || n != 1 {
		t.Fatalf("DeleteConversationsBefore = %d/%v, want 1/nil", n, err)
	}
	convs, err := s.ListConversations()
	if err != nil || len(convs) != 2 {
		t.Fatalf("after purge: %v %+v", err, convs)
	}
	// The expired conversation's messages go with it; the fresh ones stay.
	if msgs, err := s.Messages("old"); err != nil || len(msgs) != 0 {
		t.Fatalf("old messages not deleted: %v %+v", err, msgs)
	}
	if msgs, err := s.Messages("fresh"); err != nil || len(msgs) != 1 {
		t.Fatalf("fresh messages not kept: %v %+v", err, msgs)
	}
	// A no-op sweep (same cutoff: nothing left is older than it) returns zero.
	n, err = s.DeleteConversationsBefore(cutoff)
	if err != nil || n != 0 {
		t.Fatalf("no-op sweep = %d/%v, want 0/nil", n, err)
	}
}

func TestOpenDBSetsBusyTimeoutAndPool(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var timeout int
	if err := db.QueryRow(`PRAGMA busy_timeout;`).Scan(&timeout); err != nil {
		t.Fatalf("busy_timeout pragma: %v", err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}
	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", got)
	}
}
