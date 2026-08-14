package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

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
		// 8-4-4-4-12 hex groups; the claude CLI requires UUID-shaped session ids.
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

// readConversationIDs returns title -> id for every conversation.
func readConversationIDs(t *testing.T, s *Store) map[string]string {
	t.Helper()
	rows, err := s.db.Query(`SELECT title, id FROM conversations`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	m := map[string]string{}
	for rows.Next() {
		var title, id string
		if err := rows.Scan(&title, &id); err != nil {
			t.Fatal(err)
		}
		m[title] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMigrateConversationIDsToValidUUIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Rewind the version marker and seed rows the way a pre-v1 database
	// would have them: UUID-shaped ids with random version/variant nibbles,
	// plus one already-valid v4 that must survive untouched.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 0`); err != nil {
		t.Fatal(err)
	}
	seeded := map[string]string{ // title -> id
		"legacy-1": "aaaaaaaa-aaaa-aaaa-3000-000000000000", // reserved variant nibble 3
		"legacy-2": "bbbbbbbb-bbbb-bbbb-7000-000000000000", // reserved variant nibble 7
		"legacy-3": "00000000-0000-0000-8000-000000000000", // version nibble 0
		"legacy-4": "dddddddd-dddd-dddd-c000-000000000000", // Microsoft variant nibble c
		"valid":    "123e4567-e89b-4122-a456-426614174000",
	}
	for title, id := range seeded {
		if _, err := db.Exec(`INSERT INTO conversations (id, title, created_at) VALUES (?, ?, ?)`,
			id, title, "2026-08-15T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	for _, m := range [][2]string{
		{seeded["legacy-1"], "user"},
		{seeded["legacy-1"], "assistant"},
		{seeded["valid"], "user"},
	} {
		if _, err := db.Exec(`INSERT INTO messages (conversation_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
			m[0], m[1], "hello", "2026-08-15T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopening runs migration 1.
	s, err = Open(path)
	if err != nil {
		t.Fatalf("Open after seed: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("user_version = %d, want 1", v)
	}

	got := readConversationIDs(t, s)
	if len(got) != len(seeded) {
		t.Fatalf("got %d conversations, want %d", len(got), len(seeded))
	}
	if got["valid"] != seeded["valid"] {
		t.Fatalf("valid v4 id was rewritten: %q -> %q", seeded["valid"], got["valid"])
	}
	for title, old := range seeded {
		if title == "valid" {
			continue
		}
		id := got[title]
		if id == old {
			t.Fatalf("%s id %q was not rewritten", title, old)
		}
		u, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("%s new id %q is not a valid UUID: %v", title, id, err)
		}
		if u.Version() != 4 || u.Variant() != uuid.RFC4122 {
			t.Fatalf("%s new id %q is not v4 RFC4122 (version %v variant %v)", title, id, u.Version(), u.Variant())
		}
	}

	// Messages must follow their conversation to the new id; the valid
	// conversation's message stays put.
	rows, err := s.db.Query(`SELECT conversation_id, role FROM messages ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var msgs [][2]string
	for rows.Next() {
		var convID, role string
		if err := rows.Scan(&convID, &role); err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, [2]string{convID, role})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("got %d messages, want 3", len(msgs))
	}
	if msgs[0][0] != got["legacy-1"] || msgs[1][0] != got["legacy-1"] {
		t.Fatalf("legacy messages did not follow the rewrite: %v", msgs)
	}
	if msgs[2][0] != seeded["valid"] {
		t.Fatalf("valid conversation's message was rewritten: %v", msgs[2])
	}

	// Idempotency: a second open must not rewrite again.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s2.Close() }()
	again := readConversationIDs(t, s2)
	for title, id := range got {
		if again[title] != id {
			t.Fatalf("%s id changed on reopen: %q -> %q", title, id, again[title])
		}
	}
}
