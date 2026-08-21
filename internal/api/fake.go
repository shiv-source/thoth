package api

import (
	"context"
	"sync"

	agentlib "github.com/shiv-source/thoth/agent"
)

// Call records one Start invocation.
type Call struct {
	SessionID string
	Prompt    string
}

// FakeClient replays scripted events and records every call. It satisfies the
// api Client seam so tests exercise the hub without a real provider or any
// network. Usage is returned from Start when non-zero.
type FakeClient struct {
	mu     sync.Mutex
	Script []agentlib.Event
	Err    error
	Usage  agentlib.Usage
	Calls  []Call
}

func (f *FakeClient) Start(_ context.Context, sessionID, prompt string, w agentlib.EventWriter) (agentlib.Usage, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, Call{SessionID: sessionID, Prompt: prompt})
	f.mu.Unlock()
	if f.Err != nil {
		return agentlib.Usage{}, f.Err
	}
	for _, e := range f.Script {
		if err := w.Write(e); err != nil {
			return agentlib.Usage{}, err
		}
	}
	return f.Usage, nil
}
