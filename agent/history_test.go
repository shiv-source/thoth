package agent_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/shiv-source/thoth/agent"
)

func userMsg(text string) agent.Message {
	return agent.Message{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock(text)}}
}

func asstMsg(text string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewTextBlock(text)}}
}

func toolUseMsg(id string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolUseBlock(id, "Read", nil)}}
}

func toolResultMsg(id string) agent.Message {
	return agent.Message{Role: agent.RoleTool, Content: []agent.Block{agent.NewToolResultBlock(id, "ok", false)}}
}

func TestCap(t *testing.T) {
	tests := []struct {
		name string
		in   []agent.Message
		n    int
		want []agent.Message
	}{
		{
			name: "empty input",
			in:   nil,
			n:    5,
			want: nil,
		},
		{
			name: "zero cap keeps everything",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1")},
			n:    0,
			want: []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1")},
		},
		{
			name: "negative cap keeps everything",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0")},
			n:    -1,
			want: []agent.Message{userMsg("u0"), asstMsg("a0")},
		},
		{
			name: "exactly n turns unchanged",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1")},
			n:    2,
			want: []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1")},
		},
		{
			name: "keeps last n turns",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1"), userMsg("u2"), asstMsg("a2")},
			n:    2,
			want: []agent.Message{userMsg("u1"), asstMsg("a1"), userMsg("u2"), asstMsg("a2")},
		},
		{
			name: "keeps a single turn",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1"), userMsg("u2"), asstMsg("a2")},
			n:    1,
			want: []agent.Message{userMsg("u2"), asstMsg("a2")},
		},
		{
			name: "drops a whole turn with its tool exchange",
			in: []agent.Message{
				userMsg("u0"), asstMsg("a0"),
				userMsg("u1"), toolUseMsg("t1"), toolResultMsg("t1"), asstMsg("a1"),
				userMsg("u2"), asstMsg("a2"),
			},
			n:    1,
			want: []agent.Message{userMsg("u2"), asstMsg("a2")},
		},
		{
			name: "keeps a paired tool exchange across the cut",
			in: []agent.Message{
				userMsg("u0"), asstMsg("a0"),
				userMsg("u1"), toolUseMsg("t1"), toolResultMsg("t1"), asstMsg("a1"),
				userMsg("u2"), asstMsg("a2"),
			},
			n: 2,
			want: []agent.Message{
				userMsg("u1"), toolUseMsg("t1"), toolResultMsg("t1"), asstMsg("a1"),
				userMsg("u2"), asstMsg("a2"),
			},
		},
		{
			name: "keeps an in-flight tool use at the end",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), toolUseMsg("t1")},
			n:    1,
			want: []agent.Message{userMsg("u1"), toolUseMsg("t1")},
		},
		{
			name: "never keeps an orphaned tool result",
			in:   []agent.Message{userMsg("u0"), toolUseMsg("t1"), userMsg("u1"), toolResultMsg("t1")},
			n:    1,
			want: []agent.Message{userMsg("u0"), toolUseMsg("t1"), userMsg("u1"), toolResultMsg("t1")},
		},
		{
			name: "no user turns unchanged",
			in:   []agent.Message{asstMsg("a0"), asstMsg("a1")},
			n:    1,
			want: []agent.Message{asstMsg("a0"), asstMsg("a1")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.Cap(tt.in, tt.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Cap(%v, %d) = %v, want %v", tt.in, tt.n, got, tt.want)
			}
		})
	}
}

func TestCapResultStartsOnUserTurn(t *testing.T) {
	tests := []struct {
		name string
		in   []agent.Message
		n    int
	}{
		{"two turns cap one", []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1")}, 1},
		{"tool exchange keeps user start", []agent.Message{userMsg("u0"), toolUseMsg("t1"), toolResultMsg("t1"), userMsg("u1"), asstMsg("a1")}, 1},
		{"three turns cap two", []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1"), userMsg("u2"), asstMsg("a2")}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.Cap(tt.in, tt.n)
			if len(got) > 0 && got[0].Role != agent.RoleUser {
				t.Fatalf("Cap(%v, %d) starts at role %q, want user", tt.in, tt.n, got[0].Role)
			}
		})
	}
}

func TestCapKeepsToolPairingsIntact(t *testing.T) {
	// Every tool_result in a capped window must have a matching tool_use
	// inside it; a capping must never strand a tool result on its own.
	history := []agent.Message{
		userMsg("u0"), toolUseMsg("t1"), toolResultMsg("t1"), asstMsg("a0"),
		userMsg("u1"), toolUseMsg("t2"), toolResultMsg("t2"), asstMsg("a1"),
		userMsg("u2"), asstMsg("a2"),
	}
	for n := 1; n <= 3; n++ {
		got := agent.Cap(history, n)
		seen := map[string]bool{}
		for _, m := range got {
			for _, b := range m.Content {
				switch b.Type {
				case agent.BlockToolUse:
					seen[b.ID] = true
				case agent.BlockToolResult:
					if !seen[b.ToolUseID] {
						t.Fatalf("Cap(history, %d) left orphaned tool_result %q in %v", n, b.ToolUseID, got)
					}
				}
			}
		}
	}
}

func TestCapDoesNotMutateInput(t *testing.T) {
	in := []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1"), userMsg("u2"), asstMsg("a2")}
	want := []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1"), userMsg("u2"), asstMsg("a2")}
	_ = agent.Cap(in, 2)
	if !reflect.DeepEqual(in, want) {
		t.Fatalf("Cap mutated its input: %v", in)
	}
}

func TestCacheMarkers(t *testing.T) {
	tests := []struct {
		name string
		in   []agent.Message
		want []int
	}{
		{
			name: "no messages tags only system",
			in:   nil,
			want: []int{agent.SystemMarker},
		},
		{
			name: "single prompt tags only system",
			in:   []agent.Message{userMsg("u0")},
			want: []int{agent.SystemMarker},
		},
		{
			name: "no history before prompt tags only system",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0")},
			want: []int{agent.SystemMarker},
		},
		{
			name: "tags the stable prefix boundary",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1")},
			want: []int{agent.SystemMarker, 1},
		},
		{
			name: "tags past a tool exchange",
			in:   []agent.Message{userMsg("u0"), toolUseMsg("t1"), toolResultMsg("t1"), userMsg("u1")},
			want: []int{agent.SystemMarker, 2},
		},
		{
			name: "never tags past the final user turn",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), toolUseMsg("t1")},
			want: []int{agent.SystemMarker, 1},
		},
		{
			name: "marker lands at the deep history boundary",
			in:   []agent.Message{userMsg("u0"), asstMsg("a0"), userMsg("u1"), asstMsg("a1"), userMsg("u2")},
			want: []int{agent.SystemMarker, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := agent.CacheMarkers(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CacheMarkers(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCacheMarkersDeterministic(t *testing.T) {
	msgs := []agent.Message{
		userMsg("u0"), toolUseMsg("t1"), toolResultMsg("t1"), asstMsg("a0"),
		userMsg("u1"), asstMsg("a1"), userMsg("u2"),
	}
	first := agent.CacheMarkers(msgs)
	for i := 0; i < 10; i++ {
		if got := agent.CacheMarkers(msgs); !reflect.DeepEqual(got, first) {
			t.Fatalf("CacheMarkers not deterministic: %v != %v", got, first)
		}
	}
}

func TestSummarizer(t *testing.T) {
	// No default implementation ships; any host type satisfying the interface
	// is the whole contract.
	var _ agent.Summarizer = (*fakeSummarizer)(nil)

	sum := &fakeSummarizer{text: "compact"}
	msgs := []agent.Message{userMsg("u0"), asstMsg("a0")}
	got, err := sum.Summarize(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if got != "compact" {
		t.Fatalf("Summarize = %q, want %q", got, "compact")
	}
	if sum.saw != len(msgs) {
		t.Fatalf("Summarize saw %d messages, want %d", sum.saw, len(msgs))
	}
}

type fakeSummarizer struct {
	text string
	saw  int
}

func (f *fakeSummarizer) Summarize(_ context.Context, msgs []agent.Message) (string, error) {
	f.saw = len(msgs)
	return f.text, nil
}
