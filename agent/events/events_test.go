package events

import (
	"reflect"
	"testing"
)

func TestEventTypeValues(t *testing.T) {
	want := map[EventType]string{
		EventDelta:    "assistant_delta",
		EventTool:     "tool_activity",
		EventThinking: "thinking",
		EventDone:     "turn_done",
		EventError:    "error",
	}
	for typ, wantVal := range want {
		if string(typ) != wantVal {
			t.Fatalf("EventType %q = %q, want %q", typ, typ, wantVal)
		}
	}
}

func TestEventShapeUnchanged(t *testing.T) {
	typ := reflect.TypeOf(Event{})
	want := []struct{ name, kind string }{
		{"Type", "string"},
		{"Text", "string"},
		{"Tool", "string"},
		{"Detail", "string"},
	}
	if typ.NumField() != len(want) {
		t.Fatalf("Event has %d fields, want %d (names/fields unchanged)", typ.NumField(), len(want))
	}
	for i, f := range want {
		sf := typ.Field(i)
		if sf.Name != f.name || sf.Type.Kind().String() != f.kind {
			t.Fatalf("Event field %d = %s (%s), want %s (%s)", i, sf.Name, sf.Type.Kind(), f.name, f.kind)
		}
	}
}

func TestWriterFuncAdapter(t *testing.T) {
	var got Event
	err := WriterFunc(func(e Event) error { got = e; return nil }).Write(Event{Type: EventDelta, Text: "x"})
	if err != nil || got.Type != EventDelta || got.Text != "x" {
		t.Fatalf("WriterFunc adapter broken: %v %+v", err, got)
	}
}
