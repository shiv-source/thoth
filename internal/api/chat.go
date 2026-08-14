package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/claude"
	"github.com/shiv-source/thoth/internal/store"
)

// clientMsg / serverMsg are the wire protocol (see Task 13 interfaces).
type clientMsg struct {
	Type           string `json:"type"`
	Text           string `json:"text,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type serverMsg struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Message string `json:"message,omitempty"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// local-only app; the Vite dev server origin must be allowed too
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub tracks one active turn per conversation and replays the latest turn's
// messages on resume so reconnects re-sync the UI.
type Hub struct {
	claude claude.Client
	store  *store.Store
	log    *slog.Logger

	mu     sync.Mutex
	turns  map[string]*turn
	recent map[string][]serverMsg // latest turn only, capped per conversation
}

type turn struct {
	cancel context.CancelFunc
	busy   bool
}

func NewHub(c claude.Client, st *store.Store, log *slog.Logger) *Hub {
	return &Hub{
		claude: c,
		store:  st,
		log:    log,
		turns:  map[string]*turn{},
		recent: map[string][]serverMsg{},
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
	defer ws.Close()

	// Turns run in their own goroutines (the read loop must stay free to
	// receive cancel frames), so every socket write funnels through one
	// writer goroutine to stay race-free.
	out := make(chan serverMsg, 64)
	quit := make(chan struct{})
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
	write := func(m serverMsg) {
		select {
		case out <- m:
		case <-quit:
		}
	}

	convID := ""
	for {
		var msg clientMsg
		if err := ws.ReadJSON(&msg); err != nil {
			close(quit)
			return nil // client went away
		}
		switch msg.Type {
		case "send":
			if convID == "" {
				title := msg.Text
				if len(title) > 60 {
					title = title[:60]
				}
				id, err := h.store.CreateConversation(title)
				if err != nil {
					write(serverMsg{Type: "error", Message: err.Error()})
					continue
				}
				convID = id
			}
			if err := h.store.AddMessage(convID, "user", msg.Text); err != nil {
				write(serverMsg{Type: "error", Message: err.Error()})
				continue
			}
			go h.runTurn(convID, msg.Text, write)
		case "cancel":
			h.cancelTurn(convID)
		case "resume":
			if buf := h.replay(msg.ConversationID); len(buf) > 0 {
				for _, m := range buf {
					write(m)
				}
			} else {
				write(serverMsg{Type: "error", Message: "unknown conversation"})
			}
		default:
			write(serverMsg{Type: "error", Message: "unknown message type: " + msg.Type})
		}
	}
}

// runTurn runs one Claude turn. A new send supersedes an in-flight turn for
// the same conversation: the old context is cancelled and the replay buffer
// resets so resume always reflects the latest turn only.
func (h *Hub) runTurn(convID, prompt string, write func(serverMsg)) {
	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	if t, ok := h.turns[convID]; ok && t.busy {
		t.cancel()
	}
	h.turns[convID] = &turn{cancel: cancel, busy: true}
	h.recent[convID] = nil
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.turns, convID)
		h.mu.Unlock()
	}()

	write(serverMsg{Type: "assistant_start"})
	var sb strings.Builder
	w := claude.WriterFunc(func(ev claude.Event) error {
		switch ev.Type {
		case claude.EventDelta:
			sb.WriteString(ev.Text)
			write(serverMsg{Type: "assistant_delta", Text: ev.Text})
			h.record(convID, serverMsg{Type: "assistant_delta", Text: ev.Text})
		case claude.EventTool:
			write(serverMsg{Type: "tool_activity", Tool: ev.Tool, Detail: ev.Detail})
			h.record(convID, serverMsg{Type: "tool_activity", Tool: ev.Tool, Detail: ev.Detail})
		case claude.EventError:
			write(serverMsg{Type: "error", Message: ev.Detail})
			h.record(convID, serverMsg{Type: "error", Message: ev.Detail})
		}
		return nil
	})
	err := h.claude.Start(ctx, convID, prompt, w)
	switch {
	case errors.Is(err, context.Canceled):
		write(serverMsg{Type: "error", Message: "cancelled"})
		return
	case err != nil:
		write(serverMsg{Type: "error", Message: err.Error()})
		h.record(convID, serverMsg{Type: "error", Message: err.Error()})
		return
	}
	if sb.Len() > 0 {
		if err := h.store.AddMessage(convID, "assistant", sb.String()); err != nil {
			h.log.Warn("persist assistant message", "err", err)
		}
	}
	m := serverMsg{Type: "turn_done"}
	write(m)
	h.record(convID, m)
}
