package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/wiki"
)

// TestSystemHealthRegisteredWhenConfigured asserts system_health is registered
// only when RegistryOptions carries a Health func, so a host that does not
// wire it does not expose it to the model.
func TestSystemHealthRegisteredWhenConfigured(t *testing.T) {
	health := DoctorHealth(t.TempDir())
	reg, err := registry(RegistryOptions{Wiki: wiki.New(t.TempDir()), Health: health})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if _, err := reg.Get("system_health"); err != nil {
		t.Fatalf("missing system_health: %v", err)
	}
	reg, err = registry(RegistryOptions{Wiki: wiki.New(t.TempDir())})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if _, err := reg.Get("system_health"); err == nil {
		t.Fatal("system_health should not be registered without Health")
	}
}

// TestConversationToolsRegisteredWhenConfigured asserts the conversation tools
// are registered only when RegistryOptions carries a Conversations seam.
func TestConversationToolsRegisteredWhenConfigured(t *testing.T) {
	st := openStore(t)
	reg, err := registry(RegistryOptions{Wiki: wiki.New(t.TempDir()), Conversations: conversationStore{st: st}})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, name := range []string{"list_conversations", "get_conversation", "search_conversations"} {
		if _, err := reg.Get(name); err != nil {
			t.Fatalf("missing conversation tool %q: %v", name, err)
		}
	}
	reg, err = registry(RegistryOptions{Wiki: wiki.New(t.TempDir())})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, name := range []string{"list_conversations", "get_conversation", "search_conversations"} {
		if _, err := reg.Get(name); err == nil {
			t.Fatalf("conversation tool %q should not be registered without a store", name)
		}
	}
}

// TestConversationStoreAdapter asserts the host adapter round-trips a real
// store: conversations list newest first, messages in insertion order.
func TestConversationStoreAdapter(t *testing.T) {
	st := openStore(t)
	adapter := conversationStore{st: st}
	ctx := context.Background()

	id, err := st.CreateConversation("first")
	if err != nil {
		t.Fatal(err)
	}
	stID, err := st.CreateConversation("second")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(id, "user", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(id, "assistant", "hi"); err != nil {
		t.Fatal(err)
	}

	convs, err := adapter.ListConversations(ctx)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("ListConversations = %d conversations, want 2", len(convs))
	}
	if convs[0].ID != stID || convs[0].Title != "second" {
		t.Fatalf("newest conversation = %+v, want second", convs[0])
	}
	if convs[1].Title != "first" || !convs[1].CreatedAt.After(time.Time{}) {
		t.Fatalf("oldest conversation = %+v, want first with a created time", convs[1])
	}

	msgs, err := adapter.Messages(ctx, id)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("Messages = %+v, want user then assistant", msgs)
	}
	if msgs[0].Content != "hello" || msgs[1].Content != "hi" {
		t.Fatalf("Messages content = %q, %q", msgs[0].Content, msgs[1].Content)
	}
}

// TestDoctorHealth asserts the health seam runs the real doctor suite against
// a data dir and reports every check by name.
func TestDoctorHealth(t *testing.T) {
	dir := t.TempDir() // fresh, unprovisioned data dir → all checks run
	reports, err := DoctorHealth(dir)(context.Background())
	if err != nil {
		t.Fatalf("DoctorHealth: %v", err)
	}
	if len(reports) == 0 {
		t.Fatal("DoctorHealth returned no reports")
	}
	names := map[string]bool{}
	for _, r := range reports {
		if r.Name == "" {
			t.Fatalf("report %+v has no name", r)
		}
		names[r.Name] = true
	}
	for _, want := range []string{"wiki", "database", "index"} {
		if !names[want] {
			t.Fatalf("health reports missing check %q (got %d checks)", want, len(reports))
		}
	}
}

// TestClientWiresConversationAndHealthTools asserts Client.New always wires the
// conversation tools from the store and registers system_health when the
// WithHealthFunc option is given.
func TestClientWiresConversationAndHealthTools(t *testing.T) {
	prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}
	dir := t.TempDir()
	st := openStore(t)
	c, err := New("claude-test", "key", wiki.New(t.TempDir()), st, nil,
		WithProvider(prov),
		WithHealthFunc(DoctorHealth(dir)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, name := range []string{"list_conversations", "get_conversation", "search_conversations", "system_health"} {
		if _, err := c.tools.Get(name); err != nil {
			t.Fatalf("missing tool %q on the client registry: %v", name, err)
		}
	}

	c, err = New("claude-test", "key", wiki.New(t.TempDir()), openStore(t), nil,
		WithProvider(prov),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.tools.Get("system_health"); err == nil {
		t.Fatal("system_health should not be registered without WithHealthFunc")
	}
	if _, err := c.tools.Get("list_conversations"); err != nil {
		t.Fatalf("list_conversations should always be wired from the store: %v", err)
	}
}

// TestConversationToolsRoundTrip asserts the wired conversation tools work end
// to end against a real store with a seeded conversation.
func TestConversationToolsRoundTrip(t *testing.T) {
	st := openStore(t)
	id, err := st.CreateConversation("planning the index")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(id, "user", "how is the index sync going?"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMessage(id, "assistant", "we shipped it today"); err != nil {
		t.Fatal(err)
	}
	reg, err := registry(RegistryOptions{Wiki: wiki.New(t.TempDir()), Conversations: conversationStore{st: st}})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	list, err := reg.Get("list_conversations")
	if err != nil {
		t.Fatal(err)
	}
	out, err := list.Run(ctx, map[string]any{"q": "index"})
	if err != nil {
		t.Fatalf("list_conversations: %v", err)
	}
	if !strings.Contains(out, id) || !strings.Contains(out, "planning the index") {
		t.Fatalf("list_conversations = %q, want the seeded conversation", out)
	}

	get, err := reg.Get("get_conversation")
	if err != nil {
		t.Fatal(err)
	}
	out, err = get.Run(ctx, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("get_conversation: %v", err)
	}
	if !strings.Contains(out, "user: how is the index sync going?") || !strings.Contains(out, "assistant: we shipped it today") {
		t.Fatalf("get_conversation = %q, want the transcript", out)
	}

	search, err := reg.Get("search_conversations")
	if err != nil {
		t.Fatal(err)
	}
	out, err = search.Run(ctx, map[string]any{"q": "shipped"})
	if err != nil {
		t.Fatalf("search_conversations: %v", err)
	}
	if !strings.Contains(out, "shipped") {
		t.Fatalf("search_conversations = %q, want the content snippet", out)
	}
}
