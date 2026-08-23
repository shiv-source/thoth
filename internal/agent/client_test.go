package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/provider/anthropic"
	"github.com/shiv-source/thoth/agent/provider/openai"
	"github.com/shiv-source/thoth/internal/index"
)

// collect is an EventWriter that records every event of a turn.
type collect struct{ events []agentlib.Event }

func (c *collect) Write(ev agentlib.Event) error { c.events = append(c.events, ev); return nil }

func TestNewRejectsMissingModel(t *testing.T) {
	if _, err := New("", "key", newWiki(t, "rb"), openStore(t), nil); err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestNewRejectsNilWiki(t *testing.T) {
	if _, err := New("claude-test", "key", nil, openStore(t), nil); err == nil {
		t.Fatal("expected error for nil wiki")
	}
}

func TestNewRejectsNilStore(t *testing.T) {
	if _, err := New("claude-test", "key", newWiki(t, "rb"), nil, nil); err == nil {
		t.Fatal("expected error for nil store")
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
			prov, err := providerFor("model-x", "key", tt.provider, srv.URL, nil)
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

	prov := &fakeProvider{stream: &fakeStream{
		deltas: []agentlib.Delta{
			agentlib.TextDelta("Hello"),
			agentlib.ThinkingDelta("hmm"),
			agentlib.StopDelta("end_turn"),
		},
		usage: agentlib.Usage{InputTokens: 12, OutputTokens: 3, CacheReadTokens: 5},
	}}
	c, err := New("claude-test", "key", w, st, openIndex(t), WithProvider(prov))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var got collect
	usage, err := c.Start(context.Background(), convID, "second question", &got)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if usage != (agentlib.Usage{InputTokens: 12, OutputTokens: 3, CacheReadTokens: 5}) {
		t.Fatalf("Start usage = %+v, want the provider's usage", usage)
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
	if _, err := c.Start(context.Background(), convID, "current", &collect{}); err != nil {
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
	if _, err := c.Start(context.Background(), convID, "hi", &collect{}); err != nil {
		t.Fatal(err)
	}
	if prov.req.System != "rulebook v1" {
		t.Fatalf("system = %q, want rulebook v1", prov.req.System)
	}

	if err := writeRulebook(t, w, "rulebook v2"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Start(context.Background(), convID, "hi again", &collect{}); err != nil {
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
	_, err = c.Start(context.Background(), convID, "hi", &got)
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
	if _, err := c.Start(context.Background(), convID, "hi", nil); err == nil {
		t.Fatal("expected error for nil writer")
	}
}

func TestClientStartTurnTimeout(t *testing.T) {
	w := newWiki(t, "rb")
	st := openStore(t)
	convID, err := st.CreateConversation("timeout")
	if err != nil {
		t.Fatal(err)
	}
	// A provider that stalls on the wire with no error: without a turn
	// timeout this would hang the turn and its socket indefinitely.
	c, err := New("claude-test", "key", w, st, nil,
		WithProvider(blockingProvider{}),
		WithTurnTimeout(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	start := time.Now()
	_, err = c.Start(context.Background(), convID, "hi", &collect{})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("turn took %v, want it to return promptly after the timeout", elapsed)
	}
}

func TestClientStartContextInjection(t *testing.T) {
	w := newWiki(t, "rb")
	st := openStore(t)
	convID, err := st.CreateConversation("ctx")
	if err != nil {
		t.Fatal(err)
	}
	ix := openIndex(t)
	if err := ix.Upsert(index.Note{
		Path: "knowledge/context.md", Title: "Context injection", Kind: "knowledge",
		Body:      "pre-searched notes let the model answer without tool round-trips",
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		prompt       string
		enabled      bool
		wantInjected bool
	}{
		{"matches inject into first turn", "how do pre-searched notes work?", true, true},
		{"no matches keeps the prompt as-is", "completely unrelated topic", true, false},
		{"setting off never injects", "how do pre-searched notes work?", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}
			c, err := New("claude-test", "key", w, st, ix, WithProvider(prov), WithContextInjection(tt.enabled))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if _, err := c.Start(context.Background(), convID, tt.prompt, &collect{}); err != nil {
				t.Fatalf("Start: %v", err)
			}
			if len(prov.req.Messages) == 0 {
				t.Fatal("provider recorded no request")
			}
			last := prov.req.Messages[len(prov.req.Messages)-1]
			text := last.Content[0].Text
			if tt.wantInjected {
				if !strings.HasPrefix(text, contextInjectionHeader) {
					t.Fatalf("injected block must precede the question: %q", text)
				}
				if !strings.Contains(text, "knowledge/context.md") {
					t.Fatalf("first-turn prompt missing the note path: %q", text)
				}
			} else if text != tt.prompt {
				t.Fatalf("prompt changed without injection: %q", text)
			}
		})
	}
}

func TestClientStartContextInjectionFailsOpen(t *testing.T) {
	w := newWiki(t, "rb")
	st := openStore(t)
	convID, err := st.CreateConversation("ctx")
	if err != nil {
		t.Fatal(err)
	}
	ix := openIndex(t)
	if err := ix.Upsert(index.Note{
		Path: "knowledge/context.md", Title: "Context injection", Kind: "knowledge",
		Body:      "pre-searched notes let the model answer without tool round-trips",
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}

	prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}
	c, err := New("claude-test", "key", w, st, ix, WithProvider(prov), WithContextInjection(true))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	prompt := "pre-searched notes"
	if _, err := c.Start(context.Background(), convID, prompt, &collect{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	last := prov.req.Messages[len(prov.req.Messages)-1]
	if last.Content[0].Text != prompt {
		t.Fatalf("prompt changed on search error, want the raw prompt: %q", last.Content[0].Text)
	}
}
