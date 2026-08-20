package provider_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/shiv-source/thoth/agent/model"
	"github.com/shiv-source/thoth/agent/provider"
)

// stubStream is a canned stream: it replays its deltas, then reports usage.
type stubStream struct {
	deltas   []model.Delta
	nextErr  error // returned by Next instead of the next delta, once set
	closeErr error // returned by Close
	usage    provider.Usage
	closed   bool
}

func (s *stubStream) Next() (model.Delta, error) {
	if s.nextErr != nil {
		return model.Delta{}, s.nextErr
	}
	if len(s.deltas) == 0 {
		return model.Delta{}, io.EOF
	}
	d := s.deltas[0]
	s.deltas = s.deltas[1:]
	return d, nil
}

func (s *stubStream) Usage() provider.Usage { return s.usage }
func (s *stubStream) Close() error          { s.closed = true; return s.closeErr }

// stubProvider records the request it was given; it implements the Provider
// seam using only the public agent API.
type stubProvider struct {
	stream *stubStream
	req    provider.Request
}

func (p *stubProvider) Stream(ctx context.Context, req provider.Request) (provider.Stream, error) {
	p.req = req
	return p.stream, nil
}

func TestStubProviderDrivesTurn(t *testing.T) {
	p := &stubProvider{stream: &stubStream{
		deltas: []model.Delta{
			model.TextDelta("Hello"),
			model.ThinkingDelta("hmm"),
			model.ToolInputDelta("toolu_1", "Read", `{"path":"x.md"}`),
			model.StopDelta("tool_use"),
		},
		usage: provider.Usage{InputTokens: 10, OutputTokens: 5},
	}}
	req := provider.Request{
		System:    "rulebook",
		Messages:  []model.Message{{Role: model.RoleUser, Content: []model.Block{model.NewTextBlock("hi")}}},
		Tools:     []provider.Tool{{Name: "Read", Description: "reads a file"}},
		MaxTokens: 1000,
	}
	s, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if !reflect.DeepEqual(p.req, req) {
		t.Fatalf("provider got %+v, want %+v", p.req, req)
	}

	// A minimal loop: accumulate deltas until the stop delta, then close.
	b := model.NewBuilder()
	var usage provider.Usage
	for {
		d, err := s.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if d.Kind == model.DeltaStop {
			usage = s.Usage()
			break
		}
		b.Add(d)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	msg, err := b.Message()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	want := model.Message{Role: model.RoleAssistant, Content: []model.Block{
		model.NewTextBlock("Hello"),
		model.NewThinkingBlock("hmm"),
		model.NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"}),
	}}
	if !reflect.DeepEqual(msg, want) {
		t.Fatalf("got %+v, want %+v", msg, want)
	}
	if usage != (provider.Usage{InputTokens: 10, OutputTokens: 5}) {
		t.Fatalf("got usage %+v", usage)
	}
}

func TestAccumulate(t *testing.T) {
	st := &stubStream{
		deltas: []model.Delta{model.TextDelta("Hi"), model.StopDelta("end_turn")},
		usage:  provider.Usage{InputTokens: 3},
	}
	resp, err := provider.Accumulate(st)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	if !st.closed {
		t.Fatal("Accumulate did not close the stream")
	}
	want := model.Message{Role: model.RoleAssistant, Content: []model.Block{model.NewTextBlock("Hi")}}
	if !reflect.DeepEqual(resp.Message, want) {
		t.Fatalf("got %+v, want %+v", resp.Message, want)
	}
	if resp.Usage.InputTokens != 3 {
		t.Fatalf("got usage %+v", resp.Usage)
	}
}

func TestAccumulateStopsAtStopDelta(t *testing.T) {
	st := &stubStream{
		deltas: []model.Delta{
			model.TextDelta("a"),
			model.StopDelta("tool_use"),
			model.TextDelta("ignored"),
		},
		usage: provider.Usage{OutputTokens: 1},
	}
	resp, err := provider.Accumulate(st)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	if len(resp.Message.Content) != 1 || !reflect.DeepEqual(resp.Message.Content[0], model.NewTextBlock("a")) {
		t.Fatalf("got %+v", resp.Message)
	}
	rest, err := st.Next()
	if err != nil || rest != model.TextDelta("ignored") {
		t.Fatalf("stream did not stop at stop delta: %+v %v", rest, err)
	}
}

func TestAccumulateEndsAtEOF(t *testing.T) {
	st := &stubStream{
		deltas: []model.Delta{model.TextDelta("x")},
		usage:  provider.Usage{OutputTokens: 1},
	}
	resp, err := provider.Accumulate(st)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	if len(resp.Message.Content) != 1 || !reflect.DeepEqual(resp.Message.Content[0], model.NewTextBlock("x")) {
		t.Fatalf("got %+v", resp.Message)
	}
	if resp.Usage.OutputTokens != 1 {
		t.Fatalf("got usage %+v", resp.Usage)
	}
}

func TestAccumulateSurfacesStreamError(t *testing.T) {
	wantErr := errors.New("network down")
	st := &stubStream{
		deltas:  []model.Delta{model.TextDelta("a")},
		nextErr: wantErr,
	}
	if _, err := provider.Accumulate(st); !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}

func TestAccumulateSurfacesCloseError(t *testing.T) {
	wantErr := errors.New("close boom")
	st := &stubStream{
		deltas:   []model.Delta{model.StopDelta("end_turn")},
		closeErr: wantErr,
	}
	if _, err := provider.Accumulate(st); !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}

func TestAccumulateSurfacesInvalidToolInput(t *testing.T) {
	st := &stubStream{
		deltas: []model.Delta{
			model.ToolInputDelta("toolu_1", "Read", `not json`),
			model.StopDelta("tool_use"),
		},
	}
	if _, err := provider.Accumulate(st); err == nil {
		t.Fatal("expected error for malformed tool input")
	}
}
