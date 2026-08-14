package claude

import (
	"context"
	"sync"
)

// Call records one Start invocation.
type Call struct {
	SessionID string
	Prompt    string
}

// FakeClient replays scripted events and records every call.
// Used by api package tests so no real CLI or network is involved.
type FakeClient struct {
	mu     sync.Mutex
	Script []Event
	Err    error
	Calls  []Call
}

func (f *FakeClient) Start(_ context.Context, sessionID, prompt string, w EventWriter) error {
	f.mu.Lock()
	f.Calls = append(f.Calls, Call{SessionID: sessionID, Prompt: prompt})
	f.mu.Unlock()
	if f.Err != nil {
		return f.Err
	}
	for _, e := range f.Script {
		if err := w.Write(e); err != nil {
			return err
		}
	}
	return nil
}
