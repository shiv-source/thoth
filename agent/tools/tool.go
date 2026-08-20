// Package tools defines the Tool extension point of the native agent: a named
// capability the model can invoke during a turn, described to the model by a
// stable name, description and input schema, and executed by Run. It ships the
// default file/search tools (read_file, write_file, list, search); hosts
// register their own tools on a Registry and the agent loop resolves tool_use
// calls through it.
package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// Tool is the extension point for agent capabilities. Name and Description
// describe the tool to the model, Schema declares the tool input as a JSON
// Schema object (type "object", properties, required), and Run executes a
// validated argument map and returns the result text.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Run(ctx context.Context, args map[string]any) (string, error)
}

// Registry is the set of tools available to one agent session. The agent loop
// resolves a model's tool_use name to its runnable Tool through it. It is not
// safe for concurrent use: register once at startup, then read.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register adds t. Registering a second tool under the same name is an error.
func (r *Registry) Register(t Tool) error {
	if t == nil {
		return errors.New("tools: cannot register a nil tool")
	}
	name := t.Name()
	if name == "" {
		return errors.New("tools: cannot register a tool with an empty name")
	}
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tools: tool %q already registered", name)
	}
	r.tools[name] = t
	return nil
}

// Get returns the tool registered under name, or an error if none exists.
func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tools: unknown tool %q", name)
	}
	return t, nil
}

// List returns every registered tool sorted by name. The stable order keeps
// the provider-facing tool list deterministic across sessions.
func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
