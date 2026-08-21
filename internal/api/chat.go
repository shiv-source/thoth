package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

// Client starts one turn for a conversation and streams normalized events,
// returning the turn's token usage. The native agent host (internal/agent)
// implements it; defining the seam at the consumer keeps the api layer free of
// a concrete chat implementation.
type Client interface {
	Start(ctx context.Context, sessionID, prompt string, w agentlib.EventWriter) (agentlib.Usage, error)
}

// clientMsg / serverMsg are the wire protocol (see Task 13 interfaces).
type clientMsg struct {
	Type           string `json:"type"`
	Text           string `json:"text,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	Active         *bool  `json:"active,omitempty"` // presence frame: carried for protocol compatibility, ignored server-side
}

type serverMsg struct {
	Type           string          `json:"type"`
	Text           string          `json:"text,omitempty"`
	Tool           string          `json:"tool,omitempty"`
	Detail         string          `json:"detail,omitempty"`
	Message        string          `json:"message,omitempty"`
	ConversationID string          `json:"conversation_id,omitempty"`
	Changes        []wiki.Change   `json:"changes,omitempty"`
	Usage          *agentlib.Usage `json:"usage,omitempty"` // turn_done only; nil when the provider reported none
}

// usagePtr returns u as a pointer when any counter is non-zero, so a turn
// whose provider reported no usage carries no usage field on the frame.
func usagePtr(u agentlib.Usage) *agentlib.Usage {
	if u == (agentlib.Usage{}) {
		return nil
	}
	return &u
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Local-only app: allow browser connections from the Vite dev server and
	// origin-less clients (curl, tests, the CLI); reject everything else so
	// a random web page cannot drive the chat API.
	CheckOrigin: allowLocalOrigin,
}

// allowLocalOrigin admits requests without an Origin header (non-browser
// clients) and origins whose host is localhost, 127.0.0.1, or ::1 on any
// port. Browsers always send Origin on cross-site requests; the Vite dev
// server (http://localhost:5173) is the only origin a real browser needs.
func allowLocalOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// Hub tracks one active turn per conversation and replays the latest turn's
// messages on resume so reconnects re-sync the UI.
type Hub struct {
	client Client
	store  *store.Store
	log    *slog.Logger
	ctx    context.Context // cancelled on server shutdown to reap in-flight turns

	mu     sync.Mutex
	turns  map[string]*turn
	recent map[string][]serverMsg // latest turn only, capped per conversation
	// clients are the connected chat sockets, keyed by their write channel;
	// Broadcast targets them for server-push frames (wiki_changed).
	clients map[chan serverMsg]*clientEntry
}

// clientEntry is one connected chat socket.
type clientEntry struct {
	out chan serverMsg
}

type turn struct {
	cancel context.CancelFunc
	busy   bool
}

// NewHub builds a Hub whose turns derive from ctx; cancelling ctx (the server
// shutdown signal) cancels every in-flight turn, so SIGTERM stops provider
// streams before the server exits. A nil ctx means background turns.
func NewHub(c Client, st *store.Store, log *slog.Logger, ctx context.Context) *Hub {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Hub{
		client:  c,
		store:   st,
		log:     log,
		ctx:     ctx,
		turns:   map[string]*turn{},
		recent:  map[string][]serverMsg{},
		clients: map[chan serverMsg]*clientEntry{},
	}
}

func (h *Hub) record(convID string, m serverMsg) {
	h.mu.Lock()
	defer h.mu.Unlock()
	buf := append(h.recent[convID], m)
	if len(buf) > 500 {
		buf = buf[len(buf)-500:]
	}
	h.recent[convID] = buf
}

func (h *Hub) replay(convID string) []serverMsg {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]serverMsg(nil), h.recent[convID]...)
}

// addClient registers a connection's write buffer so Broadcast reaches it.
func (h *Hub) addClient(out chan serverMsg) {
	h.mu.Lock()
	h.clients[out] = &clientEntry{out: out}
	h.mu.Unlock()
}

// removeClient drops a connection from the broadcast registry.
func (h *Hub) removeClient(out chan serverMsg) {
	h.mu.Lock()
	delete(h.clients, out)
	h.mu.Unlock()
}

// Broadcast fans a server-push frame out to every connected client.
// Delivery is non-blocking: a client whose write buffer is full is skipped
// (the frame is dropped for it, and the next push catches up), so a dead
// socket can never stall the publisher.
func (h *Hub) Broadcast(m serverMsg) {
	h.mu.Lock()
	outs := make([]chan serverMsg, 0, len(h.clients))
	for _, e := range h.clients {
		outs = append(outs, e.out)
	}
	h.mu.Unlock()
	for _, out := range outs {
		select {
		case out <- m:
		default:
		}
	}
}

func (h *Hub) cancelTurn(convID string) {
	h.mu.Lock()
	t, ok := h.turns[convID]
	h.mu.Unlock()
	if ok && t.busy {
		t.cancel()
	}
}

func (h *Hub) chat(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = ws.Close() }()

	// Turns run in their own goroutines (the read loop must stay free to
	// receive cancel frames), so every socket write funnels through one
	// writer goroutine to stay race-free.
	write, out, quit := writeLoop(ws)
	h.addClient(out)
	defer h.removeClient(out)

	convID := ""
	for {
		var msg clientMsg
		if err := ws.ReadJSON(&msg); err != nil {
			close(quit)
			return nil // client went away
		}
		switch msg.Type {
		case "send":
			h.handleSend(&convID, msg.Text, write)
		case "cancel":
			h.cancelTurn(convID)
		case "resume":
			// A successful resume pins the connection to that conversation so
			// a following send continues it instead of starting a new one.
			if h.handleResume(msg.ConversationID, write) {
				convID = msg.ConversationID
			}
		case "open":
			// Pin the connection to an existing conversation so the next send
			// continues it — no replay, no other effect (unlike resume).
			if h.conversationExists(msg.ConversationID) {
				convID = msg.ConversationID
			} else {
				write(serverMsg{Type: "error", Message: "unknown conversation"})
			}
		case "new_chat":
			// Unpin the connection: cancel the in-flight turn first so its
			// stream does not bleed into the fresh chat, then let the next
			// send create a brand-new conversation.
			h.cancelTurn(convID)
			convID = ""
		case "presence":
			// The tab reported its visibility. The native agent keeps no idle
			// processes to flush, so presence carries no server-side effect;
			// the frame is still accepted so the WS protocol is unchanged.
		default:
			write(serverMsg{Type: "error", Message: "unknown message type: " + msg.Type})
		}
	}
}

// writeLoop owns the socket: every write goes through it so concurrent turn
// goroutines never race on the connection. The returned write blocks until
// the message is handed off (or the loop has quit); out is the buffer the
// caller registers with the hub so Broadcast can target this connection.
func writeLoop(ws *websocket.Conn) (write func(serverMsg), out chan serverMsg, quit chan struct{}) {
	out = make(chan serverMsg, 64)
	quit = make(chan struct{})
	go func() {
		for {
			select {
			case m := <-out:
				_ = ws.WriteJSON(m) // keep draining on write errors; the read loop notices the dead conn
			case <-quit:
				return
			}
		}
	}()
	write = func(m serverMsg) {
		select {
		case out <- m:
		case <-quit:
		}
	}
	return write, out, quit
}

// handleSend persists the user message and starts a turn, creating the
// conversation on first send. On any error it reports over the socket and
// leaves the conversation untouched for the next message.
func (h *Hub) handleSend(convID *string, text string, write func(serverMsg)) {
	if *convID == "" {
		id, err := h.store.CreateConversation(truncateTitle(text))
		if err != nil {
			write(serverMsg{Type: "error", Message: err.Error()})
			return
		}
		*convID = id
	}
	if err := h.store.AddMessage(*convID, "user", text); err != nil {
		write(serverMsg{Type: "error", Message: err.Error()})
		return
	}
	go h.runTurn(*convID, text, write)
}

// handleResume replays the latest turn of a conversation so a reconnect can
// re-sync the UI. It reports whether the conversation existed.
func (h *Hub) handleResume(convID string, write func(serverMsg)) bool {
	if buf := h.replay(convID); len(buf) > 0 {
		for _, m := range buf {
			write(m)
		}
		return true
	}
	write(serverMsg{Type: "error", Message: "unknown conversation"})
	return false
}

// conversationExists reports whether the store knows the conversation. The
// local app keeps conversations in a list query, so a scan is plenty.
func (h *Hub) conversationExists(convID string) bool {
	convs, err := h.store.ListConversations()
	if err != nil {
		return false
	}
	for _, c := range convs {
		if c.ID == convID {
			return true
		}
	}
	return false
}

// truncateTitle caps a conversation title at 60 runes; slicing the raw
// string would split a multi-byte UTF-8 sequence in half.
func truncateTitle(s string) string {
	r := []rune(s)
	if len(r) > 60 {
		return string(r[:60])
	}
	return s
}

// runTurn runs one agent turn. A new send supersedes an in-flight turn for
// the same conversation: the old context is cancelled and the replay buffer
// resets so resume always reflects the latest turn only. The turn context is
// derived from the hub context, so server shutdown cancels in-flight turns
// before the server exits.
func (h *Hub) runTurn(convID, prompt string, write func(serverMsg)) {
	ctx, cancel := context.WithCancel(h.ctx)
	t := &turn{cancel: cancel, busy: true}
	h.mu.Lock()
	if prev, ok := h.turns[convID]; ok && prev.busy {
		prev.cancel()
	}
	h.turns[convID] = t
	h.recent[convID] = nil
	h.mu.Unlock()
	// Delete only if this turn is still the registered one: a superseding
	// turn replaces the map entry, and deleting here would wipe the new
	// turn's slot.
	defer func() {
		h.mu.Lock()
		if h.turns[convID] == t {
			delete(h.turns, convID)
		}
		h.mu.Unlock()
	}()

	write(serverMsg{Type: "assistant_start"})
	var sb strings.Builder
	usage, err := h.client.Start(ctx, convID, prompt, h.turnWriter(&sb, write, convID))
	if h.finishTurn(convID, err, write) {
		return
	}
	h.persistTurn(convID, sb.String())
	m := serverMsg{Type: "turn_done", ConversationID: convID, Usage: usagePtr(usage)}
	// Record before writing: once the client has seen this frame it may
	// already be resuming, and the replay must include everything sent.
	h.record(convID, m)
	write(m)
}

// turnWriter bridges agent events to the socket and the replay buffer.
func (h *Hub) turnWriter(sb *strings.Builder, write func(serverMsg), convID string) agentlib.WriterFunc {
	return agentlib.WriterFunc(func(ev agentlib.Event) error {
		switch ev.Type {
		case agentlib.EventDelta:
			sb.WriteString(ev.Text)
			h.record(convID, serverMsg{Type: "assistant_delta", Text: ev.Text})
			write(serverMsg{Type: "assistant_delta", Text: ev.Text})
		case agentlib.EventThinking:
			// The thinking text rides the frame so the UI can show what the
			// model is working on. Recorded before writing, so a reconnect
			// mid-thinking resumes the exact state that was shown.
			h.record(convID, serverMsg{Type: "assistant_thinking", Text: ev.Text})
			write(serverMsg{Type: "assistant_thinking", Text: ev.Text})
		case agentlib.EventTool:
			h.record(convID, serverMsg{Type: "tool_activity", Tool: ev.Tool, Detail: ev.Detail})
			write(serverMsg{Type: "tool_activity", Tool: ev.Tool, Detail: ev.Detail})
		case agentlib.EventError:
			h.record(convID, serverMsg{Type: "error", Message: ev.Detail})
			write(serverMsg{Type: "error", Message: ev.Detail})
		}
		return nil
	})
}

// finishTurn reports a cancelled or failed turn and reports whether the turn
// ended prematurely (and must not persist output).
func (h *Hub) finishTurn(convID string, err error, write func(serverMsg)) bool {
	switch {
	case errors.Is(err, context.Canceled):
		write(serverMsg{Type: "error", Message: "cancelled"})
		return true
	case err != nil:
		h.record(convID, serverMsg{Type: "error", Message: err.Error()})
		write(serverMsg{Type: "error", Message: err.Error()})
		return true
	}
	return false
}

// persistTurn stores a completed assistant answer (no-op when the turn
// produced no text).
func (h *Hub) persistTurn(convID, text string) {
	if text == "" {
		return
	}
	if err := h.store.AddMessage(convID, "assistant", text); err != nil {
		h.log.Warn("persist assistant message", "err", err)
	}
}
