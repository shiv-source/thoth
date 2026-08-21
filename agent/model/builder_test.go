package model

import (
	"reflect"
	"testing"
)

func TestBuilderAccumulatesText(t *testing.T) {
	b := NewBuilder()
	b.Add(TextDelta("Hello "))
	b.Add(TextDelta("wiki"))
	msg, err := b.Message()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	want := Message{Role: RoleAssistant, Content: []Block{NewTextBlock("Hello wiki")}}
	if !reflect.DeepEqual(msg, want) {
		t.Fatalf("got %+v, want %+v", msg, want)
	}
}

func TestBuilderAccumulatesThinking(t *testing.T) {
	b := NewBuilder()
	b.Add(ThinkingDelta("hmm"))
	b.Add(ThinkingDelta("..."))
	msg, err := b.Message()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	want := Message{Role: RoleAssistant, Content: []Block{NewThinkingBlock("hmm...")}}
	if !reflect.DeepEqual(msg, want) {
		t.Fatalf("got %+v, want %+v", msg, want)
	}
}

func TestBuilderAccumulatesToolInput(t *testing.T) {
	b := NewBuilder()
	b.Add(ToolInputDelta("toolu_1", "Read", `{"path":"x"`))
	b.Add(ToolInputDelta("toolu_1", "Read", `{"path":"x.md"}`))
	msg, err := b.Message()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	want := Message{Role: RoleAssistant, Content: []Block{
		NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"}),
	}}
	if !reflect.DeepEqual(msg, want) {
		t.Fatalf("got %+v, want %+v", msg, want)
	}
}

func TestBuilderSeparatesInterleavedBlocks(t *testing.T) {
	b := NewBuilder()
	b.Add(TextDelta("a"))
	b.Add(ToolInputDelta("toolu_1", "Read", `{"path":"x.md"}`))
	b.Add(TextDelta(" b"))
	msg, err := b.Message()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	want := Message{Role: RoleAssistant, Content: []Block{
		NewTextBlock("a"),
		NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"}),
		NewTextBlock(" b"),
	}}
	if !reflect.DeepEqual(msg, want) {
		t.Fatalf("got %+v, want %+v", msg, want)
	}
}

func TestBuilderInvalidToolInput(t *testing.T) {
	b := NewBuilder()
	b.Add(ToolInputDelta("toolu_1", "Read", `not json`))
	if _, err := b.Message(); err == nil {
		t.Fatal("expected error for invalid tool input")
	}
}

func TestBuilderIgnoresStop(t *testing.T) {
	b := NewBuilder()
	b.Add(StopDelta("tool_use"))
	b.Add(TextDelta("still streamed"))
	msg, err := b.Message()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if len(msg.Content) != 1 || !reflect.DeepEqual(msg.Content[0], NewTextBlock("still streamed")) {
		t.Fatalf("got %+v", msg)
	}
}
