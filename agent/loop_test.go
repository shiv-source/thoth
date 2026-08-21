package agent_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/tools"
)

// scriptedStream replays canned deltas, then reports usage.
type scriptedStream struct {
	deltas []agent.Delta
	usage  agent.Usage
	closed bool
}

func (s *scriptedStream) Next() (agent.Delta, error) {
	if len(s.deltas) == 0 {
		return agent.Delta{}, io.EOF
	}
	d := s.deltas[0]
	s.deltas = s.deltas[1:]
	return d, nil
}

func (s *scriptedStream) Usage() agent.Usage { return s.usage }
func (s *scriptedStream) Close() error       { s.closed = true; return nil }

// script is one scripted provider turn.
type script struct {
	deltas []agent.Delta
	usage  agent.Usage
	err    error // Stream returns this instead of a stream
}

// scriptedProvider replays one script per Stream call and records every
// request it is given.
type scriptedProvider struct {
	scripts []script
	calls   []agent.Request
}

func (p *scriptedProvider) Stream(_ context.Context, req agent.Request) (agent.Stream, error) {
	p.calls = append(p.calls, req)
	s := p.scripts[0]
	p.scripts = p.scripts[1:]
	if s.err != nil {
		return nil, s.err
	}
	return &scriptedStream{deltas: s.deltas, usage: s.usage}, nil
}

// fakeTool is a scripted tools.Tool.
type fakeTool struct {
	name  string
	out   string
	err   error
	args  map[string]any
	calls int
}

func (t *fakeTool) Name() string        { return t.name }
func (t *fakeTool) Description() string { return "test tool" }
func (t *fakeTool) Schema() map[string]any {
	return map[string]any{"type": "object"}
}
func (t *fakeTool) Run(_ context.Context, args map[string]any) (string, error) {
	t.calls++
	t.args = args
	if t.err != nil {
		return "", t.err
	}
	return t.out, nil
}

// eventRecorder collects written events in order.
type eventRecorder struct {
	events []agent.Event
}

func (r *eventRecorder) Write(e agent.Event) error {
	r.events = append(r.events, e)
	return nil
}

func mustAgent(t *testing.T, p agent.Provider, reg *tools.Registry, opts agent.Options) *agent.Agent {
	t.Helper()
	opts.Provider = p
	opts.Tools = reg
	a, err := agent.New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func TestLoopDrivesMultiIterationToolConversation(t *testing.T) {
	echo := &fakeTool{name: "Echo", out: "echoed"}
	reg := tools.NewRegistry()
	if err := reg.Register(echo); err != nil {
		t.Fatalf("Register: %v", err)
	}
	p := &scriptedProvider{scripts: []script{
		{
			deltas: []agent.Delta{
				agent.TextDelta("thinking"),
				agent.ToolInputDelta("toolu_1", "Echo", `{"v":"hi"}`),
				agent.StopDelta("tool_use"),
			},
			usage: agent.Usage{InputTokens: 10, OutputTokens: 2},
		},
		{
			deltas: []agent.Delta{
				agent.TextDelta("done"),
				agent.StopDelta("end_turn"),
			},
			usage: agent.Usage{InputTokens: 11, OutputTokens: 1},
		},
	}}
	a := mustAgent(t, p, reg, agent.Options{System: "rulebook", MaxOutputTokens: 512, MaxIterations: 5})
	rec := &eventRecorder{}
	usage, err := a.Start(context.Background(), "c1", "prompt", rec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	wantEvents := []agent.Event{
		{Type: agent.EventDelta, Text: "thinking"},
		{Type: agent.EventTool, Tool: "Echo", Detail: `{"v":"hi"}`},
		{Type: agent.EventDelta, Text: "done"},
		{Type: agent.EventDone},
	}
	if !reflect.DeepEqual(rec.events, wantEvents) {
		t.Fatalf("events = %+v, want %+v", rec.events, wantEvents)
	}
	if usage != (agent.Usage{InputTokens: 21, OutputTokens: 3}) {
		t.Fatalf("Start usage = %+v, want accumulated 21 in / 3 out", usage)
	}

	if len(p.calls) != 2 {
		t.Fatalf("provider called %d times, want 2", len(p.calls))
	}
	wantTurn1 := []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("prompt")}},
	}
	if !reflect.DeepEqual(p.calls[0].Messages, wantTurn1) {
		t.Fatalf("turn 1 messages = %+v, want %+v", p.calls[0].Messages, wantTurn1)
	}
	if p.calls[0].System != "rulebook" || p.calls[0].MaxTokens != 512 {
		t.Fatalf("turn 1 request = %+v", p.calls[0])
	}
	wantTurn2 := []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("prompt")}},
		{Role: agent.RoleAssistant, Content: []agent.Block{
			agent.NewTextBlock("thinking"),
			agent.NewToolUseBlock("toolu_1", "Echo", map[string]any{"v": "hi"}),
		}},
		{Role: agent.RoleTool, Content: []agent.Block{
			agent.NewToolResultBlock("toolu_1", "echoed", false),
		}},
	}
	if !reflect.DeepEqual(p.calls[1].Messages, wantTurn2) {
		t.Fatalf("turn 2 messages = %+v, want %+v", p.calls[1].Messages, wantTurn2)
	}
	if echo.calls != 1 || !reflect.DeepEqual(echo.args, map[string]any{"v": "hi"}) {
		t.Fatalf("tool ran %d times with %v", echo.calls, echo.args)
	}
	if got := a.Usage(); got != (agent.Usage{InputTokens: 21, OutputTokens: 3}) {
		t.Fatalf("Usage = %+v", got)
	}
}

func TestLoopToolErrorPassesBackAsResult(t *testing.T) {
	boom := &fakeTool{name: "Boom", err: errors.New("kaboom")}
	reg := tools.NewRegistry()
	if err := reg.Register(boom); err != nil {
		t.Fatalf("Register: %v", err)
	}
	p := &scriptedProvider{scripts: []script{
		{
			deltas: []agent.Delta{
				agent.ToolInputDelta("toolu_1", "Boom", `{}`),
				agent.StopDelta("tool_use"),
			},
		},
		{
			deltas: []agent.Delta{
				agent.TextDelta("recovered"),
				agent.StopDelta("end_turn"),
			},
		},
	}}
	a := mustAgent(t, p, reg, agent.Options{MaxIterations: 5})
	rec := &eventRecorder{}
	if _, err := a.Start(context.Background(), "c1", "prompt", rec); err != nil {
		t.Fatalf("Start: %v", err)
	}
	got := p.calls[1].Messages[2].Content[0]
	if got.Type != agent.BlockToolResult || !got.IsError {
		t.Fatalf("tool_result = %+v, want is_error result", got)
	}
	if want := "error: kaboom"; got.Text != want {
		t.Fatalf("tool_result text = %q, want %q", got.Text, want)
	}
}

func TestLoopUnknownToolFailsExplicitly(t *testing.T) {
	reg := tools.NewRegistry() // empty: no tools registered
	p := &scriptedProvider{scripts: []script{
		{
			deltas: []agent.Delta{
				agent.ToolInputDelta("toolu_1", "Ghost", `{}`),
				agent.StopDelta("tool_use"),
			},
		},
	}}
	a := mustAgent(t, p, reg, agent.Options{MaxIterations: 5})
	rec := &eventRecorder{}
	_, err := a.Start(context.Background(), "c1", "prompt", rec)
	if err == nil {
		t.Fatal("Start succeeded, want unknown-tool failure")
	}
	if rec.events[len(rec.events)-1].Type != agent.EventError {
		t.Fatalf("last event = %+v, want EventError", rec.events[len(rec.events)-1])
	}
	if len(p.calls) != 1 {
		t.Fatalf("loop did not stop after unknown tool: %d calls", len(p.calls))
	}
}

func TestLoopEnforcesIterationCap(t *testing.T) {
	reg := tools.NewRegistry()
	if err := reg.Register(&fakeTool{name: "Loop", out: "again"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	alwaysTool := &alwaysToolProvider{}
	a := mustAgent(t, alwaysTool, reg, agent.Options{MaxIterations: 3})
	rec := &eventRecorder{}
	_, err := a.Start(context.Background(), "c1", "prompt", rec)
	if err == nil {
		t.Fatal("Start succeeded, want iteration-cap failure")
	}
	if rec.events[len(rec.events)-1].Type != agent.EventError {
		t.Fatalf("last event = %+v, want EventError", rec.events[len(rec.events)-1])
	}
	if alwaysTool.calls != 3 {
		t.Fatalf("provider called %d times, want 3 (cap)", alwaysTool.calls)
	}
}

// alwaysToolProvider returns the same tool_use turn every Stream call.
type alwaysToolProvider struct {
	calls int
}

func (p *alwaysToolProvider) Stream(_ context.Context, _ agent.Request) (agent.Stream, error) {
	p.calls++
	return &scriptedStream{
		deltas: []agent.Delta{
			agent.ToolInputDelta("toolu_1", "Loop", `{}`),
			agent.StopDelta("tool_use"),
		},
	}, nil
}

func TestLoopDefaultMaxIterations(t *testing.T) {
	if _, err := agent.New(agent.Options{}); err == nil {
		t.Fatal("New with no Provider succeeded, want error")
	}
	p := &scriptedProvider{scripts: []script{{
		deltas: []agent.Delta{agent.TextDelta("hi"), agent.StopDelta("end_turn")},
	}}}
	a := mustAgent(t, p, nil, agent.Options{})
	if _, err := a.Start(context.Background(), "c1", "prompt", &eventRecorder{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestLoopCancellationReturnsCtxErrAndClosesStream(t *testing.T) {
	done := make(chan struct{})
	p := &cancelProvider{block: done, entered: make(chan struct{})}
	a := mustAgent(t, p, nil, agent.Options{})
	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan error, 1)
	go func() { _, err := a.Start(ctx, "c1", "prompt", &eventRecorder{}); started <- err }()
	select {
	case <-p.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not reach the stream")
	}
	cancel()
	close(done)

	select {
	case err := <-started:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Start returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after cancel")
	}
	if !p.streamClosed() {
		t.Fatal("stream was not closed after cancel")
	}
}

// cancelProvider returns a stream whose Next blocks until block is closed,
// then reports the context cancellation. entered signals that Stream ran.
type cancelProvider struct {
	block   <-chan struct{}
	entered chan struct{}
	mu      sync.Mutex
	closed  bool
}

func (p *cancelProvider) Stream(ctx context.Context, _ agent.Request) (agent.Stream, error) {
	close(p.entered)
	return &blockingStream{ctx: ctx, block: p.block, closed: &p.closed, mu: &p.mu}, nil
}

func (p *cancelProvider) streamClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

type blockingStream struct {
	ctx    context.Context
	block  <-chan struct{}
	closed *bool
	mu     *sync.Mutex
}

func (s *blockingStream) Next() (agent.Delta, error) {
	<-s.block
	return agent.Delta{}, s.ctx.Err()
}

func (s *blockingStream) Usage() agent.Usage { return agent.Usage{} }

func (s *blockingStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.closed = true
	return nil
}

func TestLoopHistoryCappedBeforePrompt(t *testing.T) {
	hist := []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("old0")}},
		{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewTextBlock("a0")}},
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("old1")}},
		{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewTextBlock("a1")}},
	}
	p := &scriptedProvider{scripts: []script{{
		deltas: []agent.Delta{agent.StopDelta("end_turn")},
	}}}
	a := mustAgent(t, p, nil, agent.Options{
		HistoryCap: 1,
		History: func(_ context.Context, convID string) ([]agent.Message, error) {
			if convID != "c1" {
				t.Fatalf("convID = %q", convID)
			}
			return hist, nil
		},
	})
	if _, err := a.Start(context.Background(), "c1", "prompt", &eventRecorder{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	want := []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("old1")}},
		{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewTextBlock("a1")}},
		{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("prompt")}},
	}
	if !reflect.DeepEqual(p.calls[0].Messages, want) {
		t.Fatalf("messages = %+v, want %+v", p.calls[0].Messages, want)
	}
}

func TestLoopEventOrderMatchesProviderDeltas(t *testing.T) {
	p := &scriptedProvider{scripts: []script{{
		deltas: []agent.Delta{
			agent.TextDelta("a"),
			agent.ThinkingDelta("t"),
			agent.TextDelta("b"),
			agent.ToolInputDelta("toolu_1", "Echo", `{}`),
			agent.StopDelta("tool_use"),
		},
	}}}
	a := mustAgent(t, p, nil, agent.Options{MaxIterations: 5})
	rec := &eventRecorder{}
	_, err := a.Start(context.Background(), "c1", "prompt", rec)
	if err == nil {
		t.Fatal("Start succeeded, want unknown-tool failure")
	}
	want := []agent.Event{
		{Type: agent.EventDelta, Text: "a"},
		{Type: agent.EventThinking, Text: "t"},
		{Type: agent.EventDelta, Text: "b"},
		{Type: agent.EventTool, Tool: "Echo", Detail: `{}`},
		{Type: agent.EventError},
	}
	if len(rec.events) != len(want) {
		t.Fatalf("events = %+v, want %+v", rec.events, want)
	}
	for i, e := range rec.events {
		if e.Type != want[i].Type || e.Text != want[i].Text || e.Tool != want[i].Tool {
			t.Fatalf("event %d = %+v, want %+v", i, e, want[i])
		}
	}
}
