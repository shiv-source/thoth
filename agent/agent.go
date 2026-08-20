// Package agent is the public, provider-agnostic core of Thoth's native Go
// agent. The library is organized into focused packages so hosts and concrete
// providers import only what they need:
//
//   - events: the normalized turn events a host forwards to its UI
//   - model: the normalized message/block/delta data model and Builder
//   - provider: the Provider wire-layer seam and its request/stream types
//   - tools: the Tool extension point, the Registry, and the default
//     root-bounded file/search tools
//   - transport: the SSE reader shared by concrete providers
//
// This root package re-exports the public API and hosts the Agent loop, so
// existing callers can keep importing github.com/shiv-source/thoth/agent.
package agent

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/shiv-source/thoth/agent/events"
	"github.com/shiv-source/thoth/agent/model"
	"github.com/shiv-source/thoth/agent/provider"
	"github.com/shiv-source/thoth/agent/tools"
)

// Options configures an Agent. New fills the defaults for fields left zero.
type Options struct {
	// Provider drives every turn of the tool-use loop. Required.
	Provider Provider
	// Model is the model id the conversation runs on. Required. Wire
	// propagation lands with the provider request wiring (T3/T4); hosts pick
	// the model at the provider client they construct.
	Model string
	// System is the system prompt prepended to every provider request.
	System string
	// History returns the prior messages of a conversation. The loop caps it
	// to HistoryCap turns, appends the user prompt, and sends it with every
	// request. Optional.
	History func(ctx context.Context, convID string) ([]Message, error)
	// HistoryCap keeps the last n user-initiated turns of history; 0 disables
	// the cap. The cap never splits a turn or strands a tool_result.
	HistoryCap int
	// Tools is the registry the loop resolves tool_use calls through. When
	// nil, no tools are advertised to the model.
	Tools *tools.Registry
	// MaxIterations bounds the number of provider turns; 0 selects the
	// default of 25. A loop whose model keeps requesting tools past the cap
	// terminates with an explicit error event instead of hanging.
	MaxIterations int
	// MaxOutputTokens bounds each provider turn's output; providers apply
	// their own default when left zero.
	MaxOutputTokens int
	// Logger receives loop diagnostics; slog.Default() when nil.
	Logger *slog.Logger
}

// defaultMaxIterations is the MaxIterations used when Options leaves it zero.
const defaultMaxIterations = 25

// Agent drives a provider tool-use loop: it streams one or more provider
// turns, executes each tool_use the model makes through the Tools registry,
// feeds the results back, and repeats until the model ends the turn. It is
// not safe for concurrent Start calls.
type Agent struct {
	provider        Provider
	model           string
	system          string
	history         func(ctx context.Context, convID string) ([]Message, error)
	historyCap      int
	tools           *tools.Registry
	maxIterations   int
	maxOutputTokens int
	logger          *slog.Logger

	mu    sync.Mutex
	usage Usage
}

// New returns an Agent configured by opts. It fails when opts.Provider or
// opts.Model is missing.
func New(opts Options) (*Agent, error) {
	if opts.Provider == nil {
		return nil, errors.New("agent: Provider is required")
	}
	if opts.Model == "" {
		return nil, errors.New("agent: Model is required")
	}
	a := &Agent{
		provider:        opts.Provider,
		model:           opts.Model,
		system:          opts.System,
		history:         opts.History,
		historyCap:      opts.HistoryCap,
		tools:           opts.Tools,
		maxIterations:   opts.MaxIterations,
		maxOutputTokens: opts.MaxOutputTokens,
		logger:          opts.Logger,
	}
	if a.maxIterations <= 0 {
		a.maxIterations = defaultMaxIterations
	}
	if a.logger == nil {
		a.logger = slog.Default()
	}
	return a, nil
}

// Usage returns the token usage accumulated across the loop's turns since the
// agent was created.
func (a *Agent) Usage() Usage {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.usage
}

func (a *Agent) addUsage(u Usage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.usage.InputTokens += u.InputTokens
	a.usage.OutputTokens += u.OutputTokens
	a.usage.CacheReadTokens += u.CacheReadTokens
	a.usage.CacheWriteTokens += u.CacheWriteTokens
}

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
