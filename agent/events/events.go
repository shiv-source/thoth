// Package events holds the normalized turn events a host forwards to its UI:
// one Event per streaming step, decoupled from any provider wire format. The
// type names and fields are stable public API — the WS protocol and the
// frontend depend on them.
package events

// EventType enumerates the kinds of events a turn can emit.
type EventType string

const (
	EventDelta    EventType = "assistant_delta"
	EventTool     EventType = "tool_activity"
	EventThinking EventType = "thinking"
	EventDone     EventType = "turn_done"
	EventError    EventType = "error"
)

// Event is one normalized turn event, decoupled from any provider wire format.
// The type name and fields are stable public API: the WS protocol and the
// frontend depend on them.
type Event struct {
	Type   EventType
	Text   string // assistant_delta payload
	Tool   string // tool_activity: tool name
	Detail string // tool_activity: tool input; error: message
}

// EventWriter receives the events of a turn as they stream.
type EventWriter interface {
	Write(Event) error
}

// WriterFunc adapts a plain function to EventWriter.
type WriterFunc func(Event) error

// Write implements EventWriter.
func (f WriterFunc) Write(e Event) error { return f(e) }
