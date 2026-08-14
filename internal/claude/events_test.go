package claude

import (
	"errors"
	"testing"
)

func TestParseLineAssistantText(t *testing.T) {
	ev, err := ParseLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello wiki"}]}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if ev.Type != EventDelta || ev.Text != "Hello wiki" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineToolUse(t *testing.T) {
	ev, err := ParseLine([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"meetings/x.md"}}]}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if ev.Type != EventTool || ev.Tool != "Read" || ev.Detail == "" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineResult(t *testing.T) {
	ev, err := ParseLine([]byte(`{"type":"result","subtype":"success","is_error":false,"result":"done"}`))
	if err != nil || ev.Type != EventDone {
		t.Fatalf("unexpected: %+v %v", ev, err)
	}
	ev, err = ParseLine([]byte(`{"type":"result","subtype":"error_max_turns","is_error":true,"result":"too many turns"}`))
	if err != nil || ev.Type != EventError || ev.Detail == "" {
		t.Fatalf("unexpected error event: %+v %v", ev, err)
	}
}

func TestParseLineIgnoresUnknown(t *testing.T) {
	_, err := ParseLine([]byte(`{"type":"system","subtype":"init"}`))
	if !errors.Is(err, ErrIgnore) {
		t.Fatalf("expected ErrIgnore, got %v", err)
	}
}

func TestParseLineRejectsGarbage(t *testing.T) {
	if _, err := ParseLine([]byte(`not json`)); err == nil || errors.Is(err, ErrIgnore) {
		t.Fatalf("expected a parse error, got %v", err)
	}
}

func TestParseLineAssistantWithoutMessage(t *testing.T) {
	// An assistant line with no message payload carries no event.
	_, err := ParseLine([]byte(`{"type":"assistant"}`))
	if !errors.Is(err, ErrIgnore) {
		t.Fatalf("expected ErrIgnore, got %v", err)
	}
}

func TestParseLineAssistantWithEmptyText(t *testing.T) {
	// Empty text blocks and unknown block types are ignored.
	_, err := ParseLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":""}]}}`))
	if !errors.Is(err, ErrIgnore) {
		t.Fatalf("expected ErrIgnore for empty text, got %v", err)
	}
	_, err = ParseLine([]byte(`{"type":"assistant","message":{"content":[{"type":"thinking","text":"hmm"}]}}`))
	if !errors.Is(err, ErrIgnore) {
		t.Fatalf("expected ErrIgnore for unknown block, got %v", err)
	}
	_, err = ParseLine([]byte(`{"type":"assistant","message":{"content":[]}}`))
	if !errors.Is(err, ErrIgnore) {
		t.Fatalf("expected ErrIgnore for empty content, got %v", err)
	}
}

func TestWriterFuncAdapter(t *testing.T) {
	var got Event
	err := WriterFunc(func(e Event) error { got = e; return nil }).Write(Event{Type: EventDelta, Text: "x"})
	if err != nil || got.Type != EventDelta || got.Text != "x" {
		t.Fatalf("WriterFunc adapter broken: %v %+v", err, got)
	}
}
