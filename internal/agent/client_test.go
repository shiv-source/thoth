package agent

import (
	"context"
	"errors"
	"testing"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/claude"
)

// collect is an EventWriter that records every event of a turn.
type collect struct{ events []claude.Event }

func (c *collect) Write(ev claude.Event) error { c.events = append(c.events, ev); return nil }

func TestNewRejectsMissingModel(t *testing.T) {
	if _, err := New("", "key", newWiki(t, "rb"), openStore(t), nil); err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestNewSelectsProviderByModel(t *testing.T) {
	for _, model := range []string{"claude-sonnet-5", "gpt-5.6-mini"} {
		if _, err := New(model, "key", newWiki(t, "rb"), openStore(t), nil); err != nil {
			t.Fatalf("New(%q): %v", model, err)
		}
	}
	if _, err := New("gemini-3.5-pro", "key", newWiki(t, "rb"), openStore(t), nil); err == nil {
		t.Fatal("expected error for unsupported model")
	}
}

func TestClientStartRunsTurnAgainstFakeProvider(t *testing.T) {
	w := newWiki(t, "custom rulebook")
	st := openStore(t)
	convID, err := st.CreateConversation("turn")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := st.AddMessage(convID, "user", "first question"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(convID, "assistant", "first answer"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(convID, "user", "second question"); err != nil {
		t.Fatal(err)
	}

	prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{
		agentlib.TextDelta("Hello"),
		agentlib.ThinkingDelta("hmm"),
		agentlib.StopDelta("end_turn"),
	}}}
	c, err := New("claude-test", "key", w, st, openIndex(t), WithProvider(prov))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var got collect
	if err := c.Start(context.Background(), convID, "second question", &got); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Deltas flow out as claude.Event, then EventDone.
	if len(got.events) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(got.events), got.events)
	}
	if e := got.events[0]; e.Type != claude.EventDelta || e.Text != "Hello" {
		t.Fatalf("event 0 = %+v, want assistant_delta Hello", e)
	}
	if e := got.events[1]; e.Type != claude.EventThinking || e.Text != "hmm" {
		t.Fatalf("event 1 = %+v, want thinking hmm", e)
	}
	if e := got.events[2]; e.Type != claude.EventDone {
		t.Fatalf("event 2 = %+v, want turn_done", e)
	}

	// The request carries the rulebook system prompt, the history (trailing
	// user message dropped — the loop appends it) and the wiki-bounded tools.
	if prov.req.System != "custom rulebook" {
		t.Fatalf("system = %q, want custom rulebook", prov.req.System)
	}
	wantRoles := []string{"user", "assistant", "user"}
	wantTexts := []string{"first question", "first answer", "second question"}
	if len(prov.req.Messages) != 3 {
		t.Fatalf("got %d messages, want 3: %+v", len(prov.req.Messages), prov.req.Messages)
	}
	for i, m := range prov.req.Messages {
		if m.Role != wantRoles[i] {
			t.Fatalf("message %d role = %q, want %q", i, m.Role, wantRoles[i])
		}
		if len(m.Content) != 1 || m.Content[0].Text != wantTexts[i] {
			t.Fatalf("message %d content = %+v, want text %q", i, m.Content, wantTexts[i])
		}
	}
	toolNames := map[string]bool{}
	for _, tl := range prov.req.Tools {
		toolNames[tl.Name] = true
	}
	for _, want := range []string{"read_file", "write_file", "list", "search"} {
		if !toolNames[want] {
			t.Fatalf("missing tool %q in %v", want, prov.req.Tools)
		}
	}
}

func TestClientStartCapsHistory(t *testing.T) {
	w := newWiki(t, "rb")
	st := openStore(t)
	convID, err := st.CreateConversation("cap")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		if err := st.AddMessage(convID, "user", "q"); err != nil {
			t.Fatal(err)
		}
		if err := st.AddMessage(convID, "assistant", "a"); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.AddMessage(convID, "user", "current"); err != nil {
		t.Fatal(err)
	}

	prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}
	c, err := New("claude-test", "key", w, st, nil, WithProvider(prov), WithHistoryCap(2))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Start(context.Background(), convID, "current", &collect{}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Last two turns of history plus the appended prompt.
	if len(prov.req.Messages) != 5 {
		t.Fatalf("got %d messages, want 5 (2 capped turns + prompt): %+v", len(prov.req.Messages), prov.req.Messages)
	}
	wantRoles := []string{"user", "assistant", "user", "assistant", "user"}
	for i, m := range prov.req.Messages {
		if m.Role != wantRoles[i] {
			t.Fatalf("message %d role = %q, want %q", i, m.Role, wantRoles[i])
		}
	}
}

func TestClientStartReadsSystemPerTurn(t *testing.T) {
	w := newWiki(t, "rulebook v1")
	st := openStore(t)
	convID, err := st.CreateConversation("fresh")
	if err != nil {
		t.Fatal(err)
	}
	prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}
	c, err := New("claude-test", "key", w, st, nil, WithProvider(prov))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Start(context.Background(), convID, "hi", &collect{}); err != nil {
		t.Fatal(err)
	}
	if prov.req.System != "rulebook v1" {
		t.Fatalf("system = %q, want rulebook v1", prov.req.System)
	}

	if err := writeRulebook(t, w, "rulebook v2"); err != nil {
		t.Fatal(err)
	}
	if err := c.Start(context.Background(), convID, "hi again", &collect{}); err != nil {
		t.Fatal(err)
	}
	if prov.req.System != "rulebook v2" {
		t.Fatalf("system after edit = %q, want rulebook v2", prov.req.System)
	}
}

func TestClientStartSurfacesProviderError(t *testing.T) {
	w := newWiki(t, "rb")
	st := openStore(t)
	convID, err := st.CreateConversation("err")
	if err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("provider down")
	prov := &fakeProvider{err: wantErr}
	c, err := New("claude-test", "key", w, st, nil, WithProvider(prov))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var got collect
	err = c.Start(context.Background(), convID, "hi", &got)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Start error = %v, want %v", err, wantErr)
	}
	if len(got.events) != 1 || got.events[0].Type != claude.EventError {
		t.Fatalf("events = %+v, want a single error event", got.events)
	}
}

func TestClientStartRequiresWriter(t *testing.T) {
	w := newWiki(t, "rb")
	st := openStore(t)
	convID, err := st.CreateConversation("writer")
	if err != nil {
		t.Fatal(err)
	}
	c, err := New("claude-test", "key", w, st, nil,
		WithProvider(&fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}),
		WithLogger(discardLog()),
		WithFolders([]string{"inbox"}),
		WithMaxIterations(5),
		WithMaxOutputTokens(1000),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Start(context.Background(), convID, "hi", nil); err == nil {
		t.Fatal("expected error for nil writer")
	}
}
