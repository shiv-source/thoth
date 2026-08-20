package claude

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shiv-source/thoth/agent"
)

// Event, EventType, EventWriter, and WriterFunc moved to the public agent
// package; the aliases below keep existing call sites compiling unchanged
// until the native agent cutover deletes this package.
type (
	Event       = agent.Event
	EventType   = agent.EventType
	EventWriter = agent.EventWriter
)

type WriterFunc = agent.WriterFunc

const (
	EventDelta    = agent.EventDelta
	EventTool     = agent.EventTool
	EventThinking = agent.EventThinking
	EventDone     = agent.EventDone
	EventError    = agent.EventError
)

// ErrIgnore signals a stream line that carries no event worth forwarding.
var ErrIgnore = errors.New("ignore line")

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
