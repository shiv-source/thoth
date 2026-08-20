package agent

import "encoding/json"

// DeltaKind enumerates the kinds of streaming deltas the loop consumes.
type DeltaKind string

const (
	DeltaText      DeltaKind = "text"
	DeltaThinking  DeltaKind = "thinking"
	DeltaToolInput DeltaKind = "tool_use_input"
	DeltaStop      DeltaKind = "stop"
)

// Delta is one streaming fragment from a provider. Text and thinking deltas
// carry Text; tool-use-input deltas carry an accumulated raw JSON fragment of
// the tool arguments plus the call identity; a stop delta closes the turn with
// its StopReason.
type Delta struct {
	Kind       DeltaKind
	Text       string
	ToolUseID  string
	ToolName   string
	Input      string // tool_use_input: accumulated raw JSON fragment
	StopReason string // stop: "end_turn" | "tool_use" | ...
}

// TextDelta returns a text-content delta.
func TextDelta(text string) Delta { return Delta{Kind: DeltaText, Text: text} }

// ThinkingDelta returns a thinking-content delta.
func ThinkingDelta(text string) Delta { return Delta{Kind: DeltaThinking, Text: text} }

// ToolInputDelta returns a fragment of a streaming tool call.
func ToolInputDelta(useID, name, input string) Delta {
	return Delta{Kind: DeltaToolInput, ToolUseID: useID, ToolName: name, Input: input}
}

// StopDelta closes a turn with the given stop reason.
func StopDelta(reason string) Delta { return Delta{Kind: DeltaStop, StopReason: reason} }

// Deltas converts a completed block into the deltas the loop would have
// streamed for it. Tool results carry no deltas and return nil.
func (b Block) Deltas() []Delta {
	switch b.Type {
	case BlockText:
		return []Delta{TextDelta(b.Text)}
	case BlockThinking:
		return []Delta{ThinkingDelta(b.Text)}
	case BlockToolUse:
		input, _ := json.Marshal(b.Input)
		return []Delta{ToolInputDelta(b.ID, b.ToolName, string(input))}
	default:
		return nil
	}
}
