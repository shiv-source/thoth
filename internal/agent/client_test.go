package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/provider/anthropic"
	"github.com/shiv-source/thoth/agent/provider/openai"
)

// collect is an EventWriter that records every event of a turn.
type collect struct{ events []agentlib.Event }

func (c *collect) Write(ev agentlib.Event) error { c.events = append(c.events, ev); return nil }

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

func TestNewRoutesByProviderName(t *testing.T) {
	w, st := newWiki(t, "rb"), openStore(t)
	tests := []struct {
		name       string
		provider   string
		wantOpenai bool
	}{
		{"anthropic", "Anthropic", false},
		{"openai", "OpenAI", true},
		{"deepseek", "DeepSeek", true},
		{"qwen", "Qwen", true},
		{"zai", "Z.AI", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New("model-x", "key", w, st, nil, WithProviderConfig(tt.provider, ""))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			_, isAnthropic := c.provider.(*anthropic.Client)
			_, isOpenAI := c.provider.(*openai.Client)
			if tt.wantOpenai && !isOpenAI {
				t.Fatalf("provider %q = %T, want *openai.Client", tt.provider, c.provider)
			}
			if !tt.wantOpenai && !isAnthropic {
				t.Fatalf("provider %q = %T, want *anthropic.Client", tt.provider, c.provider)
			}
		})
	}
}

func TestNewRoutesBaseURL(t *testing.T) {
	// The custom base URL must be honored by the constructed provider: a
	// Stream request hits the httptest server, not the vendor default.
	var mu sync.Mutex
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits = append(hits, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	for _, tt := range []struct{ provider, path string }{
		{"Anthropic", "/v1/messages"},
		{"DeepSeek", "/v1/chat/completions"},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			prov, err := providerFor("model-x", "key", tt.provider, srv.URL)
			if err != nil {
				t.Fatalf("providerFor: %v", err)
			}
			stream, err := prov.Stream(context.Background(), agentlib.Request{})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			_ = stream.Close()
			mu.Lock()
			defer mu.Unlock()
			if !slices.Contains(hits, tt.path) {
				t.Fatalf("provider %q did not hit %s (hits=%v)", tt.provider, tt.path, hits)
			}
		})
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

	// Deltas flow out as agent events, then EventDone.
	if len(got.events) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(got.events), got.events)
	}
	if e := got.events[0]; e.Type != agentlib.EventDelta || e.Text != "Hello" {
		t.Fatalf("event 0 = %+v, want assistant_delta Hello", e)
	}
	if e := got.events[1]; e.Type != agentlib.EventThinking || e.Text != "hmm" {
		t.Fatalf("event 1 = %+v, want thinking hmm", e)
	}
	if e := got.events[2]; e.Type != agentlib.EventDone {
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
	if len(got.events) != 1 || got.events[0].Type != agentlib.EventError {
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
