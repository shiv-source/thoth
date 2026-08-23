package api

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-warehouse/events"
	"github.com/gorilla/websocket"
	syncsvc "github.com/shiv-source/thoth/internal/sync"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestHubBroadcastDeliversToClients(t *testing.T) {
	d := testDeps(t)
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background())
	a, b := make(chan serverMsg, 64), make(chan serverMsg, 64)
	hub.addClient(a)
	hub.addClient(b)

	hub.Broadcast(serverMsg{Type: "wiki_changed", Changes: []wiki.Change{
		{Op: wiki.OpWrite, Path: "notes/a.md"},
	}})

	for name, out := range map[string]chan serverMsg{"a": a, "b": b} {
		select {
		case got := <-out:
			if got.Type != "wiki_changed" || len(got.Changes) != 1 ||
				got.Changes[0].Op != wiki.OpWrite || got.Changes[0].Path != "notes/a.md" {
				t.Fatalf("client %s: unexpected frame %+v", name, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %s: no frame within 1s", name)
		}
	}
}

func TestHubBroadcastDropsSlowClient(t *testing.T) {
	d := testDeps(t)
	hub := NewHub(d.Claude, d.Store, d.Log, context.Background())

	// A slow client whose write buffer is full (a dead socket that never
	// drains) must be skipped, never blocking the broadcast.
	slow := make(chan serverMsg, 64)
	for i := 0; i < cap(slow); i++ {
		slow <- serverMsg{Type: "assistant_delta", Text: "backlog"}
	}
	fast := make(chan serverMsg, 64)
	hub.addClient(slow)
	hub.addClient(fast)

	hub.Broadcast(serverMsg{Type: "wiki_changed"})

	select {
	case got := <-fast:
		if got.Type != "wiki_changed" {
			t.Fatalf("fast client: unexpected frame %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("fast client: no frame within 1s")
	}
	for i := 0; i < cap(slow); i++ {
		if m := <-slow; m.Type == "wiki_changed" {
			t.Fatal("slow client must not receive the broadcast")
		}
	}

	// A removed client receives nothing further.
	hub.removeClient(fast)
	hub.Broadcast(serverMsg{Type: "wiki_changed"})
	select {
	case m := <-fast:
		t.Fatalf("removed client received %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSyncResultFrameReachesSocket(t *testing.T) {
	d := testDeps(t)
	bus := events.New(events.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	d.Events = bus
	ctx := context.Background()
	t.Cleanup(func() { bus.Close(); _ = bus.Wait(ctx) })
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Roundtrip first: a reply proves the connection is registered with the
	// hub, so the following publish cannot race client registration.
	if err := conn.WriteJSON(map[string]string{"type": "bogus"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" {
		t.Fatalf("expected error reply, got %+v", m)
	}

	if err := bus.Publish(ctx, syncsvc.Result{
		ConnectionID: 3, Name: "backup", OK: false, Error: "no credentials stored",
	}); err != nil {
		t.Fatal(err)
	}

	m := readMsg(t, conn)
	if m["type"] != "sync_result" {
		t.Fatalf("expected sync_result frame, got %+v", m)
	}
	payload, ok := m["sync_result"].(map[string]any)
	if !ok || payload["connection_id"] != float64(3) || payload["name"] != "backup" ||
		payload["ok"] != false || payload["error"] != "no credentials stored" {
		t.Fatalf("sync_result payload = %+v", m["sync_result"])
	}
}

func TestWikiChangedFrameReachesSocket(t *testing.T) {
	d := testDeps(t)
	bus := events.New(events.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	d.Events = bus
	ctx := context.Background()
	t.Cleanup(func() { bus.Close(); _ = bus.Wait(ctx) })
	e := New(d)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(t, e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Roundtrip first: a reply proves the connection is registered with the
	// hub, so the following publish cannot race client registration.
	if err := conn.WriteJSON(map[string]string{"type": "bogus"}); err != nil {
		t.Fatal(err)
	}
	if m := readMsg(t, conn); m["type"] != "error" {
		t.Fatalf("expected error reply, got %+v", m)
	}

	if err := bus.Publish(ctx, wiki.Changed{Changes: []wiki.Change{
		{Op: wiki.OpWrite, Path: "notes/a.md"},
		{Op: wiki.OpRemove, Path: "old.md"},
	}}); err != nil {
		t.Fatal(err)
	}

	m := readMsg(t, conn)
	if m["type"] != "wiki_changed" {
		t.Fatalf("expected wiki_changed frame, got %+v", m)
	}
	changes, ok := m["changes"].([]any)
	if !ok || len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %+v", m["changes"])
	}
	first, ok := changes[0].(map[string]any)
	if !ok || first["op"] != "write" || first["path"] != "notes/a.md" {
		t.Fatalf("first change = %+v", changes[0])
	}
}
