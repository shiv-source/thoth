package claude

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrIgnore signals a stream line that carries no event worth forwarding.
var ErrIgnore = errors.New("ignore line")

type EventType string

const (
	EventDelta    EventType = "assistant_delta"
	EventTool     EventType = "tool_activity"
	EventThinking EventType = "thinking"
	EventDone     EventType = "turn_done"
	EventError    EventType = "error"
)

type Event struct {
	Type   EventType
	Text   string // assistant_delta payload
	Tool   string // tool_activity: tool name
	Detail string // tool_activity: tool input; error: message
}

type EventWriter interface {
	Write(Event) error
}

// WriterFunc adapts a plain function to EventWriter.
type WriterFunc func(Event) error

func (f WriterFunc) Write(e Event) error { return f(e) }

// rawLine is the tolerant view of one stream-json line. Unknown fields are
// ignored by design so the parser survives CLI output additions. Message is
// kept raw: the CLI emits it as an object on assistant lines but as a string
// on others (e.g. system permission_denied or --include-partial-messages
// "message" lines), so it is decoded lazily — only for assistant lines.
type rawLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	IsError bool            `json:"is_error"`
	Result  string          `json:"result"`
}

type rawMsg struct {
	Content []rawBlock `json:"content"`
}

type rawBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ParseLine converts one stream-json line into zero or one Event.
// Lines that carry no event return ErrIgnore.
func ParseLine(line []byte) (Event, error) {
	var raw rawLine
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, fmt.Errorf("parse stream line: %w", err)
	}
	switch raw.Type {
	case "assistant":
		if len(raw.Message) == 0 {
			return Event{}, ErrIgnore
		}
		var msg rawMsg
		if err := json.Unmarshal(raw.Message, &msg); err != nil {
			return Event{}, ErrIgnore // wrong-shaped payload: not a delta
		}
		// Text and tool blocks win over thinking; only a content made solely
		// of thinking blocks surfaces as a thinking event carrying the last
		// block's text (the UI shows what the model is working on).
		thinkingText := ""
		for _, b := range msg.Content {
			switch b.Type {
			case "text":
				if b.Text != "" {
					return Event{Type: EventDelta, Text: b.Text}, nil
				}
			case "tool_use":
				return Event{Type: EventTool, Tool: b.Name, Detail: string(b.Input)}, nil
			case "thinking":
				if b.Text != "" {
					thinkingText = b.Text
				}
			}
		}
		if thinkingText != "" {
			return Event{Type: EventThinking, Text: thinkingText}, nil
		}
		return Event{}, ErrIgnore
	case "result":
		if raw.IsError {
			return Event{Type: EventError, Detail: raw.Result}, nil
		}
		return Event{Type: EventDone}, nil
	default:
		return Event{}, ErrIgnore
	}
}
