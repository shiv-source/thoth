// Package agent is the public, provider-agnostic core of Thoth's native Go
// agent. The library is organized into focused packages so hosts and concrete
// providers import only what they need:
//
//   - events: the normalized turn events a host forwards to its UI
//   - model: the normalized message/block/delta data model and Builder
//   - provider: the Provider wire-layer seam and its request/stream types
//   - transport: the SSE reader shared by concrete providers
//
// This root package re-exports the public API, so existing callers can keep
// importing github.com/shiv-source/thoth/agent. The Agent loop itself will
// also live here.
package agent

import (
	"github.com/shiv-source/thoth/agent/events"
	"github.com/shiv-source/thoth/agent/model"
	"github.com/shiv-source/thoth/agent/provider"
)

// Event types, re-exported from agent/events.
type (
	Event       = events.Event
	EventType   = events.EventType
	EventWriter = events.EventWriter
)

// WriterFunc adapts a plain function to EventWriter.
type WriterFunc = events.WriterFunc

const (
	EventDelta    = events.EventDelta
	EventTool     = events.EventTool
	EventThinking = events.EventThinking
	EventDone     = events.EventDone
	EventError    = events.EventError
)

// Roles, re-exported from agent/model.
const (
	RoleUser      = model.RoleUser
	RoleAssistant = model.RoleAssistant
	RoleTool      = model.RoleTool
)

// Block types, re-exported from agent/model.
type (
	Block     = model.Block
	BlockType = model.BlockType
)

const (
	BlockText       = model.BlockText
	BlockToolUse    = model.BlockToolUse
	BlockToolResult = model.BlockToolResult
	BlockThinking   = model.BlockThinking
)

// NewTextBlock returns a text block with the given content.
func NewTextBlock(text string) Block { return model.NewTextBlock(text) }

// NewThinkingBlock returns a thinking block with the given reasoning text.
func NewThinkingBlock(text string) Block { return model.NewThinkingBlock(text) }

// NewToolUseBlock returns a tool_use block invoking name with input.
func NewToolUseBlock(id, name string, input map[string]any) Block {
	return model.NewToolUseBlock(id, name, input)
}

// NewToolResultBlock returns a tool_result block answering the tool_use with
// the given id.
func NewToolResultBlock(toolUseID, content string, isError bool) Block {
	return model.NewToolResultBlock(toolUseID, content, isError)
}

// ParseBlock builds a Block from one raw wire block.
func ParseBlock(raw []byte) (Block, error) { return model.ParseBlock(raw) }

// Message is one turn of the conversation history.
type Message = model.Message

// ParseMessage builds a Message from a raw {"role","content"} JSON object.
func ParseMessage(raw []byte) (Message, error) { return model.ParseMessage(raw) }

// Delta types, re-exported from agent/model.
type (
	Delta     = model.Delta
	DeltaKind = model.DeltaKind
)

const (
	DeltaText      = model.DeltaText
	DeltaThinking  = model.DeltaThinking
	DeltaToolInput = model.DeltaToolInput
	DeltaStop      = model.DeltaStop
)

// TextDelta returns a text-content delta.
func TextDelta(text string) Delta { return model.TextDelta(text) }

// ThinkingDelta returns a thinking-content delta.
func ThinkingDelta(text string) Delta { return model.ThinkingDelta(text) }

// ToolInputDelta returns a fragment of a streaming tool call.
func ToolInputDelta(useID, name, input string) Delta {
	return model.ToolInputDelta(useID, name, input)
}

// StopDelta closes a turn with the given stop reason.
func StopDelta(reason string) Delta { return model.StopDelta(reason) }

// Builder accumulates streaming deltas into the message the loop acts on.
type Builder = model.Builder

// NewBuilder returns an empty builder.
func NewBuilder() *Builder { return model.NewBuilder() }

// Provider seam, re-exported from agent/provider.
type (
	Provider = provider.Provider
	Request  = provider.Request
	Response = provider.Response
	Stream   = provider.Stream
	Tool     = provider.Tool
	Usage    = provider.Usage
)

// Accumulate consumes a stream to completion and returns the completed
// response. The stream is Closed on every path.
func Accumulate(s Stream) (Response, error) { return provider.Accumulate(s) }
