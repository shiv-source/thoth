package agent

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseBlockText(t *testing.T) {
	b, err := ParseBlock([]byte(`{"type":"text","text":"Hello wiki"}`))
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	want := NewTextBlock("Hello wiki")
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("got %+v, want %+v", b, want)
	}
}

func TestParseBlockThinking(t *testing.T) {
	// Anthropic wire shape uses "thinking"; the CLI stream uses "text".
	for _, raw := range []string{
		`{"type":"thinking","thinking":"hmm"}`,
		`{"type":"thinking","text":"hmm"}`,
	} {
		b, err := ParseBlock([]byte(raw))
		if err != nil || !reflect.DeepEqual(b, NewThinkingBlock("hmm")) {
			t.Fatalf("thinking block %s: got %+v %v", raw, b, err)
		}
	}
}

func TestParseBlockToolUse(t *testing.T) {
	raw := `{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"meetings/x.md"}}`
	b, err := ParseBlock([]byte(raw))
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	want := NewToolUseBlock("toolu_1", "Read", map[string]any{"file_path": "meetings/x.md"})
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("got %+v, want %+v", b, want)
	}
}

func TestParseBlockToolResult(t *testing.T) {
	b, err := ParseBlock([]byte(`{"type":"tool_result","tool_use_id":"toolu_1","content":"done","is_error":false}`))
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	want := NewToolResultBlock("toolu_1", "done", false)
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("got %+v, want %+v", b, want)
	}
}

func TestParseBlockRejectsUnknown(t *testing.T) {
	if _, err := ParseBlock([]byte(`{"type":"nope"}`)); err == nil {
		t.Fatal("expected error for unknown block type")
	}
	if _, err := ParseBlock([]byte(`not json`)); err == nil {
		t.Fatal("expected error for garbage")
	}
}

func TestBlockRoundTrip(t *testing.T) {
	blocks := []Block{
		NewTextBlock("hello"),
		NewThinkingBlock("reasoning"),
		NewToolUseBlock("toolu_1", "Read", map[string]any{"file_path": "a/b.md"}),
		// Numbers parse losslessly as json.Number, so the wire form is the
		// ground truth for round-trip equality (a raw int64 becomes a
		// json.Number after the trip).
		NewToolUseBlock("toolu_2", "Search", map[string]any{"query": "note", "n": 5, "big": int64(9007199254740993)}),
		NewToolResultBlock("toolu_1", "contents", false),
		NewToolResultBlock("toolu_2", "boom", true),
	}
	for _, b := range blocks {
		raw, err := b.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON(%+v): %v", b, err)
		}
		got, err := ParseBlock(raw)
		if err != nil {
			t.Fatalf("ParseBlock(%s): %v", raw, err)
		}
		again, err := got.MarshalJSON()
		if err != nil {
			t.Fatalf("re-MarshalJSON(%+v): %v", got, err)
		}
		if string(again) != string(raw) {
			t.Fatalf("wire round trip changed %s -> %s", raw, again)
		}
	}
}

func TestToolResultRendersForToolUse(t *testing.T) {
	use := NewToolUseBlock("toolu_1", "Write", map[string]any{"path": "x.md"})
	res := use.ToolResult("saved", false)
	want := NewToolResultBlock("toolu_1", "saved", false)
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("got %+v, want %+v", res, want)
	}
	raw, err := res.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(raw), `"tool_result"`) {
		t.Fatalf("expected tool_result wire shape, got %s", raw)
	}
}

func TestMessageRoundTrip(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		Content: []Block{
			NewTextBlock("answer"),
			NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"}),
		},
	}
	raw, err := msg.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	got, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage(%s): %v", raw, err)
	}
	if !reflect.DeepEqual(got, msg) {
		t.Fatalf("round trip: got %+v, want %+v", got, msg)
	}
}

func TestParseMessage(t *testing.T) {
	raw := []byte(`{"role":"assistant","content":[{"type":"text","text":"answer"},{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}`)
	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Role != RoleAssistant || len(msg.Content) != 2 {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.Content[1].Type != BlockToolResult || msg.Content[1].ToolUseID != "toolu_1" {
		t.Fatalf("unexpected tool_result block: %+v", msg.Content[1])
	}
}
