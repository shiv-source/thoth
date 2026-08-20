package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/shiv-source/thoth/agent/provider"
	"github.com/shiv-source/thoth/agent/tools"
)

// stopToolUse is the normalized stop reason the loop reacts to.
const stopToolUse = "tool_use"

// Start runs the tool-use loop for one user prompt in conversation convID,
// streaming normalized events to w. The first request carries the system
// prompt, the capped history, and the user prompt; each turn the model
// requests tools, Start executes them through the registry, appends the
// tool_use and tool_result messages, and repeats until the model ends the turn
// or the iteration cap is hit. Cancelling ctx aborts the in-flight provider
// stream and returns ctx.Err(). Start emits EventDone before returning nil.
func (a *Agent) Start(ctx context.Context, convID, prompt string, w EventWriter) error {
	if w == nil {
		return errors.New("agent: EventWriter is required")
	}
	messages, err := a.conversation(ctx, convID, prompt)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return a.fail(w, err)
	}
	for iter := 1; ; iter++ {
		if iter > a.maxIterations {
			return a.fail(w, fmt.Errorf("agent: tool loop exceeded %d iterations", a.maxIterations))
		}
		s, err := a.provider.Stream(ctx, provider.Request{
			System:    a.system,
			Messages:  messages,
			Tools:     requestTools(a.tools),
			MaxTokens: a.maxOutputTokens,
		})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return a.fail(w, fmt.Errorf("agent: start turn: %w", err))
		}
		msg, stop, _, err := a.consumeTurn(s, w)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return a.fail(w, err)
		}
		if stop != stopToolUse {
			if err := w.Write(Event{Type: EventDone}); err != nil {
				return a.fail(w, err)
			}
			return nil
		}
		messages = append(messages, msg)
		for _, use := range msg.Content {
			if use.Type != BlockToolUse {
				continue
			}
			a.logger.Info("executing tool", "tool", use.ToolName, "conv_id", convID, "iteration", iter)
			result, err := a.runTool(ctx, convID, use)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return a.fail(w, err)
			}
			messages = append(messages, Message{Role: RoleTool, Content: []Block{result}})
		}
	}
}

// conversation builds the message list for the first provider turn: the
// host-injected history capped to HistoryCap turns, then the user prompt.
func (a *Agent) conversation(ctx context.Context, convID, prompt string) ([]Message, error) {
	var messages []Message
	if a.history != nil {
		h, err := a.history(ctx, convID)
		if err != nil {
			return nil, fmt.Errorf("agent: load history: %w", err)
		}
		messages = Cap(h, a.historyCap)
	}
	return append(messages, Message{
		Role:    RoleUser,
		Content: []Block{NewTextBlock(prompt)},
	}), nil
}

// consumeTurn reads every delta of s, streams the text and thinking events to
// w, and returns the accumulated assistant message, the stop reason, and the
// turn's token usage. When the turn stopped for tools, it emits one EventTool
// per completed tool_use block. The stream is closed on every path.
func (a *Agent) consumeTurn(s Stream, w EventWriter) (Message, string, Usage, error) {
	defer func() {
		if err := s.Close(); err != nil {
			a.logger.Warn("close stream", "err", err)
		}
	}()
	b := NewBuilder()
	var stop string
	for {
		d, err := s.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Message{}, "", Usage{}, fmt.Errorf("agent: stream: %w", err)
		}
		if d.Kind == DeltaStop {
			stop = d.StopReason
			break
		}
		b.Add(d)
		if err := writeDelta(w, d); err != nil {
			return Message{}, "", Usage{}, err
		}
	}
	usage := s.Usage()
	msg, err := b.Message()
	if err != nil {
		return Message{}, "", usage, err
	}
	if stop == stopToolUse {
		for _, use := range msg.Content {
			if use.Type != BlockToolUse {
				continue
			}
			if err := writeToolEvent(w, use); err != nil {
				return Message{}, "", usage, err
			}
		}
	}
	a.addUsage(usage)
	return msg, stop, usage, nil
}

// runTool resolves one tool_use block against the registry and runs it. A tool
// that errors comes back as a tool_result marked is_error; an unknown tool
// fails the turn explicitly.
func (a *Agent) runTool(ctx context.Context, convID string, use Block) (Block, error) {
	if a.tools == nil {
		return Block{}, fmt.Errorf("agent: tool %q is not registered", use.ToolName)
	}
	t, err := a.tools.Get(use.ToolName)
	if err != nil {
		return Block{}, fmt.Errorf("agent: tool %q is not registered", use.ToolName)
	}
	out, runErr := t.Run(ctx, use.Input)
	if runErr != nil {
		a.logger.Warn("tool failed", "tool", use.ToolName, "conv_id", convID, "err", runErr)
		out = fmt.Sprintf("error: %v", runErr)
	}
	return use.ToolResult(out, runErr != nil), nil
}

// fail logs err, emits an error event, and returns err.
func (a *Agent) fail(w EventWriter, err error) error {
	a.logger.Error("turn failed", "err", err)
	if werr := w.Write(Event{Type: EventError, Detail: err.Error()}); werr != nil {
		return errors.Join(err, werr)
	}
	return err
}

// writeDelta maps one streaming delta to its event. Tool-input fragments
// produce no event until the block completes.
func writeDelta(w EventWriter, d Delta) error {
	switch d.Kind {
	case DeltaText:
		return w.Write(Event{Type: EventDelta, Text: d.Text})
	case DeltaThinking:
		return w.Write(Event{Type: EventThinking, Text: d.Text})
	default:
		return nil
	}
}

// writeToolEvent emits the tool activity event for a completed tool_use block.
func writeToolEvent(w EventWriter, use Block) error {
	input := use.Input
	if input == nil {
		input = map[string]any{}
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("agent: encode tool input: %w", err)
	}
	return w.Write(Event{Type: EventTool, Tool: use.ToolName, Detail: string(raw)})
}

// requestTools renders the registry as the provider-facing tool list, sorted
// for determinism. A nil registry advertises no tools.
func requestTools(reg *tools.Registry) []provider.Tool {
	if reg == nil {
		return nil
	}
	list := reg.List()
	out := make([]provider.Tool, 0, len(list))
	for _, t := range list {
		out = append(out, provider.Tool{Name: t.Name(), Description: t.Description(), Schema: t.Schema()})
	}
	return out
}
