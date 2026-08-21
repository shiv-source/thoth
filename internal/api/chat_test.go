package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	agentlib "github.com/shiv-source/thoth/agent"
)

func wsURL(t *testing.T, e http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func readMsg(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
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
	fake := &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventDelta, Text: "Hello "},
		{Type: agentlib.EventDelta, Text: "wiki"},
		{Type: agentlib.EventDone},
	}}
	d.Claude = fake
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

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

func TestChatStreamsToolActivity(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventTool, Tool: "Read", Detail: "note.md"},
		{Type: agentlib.EventDelta, Text: "answer"},
		{Type: agentlib.EventDone},
	}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "look at my notes"}); err != nil {
		t.Fatal(err)
	}
	var gotTool, gotDone bool
	for !gotDone {
		m := readMsg(t, conn)
		switch m["type"] {
		case "tool_activity":
			if m["tool"] != "Read" || m["detail"] != "note.md" {
				t.Fatalf("unexpected tool frame: %+v", m)
			}
			gotTool = true
		case "turn_done":
			gotDone = true
		case "error":
			t.Fatalf("unexpected error event: %+v", m)
		}
	}
	if !gotTool {
		t.Fatal("no tool_activity received")
	}
}

func TestChatResumeReplays(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventDelta, Text: "previous answer"},
		{Type: agentlib.EventDone},
	}}
	e := New(d)

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn1.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn1)["type"] != "turn_done" {
	}
	_ = conn1.Close()

	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("expected one conversation: %v %+v", err, convs)
	}

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn2.Close() }()
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

func (hangClient) Start(ctx context.Context, _, _ string, _ agentlib.EventWriter) (agentlib.Usage, error) {
	<-ctx.Done()
	return agentlib.Usage{}, ctx.Err()
}

func TestChatUnknownMessageType(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

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
	defer func() { _ = conn.Close() }()

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
	defer func() { _ = conn.Close() }()

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
	d.Claude = &FakeClient{Err: errors.New("boom")}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

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
	d.Claude = &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	long := strings.Repeat("x", 100)
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": long}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn)["type"] != "turn_done" {
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
	defer func() { _ = conn.Close() }()

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

func TestChatOriginCheck(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	u := wsURL(t, e)

	// An evil browser origin must not even complete the handshake. The URL
	// still points at the local server; only the Origin header is hostile.
	if _, _, err := websocket.DefaultDialer.Dial(u, http.Header{"Origin": []string{"http://evil.example.com"}}); err == nil {
		t.Fatal("expected handshake failure for evil origin")
	}

	// The Vite dev server origin is allowed.
	hdr := http.Header{"Origin": []string{"http://localhost:5173"}}
	conn, _, err := websocket.DefaultDialer.Dial(u, hdr)
	if err != nil {
		t.Fatalf("dial with localhost origin: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// And so is an origin-less (non-browser) client.
	conn2, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial without origin: %v", err)
	}
	defer func() { _ = conn2.Close() }()

	// IPv6 loopback on any port too.
	hdr6 := http.Header{"Origin": []string{"http://[::1]:8080"}}
	conn3, _, err := websocket.DefaultDialer.Dial(u, hdr6)
	if err != nil {
		t.Fatalf("dial with [::1] origin: %v", err)
	}
	defer func() { _ = conn3.Close() }()
}

func TestChatTurnDoneCarriesConversationID(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for {
		m := readMsg(t, conn)
		if m["type"] != "turn_done" {
			continue
		}
		if id, _ := m["conversation_id"].(string); id == "" {
			t.Fatalf("turn_done missing conversation_id: %+v", m)
		}
		return
	}
}

func TestChatTurnDoneCarriesUsage(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{
		Script: []agentlib.Event{
			{Type: agentlib.EventDelta, Text: "answer"},
			{Type: agentlib.EventDone},
		},
		Usage: agentlib.Usage{InputTokens: 10, OutputTokens: 4, CacheReadTokens: 5, CacheWriteTokens: 3},
	}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for {
		m := readMsg(t, conn)
		if m["type"] != "turn_done" {
			continue
		}
		u, ok := m["usage"].(map[string]any)
		if !ok {
			t.Fatalf("turn_done missing usage: %+v", m)
		}
		if u["input_tokens"] != float64(10) || u["output_tokens"] != float64(4) ||
			u["cache_read_tokens"] != float64(5) || u["cache_write_tokens"] != float64(3) {
			t.Fatalf("usage = %v, want input 10 / output 4 / cache read 5 / cache write 3", u)
		}
		break
	}

	// The assistant message row carries the same breakdown as JSON.
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("conversation not persisted: %v %+v", err, convs)
	}
	msgs, err := d.Store.Messages(convs[0].ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("messages not persisted: %v %+v", err, msgs)
	}
	if got := string(msgs[1].Usage); got != `{"input_tokens":10,"output_tokens":4,"cache_read_tokens":5,"cache_write_tokens":3}` {
		t.Fatalf("persisted usage = %s", got)
	}
}

func TestChatTurnDoneOmitsEmptyUsage(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for {
		m := readMsg(t, conn)
		if m["type"] != "turn_done" {
			continue
		}
		if _, ok := m["usage"]; ok {
			t.Fatalf("turn_done must omit usage when the provider reported none: %+v", m)
		}
		return
	}
}

func TestChatOpenPinsConnectionForNextSend(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	e := New(d)

	// The conversation exists before the socket connects (it was loaded from
	// history, so nothing is replayed and nothing is created).
	id, err := d.Store.CreateConversation("existing conversation")
	if err != nil {
		t.Fatal(err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "open", "conversation_id": id}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "follow up"}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn)["type"] != "turn_done" {
	}

	// The turn ran against the opened conversation's id, and no new
	// conversation was created.
	fake := d.Claude.(*FakeClient)
	if len(fake.Calls) != 1 || fake.Calls[0].SessionID != id {
		t.Fatalf("send after open must use the opened conversation, got %+v", fake.Calls)
	}
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("open must not create a conversation: %v %+v", err, convs)
	}
}

func TestChatOpenUnknownConversation(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "open", "conversation_id": "no-such-conv"}); err != nil {
		t.Fatal(err)
	}
	m := readMsg(t, conn)
	if m["type"] != "error" || m["message"] != "unknown conversation" {
		t.Fatalf("expected unknown-conversation error frame, got %+v", m)
	}
	// An unknown open must not pin the connection: the next send still
	// creates a fresh conversation.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn)["type"] != "turn_done" {
	}
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("expected one new conversation, got %v %+v", err, convs)
	}
}

func TestChatResumePinsConnectionForNextSend(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventDelta, Text: "previous answer"},
		{Type: agentlib.EventDone},
	}}
	e := New(d)

	// First session creates a conversation.
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn1.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn1)["type"] != "turn_done" {
	}
	_ = conn1.Close()

	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("expected one conversation: %v %+v", err, convs)
	}

	// Second session resumes, then sends: the send must continue the resumed
	// conversation rather than create a new one.
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn2.Close() }()
	if err := conn2.WriteJSON(map[string]string{"type": "resume", "conversation_id": convs[0].ID}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn2)["type"] != "turn_done" {
	}
	if err := conn2.WriteJSON(map[string]string{"type": "send", "text": "follow up"}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn2)["type"] != "turn_done" {
	}
	convs, err = d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("send after resume must reuse the conversation, got %v %+v", err, convs)
	}
	msgs, err := d.Store.Messages(convs[0].ID)
	if err != nil || len(msgs) != 4 {
		t.Fatalf("expected 4 messages in the resumed conversation, got %v %+v", err, msgs)
	}
}

// ctxAwareFake respects its context: it blocks until the context is done and
// returns its error, exercising the hub's shutdown wiring.
type ctxAwareFake struct{}

func (ctxAwareFake) Start(ctx context.Context, _, _ string, _ agentlib.EventWriter) (agentlib.Usage, error) {
	<-ctx.Done()
	return agentlib.Usage{}, ctx.Err()
}

// scriptedHangClient blocks on the first Start until cancelled, then streams a
// completed turn on every later call — exercising the supersede path with a
// visible second turn.
type scriptedHangClient struct {
	mu   sync.Mutex
	done bool
}

func (c *scriptedHangClient) Start(ctx context.Context, _, _ string, w agentlib.EventWriter) (agentlib.Usage, error) {
	c.mu.Lock()
	if !c.done {
		c.done = true
		c.mu.Unlock()
		<-ctx.Done()
		return agentlib.Usage{}, ctx.Err()
	}
	c.mu.Unlock()
	if err := w.Write(agentlib.Event{Type: agentlib.EventDelta, Text: "second answer"}); err != nil {
		return agentlib.Usage{}, err
	}
	if err := w.Write(agentlib.Event{Type: agentlib.EventDone}); err != nil {
		return agentlib.Usage{}, err
	}
	return agentlib.Usage{}, nil
}

func TestChatSupersededTurnSuppressesCancelledFrame(t *testing.T) {
	d := testDeps(t)
	d.Claude = &scriptedHangClient{}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "first"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start, got %+v", m)
	}
	// Supersede with a second send that produces a real answer. The
	// superseded turn must not leak a spurious "cancelled" frame into the
	// second turn's stream.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "second"}); err != nil {
		t.Fatal(err)
	}
	for {
		m := readMsg(t, conn)
		switch m["type"] {
		case "error":
			t.Fatalf("spurious error frame: %+v", m)
		case "turn_done":
			return
		}
	}
}

func TestChatHubCancellationEndsTurns(t *testing.T) {
	d := testDeps(t)
	d.Claude = ctxAwareFake{}
	ctx, cancel := context.WithCancel(context.Background())
	d.Ctx = ctx
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "long task"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start, got %+v", m)
	}
	// Cancelling the hub context (server shutdown) must end the turn with a
	// cancelled error frame, exactly like the per-turn cancel path.
	cancel()
	if m := readMsg(t, conn); m["type"] != "error" || m["message"] != "cancelled" {
		t.Fatalf("expected cancelled error after hub cancel, got %+v", m)
	}
}

func TestChatSupersededTurnDoesNotDeleteNewTurn(t *testing.T) {
	d := testDeps(t)
	d.Claude = hangClient{}
	e, hub := newServer(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "first"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start for turn 1, got %+v", m)
	}
	// Supersede: a second send cancels turn 1 and registers turn 2.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "second"}); err != nil {
		t.Fatal(err)
	}
	// Consume the second turn's assistant_start, skipping the first turn's
	// cancelled error whichever order they arrive in.
	for {
		if m := readMsg(t, conn); m["type"] == "assistant_start" {
			break
		}
	}

	// Turn 1 has now finished (its context was cancelled) and its deferred
	// cleanup ran: it must NOT have deleted turn 2's map entry.
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("expected one conversation: %v %+v", err, convs)
	}
	convID := convs[0].ID
	deadline := time.Now().Add(3 * time.Second)
	for {
		hub.mu.Lock()
		t2, ok := hub.turns[convID]
		hub.mu.Unlock()
		if ok && t2.busy {
			break // turn 2's entry survived turn 1's cleanup
		}
		if time.Now().After(deadline) {
			t.Fatalf("superseded turn deleted the new turn's map entry")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Cancel turn 2 so the goroutine exits before the test ends.
	if err := conn.WriteJSON(map[string]string{"type": "cancel"}); err != nil {
		t.Fatal(err)
	}
	for {
		if m := readMsg(t, conn); m["type"] == "error" && m["message"] == "cancelled" {
			break
		}
	}
}

func TestChatTruncatesTitleRuneSafe(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// 70 CJK runes are 210 bytes: a byte-sliced truncation would cut a rune
	// in half and corrupt the title. The title must be exactly 60 runes.
	long := strings.Repeat("日", 70)
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": long}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn)["type"] != "turn_done" {
	}
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("conversation not persisted: %v %+v", err, convs)
	}
	if want := strings.Repeat("日", 60); convs[0].Title != want {
		t.Fatalf("title not truncated to 60 runes: %q (%d bytes)", convs[0].Title, len(convs[0].Title))
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
	defer func() { _ = conn.Close() }()

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

func TestChatNewChatUnpinsConnection(t *testing.T) {
	d := testDeps(t)
	id1, err := d.Store.CreateConversation("first")
	if err != nil {
		t.Fatal(err)
	}
	fake := &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	d.Claude = fake
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Pin to the first conversation and chat in it.
	if err := conn.WriteJSON(map[string]string{"type": "open", "conversation_id": id1}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello one"}); err != nil {
		t.Fatal(err)
	}
	conv1 := waitTurnDone(t, conn)

	// New chat unpins: the next send must create a brand-new conversation.
	if err := conn.WriteJSON(map[string]string{"type": "new_chat"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello two"}); err != nil {
		t.Fatal(err)
	}
	conv2 := waitTurnDone(t, conn)

	if conv1 != id1 {
		t.Fatalf("first turn ran in %q, want %q", conv1, id1)
	}
	if conv2 == id1 || conv2 == "" {
		t.Fatalf("new chat reused the old conversation: %q", conv2)
	}
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 2 {
		t.Fatalf("expected two conversations: %v %+v", err, convs)
	}
	msgs, err := d.Store.Messages(id1)
	if err != nil || len(msgs) != 1 || msgs[0].Content != "hello one" {
		t.Fatalf("first conversation changed: %v %+v", err, msgs)
	}
	if len(fake.Calls) != 2 || fake.Calls[0].SessionID != id1 || fake.Calls[1].SessionID != conv2 {
		t.Fatalf("session ids wrong: %+v", fake.Calls)
	}
}

// waitTurnDone reads frames until turn_done, failing on error frames.
func waitTurnDone(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	for {
		m := readMsg(t, conn)
		switch m["type"] {
		case "error":
			t.Fatalf("unexpected error frame: %+v", m)
		case "turn_done":
			id, _ := m["conversation_id"].(string)
			return id
		}
	}
}

func TestChatNewChatCancelsBusyTurn(t *testing.T) {
	d := testDeps(t)
	d.Claude = hangClient{}
	e, hub := newServer(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// First send starts a hanging turn.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "first"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start, got %+v", m)
	}

	// New chat cancels it: the cancelled frame arrives before anything else.
	if err := conn.WriteJSON(map[string]string{"type": "new_chat"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" || m["message"] != "cancelled" {
		t.Fatalf("expected cancelled error frame, got %+v", m)
	}

	// The next send starts a fresh conversation.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "second"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "assistant_start" {
		t.Fatalf("expected assistant_start for the second turn, got %+v", m)
	}
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 2 {
		t.Fatalf("expected two conversations: %v %+v", err, convs)
	}

	// End the hanging second turn.
	if err := conn.WriteJSON(map[string]string{"type": "cancel"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		hub.mu.Lock()
		busy := 0
		for _, t := range hub.turns {
			if t.busy {
				busy++
			}
		}
		hub.mu.Unlock()
		if busy == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("turns did not drain after cancel")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestChatForwardsThinkingFrames(t *testing.T) {
	d := testDeps(t)
	fake := &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventThinking},
		{Type: agentlib.EventDelta, Text: "answer"},
		{Type: agentlib.EventDone},
	}}
	d.Claude = fake
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "question"}); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	var convID string
	for {
		m := readMsg(t, conn)
		switch m["type"] {
		case "error":
			t.Fatalf("unexpected error frame: %+v", m)
		case "turn_done":
			convID, _ = m["conversation_id"].(string)
			goto done
		default:
			got = append(got, m["type"].(string))
		}
	}
done:
	want := []string{"assistant_start", "assistant_thinking", "assistant_delta"}
	if len(got) != len(want) {
		t.Fatalf("frames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("frame %d = %q, want %q (%v)", i, got[i], want[i], got)
		}
	}
	// A reconnect replay must include the thinking frame (resume keeps the
	// "Thinking…" state for mid-thinking reconnects).
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial2: %v", err)
	}
	defer func() { _ = conn2.Close() }()
	if err := conn2.WriteJSON(map[string]string{"type": "resume", "conversation_id": convID}); err != nil {
		t.Fatal(err)
	}
	replayed := []string{}
	for {
		m := readMsg(t, conn2)
		switch m["type"] {
		case "turn_done":
			goto replayDone
		default:
			replayed = append(replayed, m["type"].(string))
		}
	}
replayDone:
	if len(replayed) == 0 || replayed[0] != "assistant_thinking" {
		t.Fatalf("replay = %v, want the thinking frame first", replayed)
	}
}

// TestChatToolTurnFrameSequence pins the exact WS frame order for a
// tool-using turn: start, tool activity, deltas, then done — the sequence
// the UI renders and a reconnect must replay.
func TestChatToolTurnFrameSequence(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{
		{Type: agentlib.EventTool, Tool: "read_file", Detail: `{"path":"note.md"}`},
		{Type: agentlib.EventDelta, Text: "here it is"},
		{Type: agentlib.EventDone},
	}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "read my note"}); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for {
		m := readMsg(t, conn)
		switch m["type"] {
		case "error":
			t.Fatalf("unexpected error frame: %+v", m)
		case "turn_done":
			goto done
		default:
			got = append(got, m["type"].(string))
		}
	}
done:
	want := []string{"assistant_start", "tool_activity", "assistant_delta"}
	if len(got) != len(want) {
		t.Fatalf("frames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("frame %d = %q, want %q (%v)", i, got[i], want[i], got)
		}
	}
	// A reconnect replay must include the tool frame in the same position.
	convs, err := d.Store.ListConversations()
	if err != nil || len(convs) != 1 {
		t.Fatalf("conversation not persisted: %v %+v", err, convs)
	}
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial2: %v", err)
	}
	defer func() { _ = conn2.Close() }()
	if err := conn2.WriteJSON(map[string]string{"type": "resume", "conversation_id": convs[0].ID}); err != nil {
		t.Fatal(err)
	}
	replayed := []string{}
	for {
		m := readMsg(t, conn2)
		switch m["type"] {
		case "turn_done":
			goto replayDone
		default:
			replayed = append(replayed, m["type"].(string))
		}
	}
replayDone:
	// The replay reflects the recorded frames only (assistant_start is not
	// replayed): the tool frame must come before the delta, as sent.
	if len(replayed) == 0 || replayed[0] != "tool_activity" {
		t.Fatalf("replay = %v, want the turn replayed from tool_activity", replayed)
	}
	if replayed[1] != "assistant_delta" {
		t.Fatalf("replay = %v, want assistant_delta after the tool frame", replayed)
	}
}

// TestChatAcceptsPresenceFrames guards the WS protocol contract: the frontend
// keeps sending presence frames (Page Visibility), and even though the native
// agent has no idle processes to flush they must be accepted — never an
// unknown-type error.
func TestChatAcceptsPresenceFrames(t *testing.T) {
	d := testDeps(t)
	d.Claude = &FakeClient{Script: []agentlib.Event{{Type: agentlib.EventDone}}}
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]any{"type": "presence", "active": false}); err != nil {
		t.Fatal(err)
	}
	// The connection stays usable after the presence frame.
	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	for readMsg(t, conn)["type"] != "turn_done" {
	}
}
