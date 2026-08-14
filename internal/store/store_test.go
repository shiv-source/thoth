package store

import (
	"path/filepath"
	"testing"
)

func TestConversationRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

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
	t.Cleanup(func() { s.Close() })
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
	t.Cleanup(func() { s.Close() })

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
	for i := 0; i < 20; i++ {
		id, err := newID()
		if err != nil {
			t.Fatalf("newID: %v", err)
		}
		// 8-4-4-4-12 hex groups; the claude CLI requires UUID-shaped session ids.
		if len(id) != 36 {
			t.Fatalf("id %q is not UUID shaped", id)
		}
		for _, r := range id {
			if !(r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Fatalf("id %q has invalid character %q", id, r)
			}
		}
	}
}
