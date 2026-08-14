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
