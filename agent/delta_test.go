package agent

import (
	"reflect"
	"testing"
)

func TestBlockDeltas(t *testing.T) {
	cases := []struct {
		block Block
		want  []Delta
	}{
		{NewTextBlock("hi"), []Delta{TextDelta("hi")}},
		{NewThinkingBlock("hmm"), []Delta{ThinkingDelta("hmm")}},
		{NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"}), []Delta{ToolInputDelta("toolu_1", "Read", `{"path":"x.md"}`)}},
		{NewToolResultBlock("toolu_1", "ok", false), nil},
	}
	for _, c := range cases {
		if got := c.block.Deltas(); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("Deltas(%+v): got %+v, want %+v", c.block, got, c.want)
		}
	}
}

func TestDeltaConstructors(t *testing.T) {
	cases := []struct {
		got  Delta
		want Delta
	}{
		{TextDelta("x"), Delta{Kind: DeltaText, Text: "x"}},
		{ThinkingDelta("y"), Delta{Kind: DeltaThinking, Text: "y"}},
		{ToolInputDelta("id", "Read", `{}`), Delta{Kind: DeltaToolInput, ToolUseID: "id", ToolName: "Read", Input: `{}`}},
		{StopDelta("end_turn"), Delta{Kind: DeltaStop, StopReason: "end_turn"}},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Fatalf("got %+v, want %+v", c.got, c.want)
		}
	}
}
