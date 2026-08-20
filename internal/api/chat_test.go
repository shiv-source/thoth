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

	"github.com/google/uuid"
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
	d.Claude = &claude.FakeClient{Script: []claude.Event{
		{Type: claude.EventTool, Tool: "Read", Detail: "note.md"},
		{Type: claude.EventDelta, Text: "answer"},
		{Type: claude.EventDone},
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

func (hangClient) Start(ctx context.Context, _, _ string, _ claude.EventWriter, _ ...claude.StartOption) error {
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
	d.Claude = &claude.FakeClient{Err: errors.New("boom")}
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
	d.Claude = &claude.FakeClient{Script: []claude.Event{{Type: claude.EventDone}}}
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
	d.Claude = &claude.FakeClient{Script: []claude.Event{{Type: claude.EventDone}}}
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

func TestChatOpenPinsConnectionForNextSend(t *testing.T) {
	d := testDeps(t)
	d.Claude = &claude.FakeClient{Script: []claude.Event{{Type: claude.EventDone}}}
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
	fake := d.Claude.(*claude.FakeClient)
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
	d.Claude = &claude.FakeClient{Script: []claude.Event{
		{Type: claude.EventDelta, Text: "previous answer"},
		{Type: claude.EventDone},
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
// returns its error, exercising the hub's shutdown wiring (the real CLI
// client behaves the same way via exec.CommandContext).
type ctxAwareFake struct{}

func (ctxAwareFake) Start(ctx context.Context, _, _ string, _ claude.EventWriter, _ ...claude.StartOption) error {
	<-ctx.Done()
	return ctx.Err()
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
	d.Claude = &claude.FakeClient{Script: []claude.Event{{Type: claude.EventDone}}}
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

// staleLockClient fails its first Start with the CLI's stale-lock error
// (what a killed turn leaves behind), then delegates to a FakeClient so the
// fork retry can succeed and be inspected.
type staleLockClient struct {
	fake  claude.FakeClient
	first string
	calls int
}

func (s *staleLockClient) Start(ctx context.Context, sessionID, prompt string, w claude.EventWriter, opts ...claude.StartOption) error {
	s.calls++
	if s.calls == 1 {
		s.first = sessionID
		return errors.New("claude exited: exit status 1 (stderr: Error: Session ID abc is already in use. )")
	}
	return s.fake.Start(ctx, sessionID, prompt, w, opts...)
}

func TestChatRotatesStaleSessionID(t *testing.T) {
	d := testDeps(t)
	stale := &staleLockClient{fake: claude.FakeClient{Script: []claude.Event{
		{Type: claude.EventDelta, Text: "recovered"},
		{Type: claude.EventDone},
	}}}
	d.Claude = stale
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteJSON(map[string]string{"type": "send", "text": "hello"}); err != nil {
		t.Fatal(err)
	}
	var convID string
	for {
		m := readMsg(t, conn)
		switch m["type"] {
		case "error":
			t.Fatalf("turn must recover from the stale lock, got %+v", m)
		case "turn_done":
			convID, _ = m["conversation_id"].(string)
			goto done
		}
	}
done:
	if stale.first == "" || convID == "" || stale.first != convID {
		t.Fatalf("first attempt session = %q, conversation = %q (a fresh conversation seeds its own id)", stale.first, convID)
	}
	// The retry forked into a fresh valid id resumed from the locked one,
	// and the store now holds the fork.
	if len(stale.fake.Calls) != 1 {
		t.Fatalf("expected exactly one retry call, got %d", len(stale.fake.Calls))
	}
	forked := stale.fake.Calls[0]
	if forked.SessionID == stale.first || forked.Resume != stale.first {
		t.Fatalf("retry call = %+v, want fresh id resumed from %q", forked, stale.first)
	}
	if _, err := uuid.Parse(forked.SessionID); err != nil {
		t.Fatalf("forked session id %q is not a valid UUID: %v", forked.SessionID, err)
	}
	sid, err := d.Store.ConversationSessionID(convID)
	if err != nil || sid != forked.SessionID {
		t.Fatalf("stored session id = %q/%v, want %q", sid, err, forked.SessionID)
	}
}

func TestChatNewChatUnpinsConnection(t *testing.T) {
	d := testDeps(t)
	id1, err := d.Store.CreateConversation("first")
	if err != nil {
		t.Fatal(err)
	}
	fake := &claude.FakeClient{Script: []claude.Event{{Type: claude.EventDone}}}
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
	fake := &claude.FakeClient{Script: []claude.Event{
		{Type: claude.EventThinking},
		{Type: claude.EventDelta, Text: "answer"},
		{Type: claude.EventDone},
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

func TestHubSessionIDUsesRotated(t *testing.T) {
	d := testDeps(t)
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background(), 0, nil)

	id, err := d.Store.CreateConversation("s")
	if err != nil {
		t.Fatal(err)
	}
	// A rotated (forked) session id is the one a turn — and the boot prewarm —
	// must use, never the conversation id.
	if err := d.Store.SetClaudeSessionID(id, "rotated-session-uuid"); err != nil {
		t.Fatal(err)
	}
	if got := hub.sessionID(id); got != "rotated-session-uuid" {
		t.Fatalf("sessionID = %q, want rotated-session-uuid", got)
	}
	// A fresh conversation seeds its own id as the session.
	id2, err := d.Store.CreateConversation("t")
	if err != nil {
		t.Fatal(err)
	}
	if got := hub.sessionID(id2); got != id2 {
		t.Fatalf("sessionID(fresh) = %q, want %q", got, id2)
	}
}

func TestHubRelaxFlushesWhenLastClientGoesAway(t *testing.T) {
	d := testDeps(t)
	flushed := make(chan struct{}, 1)
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background(), 50*time.Millisecond, func() { flushed <- struct{}{} })

	out := make(chan serverMsg, 64)
	hub.addClient(out)
	// A connected, visible client counts as active: nothing arms.
	select {
	case <-flushed:
		t.Fatal("flushed while a client is active")
	case <-time.After(120 * time.Millisecond):
	}
	// Hiding the last tab arms the relaxation timer.
	hub.setPresence(out, false)
	select {
	case <-flushed:
	case <-time.After(5 * time.Second):
		t.Fatal("relaxation flush did not fire after going away")
	}
}

func TestHubRelaxCancelledOnReturn(t *testing.T) {
	d := testDeps(t)
	flushed := make(chan struct{}, 1)
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background(), 100*time.Millisecond, func() { flushed <- struct{}{} })

	out := make(chan serverMsg, 64)
	hub.addClient(out)
	hub.setPresence(out, false) // away: arms the timer
	time.Sleep(30 * time.Millisecond)
	hub.setPresence(out, true) // back before the timeout: timer cancels
	select {
	case <-flushed:
		t.Fatal("flush fired after the client returned")
	case <-time.After(200 * time.Millisecond):
	}
	// Leaving again restarts the timer.
	hub.setPresence(out, false)
	select {
	case <-flushed:
	case <-time.After(5 * time.Second):
		t.Fatal("second relaxation flush did not fire")
	}
}

func TestHubRelaxOnDisconnect(t *testing.T) {
	d := testDeps(t)
	flushed := make(chan struct{}, 1)
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background(), 50*time.Millisecond, func() { flushed <- struct{}{} })

	out := make(chan serverMsg, 64)
	hub.addClient(out)
	hub.removeClient(out) // last socket closed (navigated away)
	select {
	case <-flushed:
	case <-time.After(5 * time.Second):
		t.Fatal("relaxation flush did not fire after disconnect")
	}
}

func TestChatPresenceFrameArmsRelaxation(t *testing.T) {
	d := testDeps(t)
	flushed := make(chan struct{}, 1)
	d.RelaxTimeout = 80 * time.Millisecond
	d.OnChatAway = func() { flushed <- struct{}{} }
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Hiding the tab (Page Visibility API) sends the presence frame, which
	// arms the relaxation flush after the timeout.
	if err := conn.WriteJSON(map[string]any{"type": "presence", "active": false}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-flushed:
	case <-time.After(5 * time.Second):
		t.Fatal("away presence frame did not trigger the relaxation flush")
	}

	// A second, visible client keeps the pool warm: nothing may flush while it
	// is connected even though the first is still hidden.
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial second: %v", err)
	}
	defer func() { _ = conn2.Close() }()
	select {
	case <-flushed:
		t.Fatal("flush fired while a visible client is connected")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestHubFireAwaySkipsWhenClientActive(t *testing.T) {
	d := testDeps(t)
	called := 0
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background(), 50*time.Millisecond, func() { called++ })
	out := make(chan serverMsg, 64)
	hub.addClient(out) // a visible client is connected
	hub.fireAway()     // the "client returned just as the timer fired" guard
	if called != 0 {
		t.Fatalf("OnAway ran with an active client: %d", called)
	}
	// With no client at all, the same path runs the callback.
	hub.removeClient(out)
	hub.fireAway()
	if called != 1 {
		t.Fatalf("OnAway did not run with no clients: %d", called)
	}
}
