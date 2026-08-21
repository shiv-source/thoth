// Package provider is the wire-layer seam every model provider implements.
// The agent loop knows nothing about a vendor's wire format: it sends one
// normalized Request and consumes normalized deltas from the returned Stream.
// Concrete providers (agent/provider/anthropic, agent/provider/openai) live
// as subpackages.
package provider

import (
	"context"
	"errors"
	"io"

	"github.com/shiv-source/thoth/agent/model"
)

// Tool describes one tool the model may call during a turn. Providers render
// these into their own wire format (name, description, input schema); the loop
// resolves the name to a runnable implementation.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any // JSON Schema for the tool input
}

// Request is the normalized, provider-agnostic description of one turn: the
// conversation history, the tools available to the model, the system prompt,
// and the token budget. Providers map it to their wire request.
type Request struct {
	System    string
	Messages  []model.Message
	Tools     []Tool
	MaxTokens int
}

// Usage holds the token counters a provider reports for a turn. Counters a
// provider does not report are zero. The json tags give the host wire layer a
// stable frame shape (usage on turn_done).
type Usage struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens"`  // prompt-cache hits; 0 when the provider has no cache
	CacheWriteTokens int `json:"cache_write_tokens"` // prompt-cache writes
}

// Response is a completed turn: the accumulated assistant message plus the
// final token usage. Accumulate produces it from a Stream.
type Response struct {
	Message model.Message
	Usage   Usage
}

// Provider is the wire-layer seam every model provider implements.
type Provider interface {
	// Stream starts a turn and returns a stream of normalized deltas. The
	// caller must Close the stream when done; cancelling ctx aborts the
	// underlying request.
	Stream(ctx context.Context, req Request) (Stream, error)
}

// Stream is one streaming turn. Next returns the deltas in order and then
// io.EOF when the turn ends; transport errors come back as non-nil errors.
// Usage is valid once Next has returned io.EOF or a stop delta was seen.
type Stream interface {
	Next() (model.Delta, error)
	Usage() Usage
	Close() error
}

// Accumulate consumes a stream to completion and returns the completed
// response: the accumulated assistant message plus the final usage. It stops
// at the first stop delta, leaving the loop to decide whether to run tools.
// The stream is Closed on every path.
func Accumulate(s Stream) (resp Response, err error) {
	defer func() {
		if cerr := s.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()
	b := model.NewBuilder()
	for {
		d, err := s.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Response{}, err
		}
		if d.Kind == model.DeltaStop {
			break
		}
		b.Add(d)
	}
	msg, err := b.Message()
	if err != nil {
		return Response{}, err
	}
	return Response{Message: msg, Usage: s.Usage()}, nil
}
