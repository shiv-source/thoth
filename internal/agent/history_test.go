package agent

import (
	"context"
	"testing"
)

func TestHistoryMapsMessagesAndDropsTrailingUser(t *testing.T) {
	st := openStore(t)
	convID, err := st.CreateConversation("hist")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range [][2]string{
		{"user", "one"},
		{"assistant", "two"},
		{"user", "three"}, // the in-flight prompt, appended by the loop
	} {
		if err := st.AddMessage(convID, m[0], m[1]); err != nil {
			t.Fatal(err)
		}
	}

	msgs, err := History(st)(context.Background(), convID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2 (trailing prompt dropped): %+v", len(msgs), msgs)
	}
	want := [][2]string{{"user", "one"}, {"assistant", "two"}}
	for i, m := range msgs {
		if m.Role != want[i][0] || len(m.Content) != 1 || m.Content[0].Text != want[i][1] {
			t.Fatalf("message %d = %+v, want role %q text %q", i, m, want[i][0], want[i][1])
		}
	}
}

func TestHistoryEmptyConversation(t *testing.T) {
	st := openStore(t)
	convID, err := st.CreateConversation("empty")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := History(st)(context.Background(), convID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("got %d messages, want none", len(msgs))
	}
}

func TestHistorySoloPromptStillDrops(t *testing.T) {
	st := openStore(t)
	convID, err := st.CreateConversation("solo")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(convID, "user", "only me"); err != nil {
		t.Fatal(err)
	}
	msgs, err := History(st)(context.Background(), convID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("got %d messages, want none (only the prompt was stored)", len(msgs))
	}
}

func TestHistoryDropsAllTrailingUserMessages(t *testing.T) {
	st := openStore(t)
	convID, err := st.CreateConversation("orphans")
	if err != nil {
		t.Fatal(err)
	}
	// A failed/cancelled turn leaves an orphaned user message with no
	// assistant reply; the in-flight prompt of the next send is another
	// trailing user message. Both must be dropped so history never replays
	// consecutive dangling user turns.
	for _, m := range [][2]string{
		{"user", "one"},
		{"assistant", "two"},
		{"user", "orphan"}, // failed turn: no assistant reply
		{"user", "three"},  // the in-flight prompt, appended by the loop
	} {
		if err := st.AddMessage(convID, m[0], m[1]); err != nil {
			t.Fatal(err)
		}
	}
	msgs, err := History(st)(context.Background(), convID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2 (both orphaned prompts dropped): %+v", len(msgs), msgs)
	}
	if msgs[0].Role != "user" || msgs[0].Content[0].Text != "one" {
		t.Fatalf("message 0 = %+v, want user \"one\"", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content[0].Text != "two" {
		t.Fatalf("message 1 = %+v, want assistant \"two\"", msgs[1])
	}
}

func TestHistorySkipsNonChatRoles(t *testing.T) {
	st := openStore(t)
	convID, err := st.CreateConversation("roles")
	if err != nil {
		t.Fatal(err)
	}
	// A stray tool role row (not a user/assistant turn) must be skipped, and
	// the trailing user prompt dropped.
	if err := st.AddMessage(convID, "tool", "result"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(convID, "user", "hello"); err != nil {
		t.Fatal(err)
	}
	msgs, err := History(st)(context.Background(), convID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("got %d messages, want none (tool role skipped, trailing user dropped)", len(msgs))
	}
}

func TestHistoryStoreError(t *testing.T) {
	st := openStore(t)
	convID, err := st.CreateConversation("boom")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := History(st)(context.Background(), convID); err == nil {
		t.Fatal("expected an error for a closed store")
	}
}
