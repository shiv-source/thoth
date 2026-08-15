package claude

import (
	"context"
	"sync"
)

// Call records one Start invocation.
type Call struct {
	SessionID string
	Prompt    string
	Resume    string // fork source, set via WithResume
}

// FakeClient replays scripted events and records every call.
// Used by api package tests so no real CLI or network is involved.
type FakeClient struct {
	mu     sync.Mutex
	Script []Event
	Err    error
	Calls  []Call
}

func (f *FakeClient) Start(_ context.Context, sessionID, prompt string, w EventWriter, opts ...StartOption) error {
	var cfg startConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	f.mu.Lock()
	f.Calls = append(f.Calls, Call{SessionID: sessionID, Prompt: prompt, Resume: cfg.resume})
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
