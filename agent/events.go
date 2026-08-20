// Package agent is the reusable, provider-agnostic core of Thoth's native
// Go agent: normalized conversation model, streaming events, and the
// provider/tool seams. It has no imports outside the standard library, so any
// Go project can embed it.
package agent

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
