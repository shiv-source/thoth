package agent_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/shiv-source/thoth/agent"
)

// stream replays canned deltas, then reports usage.
type stream struct {
	deltas []agent.Delta
	usage  agent.Usage
}

func (s *stream) Next() (agent.Delta, error) {
	if len(s.deltas) == 0 {
		return agent.Delta{}, io.EOF
	}
	d := s.deltas[0]
	s.deltas = s.deltas[1:]
	return d, nil
}

func (s *stream) Usage() agent.Usage { return s.usage }
func (s *stream) Close() error       { return nil }

// fakeProvider implements the Provider seam using only the root agent API.
type fakeProvider struct{}

func (fakeProvider) Stream(ctx context.Context, req agent.Request) (agent.Stream, error) {
	if req.System != "rulebook" {
		return nil, errors.New("unexpected system prompt")
	}
	return &stream{
		deltas: []agent.Delta{
			agent.TextDelta("Hello"),
			agent.ThinkingDelta("hmm"),
			agent.StopDelta("end_turn"),
		},
		usage: agent.Usage{InputTokens: 7, OutputTokens: 3},
	}, nil
}

func TestRootAPIReexports(t *testing.T) {
	// Event writer and type constants.
	var got agent.Event
	err := agent.WriterFunc(func(e agent.Event) error { got = e; return nil }).Write(agent.Event{Type: agent.EventDelta, Text: "x"})
	if err != nil || got.Type != agent.EventDelta || got.Text != "x" {
		t.Fatalf("event writer broken: %+v %v", got, err)
	}
	wantTypes := map[agent.EventType]string{
		agent.EventDelta:    "assistant_delta",
		agent.EventTool:     "tool_activity",
		agent.EventThinking: "thinking",
		agent.EventDone:     "turn_done",
		agent.EventError:    "error",
	}
	for typ, val := range wantTypes {
		if string(typ) != val {
			t.Fatalf("EventType %q = %q", typ, typ)
		}
	}

	// Block constructors + ParseBlock round-trip.
	want := agent.NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"})
	raw, err := want.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	block, err := agent.ParseBlock(raw)
	if err != nil || !reflect.DeepEqual(block, want) {
		t.Fatalf("block round trip: %+v %v", block, err)
	}
	if agent.NewThinkingBlock("t").Text != "t" {
		t.Fatal("NewThinkingBlock broken")
	}
	res := agent.NewToolResultBlock("toolu_1", "ok", false)
	if res.Type != agent.BlockToolResult || res.ToolUseID != "toolu_1" || res.IsError {
		t.Fatalf("NewToolResultBlock broken: %+v", res)
	}
	for typ, val := range map[agent.BlockType]string{
		agent.BlockText:       "text",
		agent.BlockToolUse:    "tool_use",
		agent.BlockToolResult: "tool_result",
		agent.BlockThinking:   "thinking",
	} {
		if string(typ) != val {
			t.Fatalf("BlockType %q = %q", typ, typ)
		}
	}

	// Message + ParseMessage + roles.
	m, err := agent.ParseMessage([]byte(`{"role":"assistant","content":[{"type":"text","text":"hi"}]}`))
	if err != nil || m.Role != agent.RoleAssistant || len(m.Content) != 1 {
		t.Fatalf("ParseMessage: %+v %v", m, err)
	}
	for role, val := range map[string]string{
		agent.RoleUser:      "user",
		agent.RoleAssistant: "assistant",
		agent.RoleTool:      "tool",
	} {
		if role != val {
			t.Fatalf("role %q != %q", role, val)
		}
	}

	// Delta constructors + builder.
	b := agent.NewBuilder()
	b.Add(agent.TextDelta("a"))
	b.Add(agent.ThinkingDelta("t"))
	b.Add(agent.ToolInputDelta("toolu_1", "Read", `{"path":"x.md"}`))
	b.Add(agent.StopDelta("tool_use"))
	msg, err := b.Message()
	if err != nil || len(msg.Content) != 3 {
		t.Fatalf("Builder.Message: %+v %v", msg, err)
	}
	if !reflect.DeepEqual(msg.Content[0], agent.NewTextBlock("a")) || !reflect.DeepEqual(msg.Content[1], agent.NewThinkingBlock("t")) {
		t.Fatalf("builder blocks: %+v", msg.Content)
	}
	for kind, val := range map[agent.DeltaKind]string{
		agent.DeltaText:      "text",
		agent.DeltaThinking:  "thinking",
		agent.DeltaToolInput: "tool_use_input",
		agent.DeltaStop:      "stop",
	} {
		if string(kind) != val {
			t.Fatalf("DeltaKind %q = %q", kind, kind)
		}
	}

	// Provider seam + Accumulate through the facade.
	p := fakeProvider{}
	s, err := p.Stream(context.Background(), agent.Request{System: "rulebook"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	resp, err := agent.Accumulate(s)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	if resp.Usage != (agent.Usage{InputTokens: 7, OutputTokens: 3}) {
		t.Fatalf("usage: %+v", resp.Usage)
	}
	if len(resp.Message.Content) != 2 || !reflect.DeepEqual(resp.Message.Content[0], agent.NewTextBlock("Hello")) || !reflect.DeepEqual(resp.Message.Content[1], agent.NewThinkingBlock("hmm")) {
		t.Fatalf("response message: %+v", resp.Message)
	}
}
