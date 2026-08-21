package agent_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/tools"
)

// errWriter records events and fails on the first write of a given type.
type errWriter struct {
	events []agent.Event
	failOn agent.EventType
}

func (w *errWriter) Write(e agent.Event) error {
	w.events = append(w.events, e)
	if e.Type == w.failOn {
		return errors.New("writer boom")
	}
	return nil
}

// closeErrStream ends normally but errors on Close.
type closeErrStream struct{}

func (closeErrStream) Next() (agent.Delta, error) { return agent.Delta{}, io.EOF }
func (closeErrStream) Usage() agent.Usage         { return agent.Usage{} }
func (closeErrStream) Close() error               { return errors.New("close boom") }

// failStream returns a non-EOF error from Next.
type failStream struct{}

func (failStream) Next() (agent.Delta, error) { return agent.Delta{}, errors.New("stream boom") }
func (failStream) Usage() agent.Usage         { return agent.Usage{} }
func (failStream) Close() error               { return nil }

// streamProvider returns one fixed stream from every Stream call.
type streamProvider struct {
	stream agent.Stream
}

func (p *streamProvider) Stream(context.Context, agent.Request) (agent.Stream, error) {
	return p.stream, nil
}

func TestLoopRejectsNilWriter(t *testing.T) {
	p := &scriptedProvider{scripts: []script{{deltas: []agent.Delta{agent.StopDelta("end_turn")}}}}
	a := mustAgent(t, p, nil, agent.Options{})
	if _, err := a.Start(context.Background(), "c1", "x", nil); err == nil {
		t.Fatal("expected error for a nil writer")
	}
}

func TestLoopHistoryErrorFailsTurn(t *testing.T) {
	boom := errors.New("history boom")
	p := &scriptedProvider{scripts: []script{{deltas: []agent.Delta{agent.StopDelta("end_turn")}}}}
	a := mustAgent(t, p, nil, agent.Options{History: func(context.Context, string) ([]agent.Message, error) {
		return nil, boom
	}})
	rec := &eventRecorder{}
	_, err := a.Start(context.Background(), "c1", "x", rec)
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("Start error = %v, want history boom", err)
	}
	if len(rec.events) != 1 || rec.events[0].Type != agent.EventError {
		t.Fatalf("events = %+v, want a single EventError", rec.events)
	}
}

func TestLoopHistoryErrorWithCancelledCtx(t *testing.T) {
	p := &scriptedProvider{scripts: []script{{deltas: []agent.Delta{agent.StopDelta("end_turn")}}}}
	a := mustAgent(t, p, nil, agent.Options{History: func(context.Context, string) ([]agent.Message, error) {
		return nil, errors.New("history boom")
	}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Start(ctx, "c1", "x", &eventRecorder{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start error = %v, want context.Canceled", err)
	}
}

func TestLoopProviderStreamError(t *testing.T) {
	boom := errors.New("provider boom")
	p := &scriptedProvider{scripts: []script{{err: boom}}}
	a := mustAgent(t, p, nil, agent.Options{})
	rec := &eventRecorder{}
	_, err := a.Start(context.Background(), "c1", "x", rec)
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("Start error = %v, want provider boom", err)
	}
	if rec.events[len(rec.events)-1].Type != agent.EventError {
		t.Fatalf("last event = %+v, want EventError", rec.events[len(rec.events)-1])
	}
}

func TestLoopProviderStreamErrorWithCancelledCtx(t *testing.T) {
	p := &scriptedProvider{scripts: []script{{err: errors.New("provider boom")}}}
	a := mustAgent(t, p, nil, agent.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Start(ctx, "c1", "x", &eventRecorder{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start error = %v, want context.Canceled", err)
	}
}

func TestLoopWriterErrorOnEventDone(t *testing.T) {
	p := &scriptedProvider{scripts: []script{{deltas: []agent.Delta{agent.StopDelta("end_turn")}}}}
	a := mustAgent(t, p, nil, agent.Options{})
	if _, err := a.Start(context.Background(), "c1", "x", &errWriter{failOn: agent.EventDone}); err == nil {
		t.Fatal("Start succeeded, want EventDone write error")
	}
}

func TestLoopStreamNextError(t *testing.T) {
	a := mustAgent(t, &streamProvider{stream: failStream{}}, nil, agent.Options{})
	rec := &eventRecorder{}
	_, err := a.Start(context.Background(), "c1", "x", rec)
	if err == nil {
		t.Fatal("Start succeeded, want stream error")
	}
	if rec.events[len(rec.events)-1].Type != agent.EventError {
		t.Fatalf("last event = %+v, want EventError", rec.events[len(rec.events)-1])
	}
}

func TestLoopStreamCloseErrorDoesNotFailTurn(t *testing.T) {
	a := mustAgent(t, &streamProvider{stream: closeErrStream{}}, nil, agent.Options{})
	if _, err := a.Start(context.Background(), "c1", "x", &eventRecorder{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestLoopWriterErrorOnDelta(t *testing.T) {
	p := &scriptedProvider{scripts: []script{{deltas: []agent.Delta{agent.TextDelta("x"), agent.StopDelta("end_turn")}}}}
	a := mustAgent(t, p, nil, agent.Options{})
	if _, err := a.Start(context.Background(), "c1", "x", &errWriter{failOn: agent.EventDelta}); err == nil {
		t.Fatal("Start succeeded, want delta write error")
	}
}

func TestLoopWriterErrorOnToolEvent(t *testing.T) {
	reg := tools.NewRegistry()
	if err := reg.Register(&fakeTool{name: "Echo", out: "ok"}); err != nil {
		t.Fatal(err)
	}
	p := &scriptedProvider{scripts: []script{{
		deltas: []agent.Delta{
			agent.ToolInputDelta("toolu_1", "Echo", `{}`),
			agent.StopDelta("tool_use"),
		},
	}}}
	a := mustAgent(t, p, reg, agent.Options{MaxIterations: 3})
	if _, err := a.Start(context.Background(), "c1", "x", &errWriter{failOn: agent.EventTool}); err == nil {
		t.Fatal("Start succeeded, want tool-event write error")
	}
}

func TestLoopUnknownToolWithCancelledCtx(t *testing.T) {
	reg := tools.NewRegistry() // empty: Ghost is unregistered
	p := &scriptedProvider{scripts: []script{{
		deltas: []agent.Delta{
			agent.ToolInputDelta("toolu_1", "Ghost", `{}`),
			agent.StopDelta("tool_use"),
		},
	}}}
	a := mustAgent(t, p, reg, agent.Options{MaxIterations: 5})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Start(ctx, "c1", "x", &eventRecorder{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start error = %v, want context.Canceled", err)
	}
}

func TestCapNoPreviousUserWhenOrphaned(t *testing.T) {
	in := []agent.Message{
		{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolUseBlock("t1", "Read", nil)}},
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("u1")}},
		{Role: agent.RoleTool, Content: []agent.Block{agent.NewToolResultBlock("t1", "ok", false)}},
	}
	if got := agent.Cap(in, 1); !reflect.DeepEqual(got, in) {
		t.Fatalf("Cap = %v, want unchanged (no earlier user turn to back the cut to)", got)
	}
}
