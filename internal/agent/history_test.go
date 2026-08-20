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
