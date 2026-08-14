package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shiv-source/thoth/internal/claude"
)

func wsURL(t *testing.T, e http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func readMsg(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal %q: %v", raw, err)
	}
	return m
}

func TestChatSendStreamsAndPersists(t *testing.T) {
	d := testDeps(t)
	fake := &claude.FakeClient{Script: []claude.Event{
		{Type: claude.EventDelta, Text: "Hello "},
		{Type: claude.EventDelta, Text: "wiki"},
		{Type: claude.EventDone},
	}}
	d.Claude = fake
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "what is in my wiki?"}); err != nil {
		t.Fatal(err)
	}

	var gotDelta, gotDone bool
	for !gotDone {
		m := readMsg(t, conn)
		switch m["type"] {
		case "assistant_delta":
			gotDelta = true
		case "turn_done":
			gotDone = true
		case "error":
			t.Fatalf("unexpected error event: %+v", m)
		}
	}
	if !gotDelta {
		t.Fatal("no assistant_delta received")
	}

	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("conversation not persisted: %v %+v", err, convs)
	}
	msgs, err := d.Store.Messages(convs[0].ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("messages not persisted: %v %+v", err, msgs)
	}
}

func TestChatResumeReplays(t *testing.T) {
	d := testDeps(t)
	d.Claude = &claude.FakeClient{Script: []claude.Event{
		{Type: claude.EventDelta, Text: "previous answer"},
		{Type: claude.EventDone},
	}}
	e := New(d)

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn1.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for {
		if readMsg(t, conn1)["type"] == "turn_done" {
			break
		}
	}
	conn1.Close()

	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("expected one conversation: %v %+v", err, convs)
	}

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	if err := conn2.WriteJSON(map[string]string{"type": "resume", "conversation_id": convs[0].ID}); err != nil {
		t.Fatal(err)
	}
	m := readMsg(t, conn2)
	if m["type"] != "assistant_delta" || m["text"] != "previous answer" {
		t.Fatalf("expected replayed delta, got %+v", m)
	}
	if m := readMsg(t, conn2); m["type"] != "turn_done" {
		t.Fatalf("expected turn_done after replay, got %+v", m)
	}
}

// hangClient blocks in Start until its context is cancelled, making the
// cancel wire message testable deterministically (FakeClient replays
// instantly and never exposes an in-flight turn).
type hangClient struct{}

func (hangClient) Start(ctx context.Context, _, _ string, _ claude.EventWriter) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestChatUnknownMessageType(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "bogus"}); err != nil {
		t.Fatal(err)
	}
	m := readMsg(t, conn)
	if m["type"] != "error" || m["message"] != "unknown message type: bogus" {
		t.Fatalf("expected unknown-type error frame, got %+v", m)
	}
}

func TestChatResumeUnknownConversation(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "resume", "conversation_id": "no-such-conv"}); err != nil {
		t.Fatal(err)
	}
	m := readMsg(t, conn)
	if m["type"] != "error" || m["message"] != "unknown conversation" {
		t.Fatalf("expected unknown-conversation error frame, got %+v", m)
	}
}

func TestChatSendStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// With the store closed, the conversation cannot be created: the socket
	// must receive an error frame and stay alive for the next message.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" {
		t.Fatalf("expected error frame, got %+v", m)
	}
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "again"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" {
		t.Fatalf("expected error frame on retry, got %+v", m)
	}
}

func TestChatTurnErrorFromClient(t *testing.T) {
	d := testDeps(t)
	d.Claude = &claude.FakeClient{Err: errors.New("boom")}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	// assistant_start arrives before the turn runs, then the error frame.
	m := readMsg(t, conn)
	if m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start, got %+v", m)
	}
	m = readMsg(t, conn)
	if m["type"] != "error" || m["message"] != "boom" {
		t.Fatalf("expected boom error frame, got %+v", m)
	}
}

func TestChatTruncatesLongTitle(t *testing.T) {
	d := testDeps(t)
	d.Claude = &claude.FakeClient{Script: []claude.Event{{Type: claude.EventDone}}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	long := strings.Repeat("x", 100)
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": long}); err != nil {
		t.Fatal(err)
	}
	for {
		if readMsg(t, conn)["type"] == "turn_done" {
			break
		}
	}
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("conversation not persisted: %v %+v", err, convs)
	}
	if convs[0].Title != strings.Repeat("x", 60) {
		t.Fatalf("title not truncated to 60 chars: %q", convs[0].Title)
	}
}

func TestChatCancelBeforeSendIsNoop(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Cancelling without a conversation must not kill the connection: the
	// following unknown-type frame still gets an error reply.
	if err := conn.WriteJSON(map[string]string{"type": "cancel"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "bogus"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" {
		t.Fatalf("expected error frame, got %+v", m)
	}
}

func TestChatCancelStopsInFlightTurn(t *testing.T) {
	d := testDeps(t)
	d.Claude = hangClient{}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "long task"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start, got %+v", m)
	}
	if err := conn.WriteJSON(map[string]string{"type": "cancel"}); err != nil {
		t.Fatal(err)
	}
	m := readMsg(t, conn)
	if m["type"] != "error" || m["message"] != "cancelled" {
		t.Fatalf("expected cancelled error, got %+v", m)
	}

	// The user message is persisted; a cancelled turn persists nothing more.
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("conversation not persisted: %v %+v", err, convs)
	}
	msgs, err := d.Store.Messages(convs[0].ID)
	if err != nil || len(msgs) != 1 || msgs[0].Role != "user" {
		t.Fatalf("expected only the user message, got %v %+v", err, msgs)
	}
}
