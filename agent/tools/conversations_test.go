package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeConversationStore is a scripted ConversationStore for tool tests.
type fakeConversationStore struct {
	convs []Conversation
	msgs  map[string][]ConversationMessage
	err   error
}

func (f fakeConversationStore) ListConversations(context.Context) ([]Conversation, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.convs, nil
}

func (f fakeConversationStore) Messages(_ context.Context, id string) ([]ConversationMessage, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.msgs[id], nil
}

func convTime() time.Time { return time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC) }

func TestListConversations(t *testing.T) {
	base := []Conversation{
		{ID: "id-b", Title: "planning notes", CreatedAt: convTime()},
		{ID: "id-a", Title: "daily standup", CreatedAt: convTime().Add(time.Hour)},
		{ID: "id-c", Title: "research", CreatedAt: convTime().Add(2 * time.Hour)},
	}
	tests := []struct {
		name   string
		store  ConversationStore
		args   map[string]any
		limit  int
		want   string
		errSub string
	}{
		{
			name:  "lists all conversations",
			store: fakeConversationStore{convs: base},
			args:  map[string]any{},
			want:  "2026-08-21T10:00:00Z\tid-b — planning notes\n2026-08-21T11:00:00Z\tid-a — daily standup\n2026-08-21T12:00:00Z\tid-c — research",
		},
		{
			name:  "filters by title query",
			store: fakeConversationStore{convs: base},
			args:  map[string]any{"q": "planning"},
			want:  "2026-08-21T10:00:00Z\tid-b — planning notes",
		},
		{
			name:  "query is case insensitive",
			store: fakeConversationStore{convs: base},
			args:  map[string]any{"q": "STANDUP"},
			want:  "2026-08-21T11:00:00Z\tid-a — daily standup",
		},
		{
			name:  "no matches renders empty",
			store: fakeConversationStore{convs: base},
			args:  map[string]any{"q": "nothing"},
			want:  "",
		},
		{
			name:  "empty title renders bare id",
			store: fakeConversationStore{convs: []Conversation{{ID: "id-x", CreatedAt: convTime()}}},
			args:  map[string]any{},
			want:  "2026-08-21T10:00:00Z\tid-x",
		},
		{
			name:  "caps results to limit",
			store: fakeConversationStore{convs: base},
			args:  map[string]any{},
			limit: 2,
			want:  "2026-08-21T10:00:00Z\tid-b — planning notes\n2026-08-21T11:00:00Z\tid-a — daily standup",
		},
		{
			name:   "store error propagates",
			store:  fakeConversationStore{err: errors.New("db closed")},
			args:   map[string]any{},
			errSub: "db closed",
		},
		{
			name:   "nil store is a clean error",
			store:  nil,
			args:   map[string]any{},
			errSub: "no conversation store is configured",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewListConversations(tt.store, tt.limit)
			out, err := tool.Run(context.Background(), tt.args)
			if tt.errSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSub) {
					t.Fatalf("Run error = %v, want %q", err, tt.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out != tt.want {
				t.Fatalf("Run out = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestGetConversation(t *testing.T) {
	msgs := []ConversationMessage{
		{Role: "user", Content: "hello", CreatedAt: convTime()},
		{Role: "assistant", Content: "hi there", CreatedAt: convTime().Add(time.Minute)},
		{Role: "user", Content: "thanks", CreatedAt: convTime().Add(2 * time.Minute)},
	}
	tests := []struct {
		name   string
		store  ConversationStore
		args   map[string]any
		limit  int
		want   string
		errSub string
	}{
		{
			name:  "renders transcript",
			store: fakeConversationStore{msgs: map[string][]ConversationMessage{"id-1": msgs}},
			args:  map[string]any{"id": "id-1"},
			want:  "user: hello\nassistant: hi there\nuser: thanks",
		},
		{
			name:  "renders most recent messages when capped",
			store: fakeConversationStore{msgs: map[string][]ConversationMessage{"id-1": msgs}},
			args:  map[string]any{"id": "id-1"},
			limit: 2,
			want:  "assistant: hi there\nuser: thanks",
		},
		{
			name:  "empty conversation",
			store: fakeConversationStore{msgs: map[string][]ConversationMessage{"id-1": nil}},
			args:  map[string]any{"id": "id-1"},
			want:  "no messages",
		},
		{
			name:   "missing id argument",
			store:  fakeConversationStore{msgs: map[string][]ConversationMessage{"id-1": msgs}},
			args:   map[string]any{},
			errSub: "missing argument",
		},
		{
			name:   "store error propagates",
			store:  fakeConversationStore{err: errors.New("db closed")},
			args:   map[string]any{"id": "id-1"},
			errSub: "db closed",
		},
		{
			name:   "nil store is a clean error",
			store:  nil,
			args:   map[string]any{"id": "id-1"},
			errSub: "no conversation store is configured",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewGetConversation(tt.store, tt.limit)
			out, err := tool.Run(context.Background(), tt.args)
			if tt.errSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSub) {
					t.Fatalf("Run error = %v, want %q", err, tt.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out != tt.want {
				t.Fatalf("Run out = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestGetConversationTruncatesLongMessages(t *testing.T) {
	long := strings.Repeat("x", conversationMessageMaxRunes+50)
	store := fakeConversationStore{msgs: map[string][]ConversationMessage{"id-1": {
		{Role: "assistant", Content: long},
	}}}
	out, err := NewGetConversation(store, 0).Run(context.Background(), map[string]any{"id": "id-1"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "[truncated]") {
		t.Fatalf("long message not truncated: %d runes", len([]rune(out)))
	}
	if n := len([]rune(out)); n > conversationMessageMaxRunes+32 {
		t.Fatalf("truncated output too large: %d runes", n)
	}
}

func TestSearchConversations(t *testing.T) {
	base := []Conversation{
		{ID: "id-b", Title: "planning notes", CreatedAt: convTime()},
		{ID: "id-a", Title: "daily standup", CreatedAt: convTime().Add(time.Hour)},
	}
	msgs := map[string][]ConversationMessage{
		"id-a": {{Role: "assistant", Content: "we shipped the index sync today", CreatedAt: convTime()}},
	}
	tests := []struct {
		name   string
		store  ConversationStore
		args   map[string]any
		want   string
		errSub string
	}{
		{
			name:  "title match",
			store: fakeConversationStore{convs: base, msgs: msgs},
			args:  map[string]any{"q": "planning"},
			want:  "2026-08-21T10:00:00Z\tid-b — planning notes",
		},
		{
			name:  "content match with snippet",
			store: fakeConversationStore{convs: base, msgs: msgs},
			args:  map[string]any{"q": "index sync"},
			want:  "2026-08-21T11:00:00Z\tid-a — daily standup\nwe shipped the index sync today",
		},
		{
			name:  "content match is case insensitive",
			store: fakeConversationStore{convs: base, msgs: msgs},
			args:  map[string]any{"q": "SHIPPED"},
			want:  "2026-08-21T11:00:00Z\tid-a — daily standup\nwe shipped the index sync today",
		},
		{
			name:  "no match renders empty",
			store: fakeConversationStore{convs: base, msgs: msgs},
			args:  map[string]any{"q": "nothing"},
			want:  "",
		},
		{
			name:   "empty query errors",
			store:  fakeConversationStore{convs: base, msgs: msgs},
			args:   map[string]any{"q": "  "},
			errSub: "q must not be empty",
		},
		{
			name:   "missing query argument",
			store:  fakeConversationStore{convs: base, msgs: msgs},
			args:   map[string]any{},
			errSub: "missing argument",
		},
		{
			name:   "store error propagates",
			store:  fakeConversationStore{err: errors.New("db closed")},
			args:   map[string]any{"q": "x"},
			errSub: "db closed",
		},
		{
			name:   "nil store is a clean error",
			store:  nil,
			args:   map[string]any{"q": "x"},
			errSub: "no conversation store is configured",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewSearchConversations(tt.store, 0, 0)
			out, err := tool.Run(context.Background(), tt.args)
			if tt.errSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSub) {
					t.Fatalf("Run error = %v, want %q", err, tt.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out != tt.want {
				t.Fatalf("Run out = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestSearchConversationsCapsMatches(t *testing.T) {
	base := []Conversation{
		{ID: "id-1", Title: "alpha", CreatedAt: convTime()},
		{ID: "id-2", Title: "beta", CreatedAt: convTime()},
		{ID: "id-3", Title: "alpha beta", CreatedAt: convTime()},
	}
	store := fakeConversationStore{convs: base, msgs: map[string][]ConversationMessage{}}
	tool := NewSearchConversations(store, 2, 0)
	out, err := tool.Run(context.Background(), map[string]any{"q": "alpha"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n := len(strings.Split(out, "\n")); n != 2 {
		t.Fatalf("search matches = %d, want 2:\n%s", n, out)
	}
}

func TestSearchConversationsSnippetPrefix(t *testing.T) {
	// The match sits well beyond the snippet's leading window, so the snippet
	// must be prefixed with an ellipsis and windowed around the match.
	base := []Conversation{{ID: "id-1", Title: "daily", CreatedAt: convTime()}}
	content := strings.Repeat("noise ", 40) + "needle here" // match at rune 240
	store := fakeConversationStore{convs: base, msgs: map[string][]ConversationMessage{
		"id-1": {{Role: "assistant", Content: content}},
	}}
	out, err := NewSearchConversations(store, 0, 0).Run(context.Background(), map[string]any{"q": "needle"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "… ") {
		t.Fatalf("snippet missing ellipsis prefix: %q", out)
	}
	if !strings.Contains(lines[1], "needle here") {
		t.Fatalf("snippet missing the match: %q", out)
	}
}

func TestSearchConversationsRespectsMessageCap(t *testing.T) {
	// The match lives outside the scanned window (only the last n messages are
	// scanned), so the conversation must not match.
	base := []Conversation{{ID: "id-1", Title: "daily", CreatedAt: convTime()}}
	msgs := []ConversationMessage{
		{Role: "assistant", Content: "the needle is in the first message"},
		{Role: "user", Content: "noise"},
		{Role: "assistant", Content: "noise"},
	}
	store := fakeConversationStore{convs: base, msgs: map[string][]ConversationMessage{"id-1": msgs}}
	out, err := NewSearchConversations(store, 0, 2).Run(context.Background(), map[string]any{"q": "needle"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "" {
		t.Fatalf("search = %q, want no match beyond the message cap", out)
	}
}

func TestGetConversationMessageCap(t *testing.T) {
	// get_conversation caps to the most recent messages; a store returning more
	// than the cap must not leak the oldest ones.
	msgs := []ConversationMessage{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}
	store := fakeConversationStore{msgs: map[string][]ConversationMessage{"id-1": msgs}}
	out, err := NewGetConversation(store, 2).Run(context.Background(), map[string]any{"id": "id-1"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(out, "one") || strings.Contains(out, "two") {
		t.Fatalf("get_conversation leaked oldest messages beyond cap: %q", out)
	}
	if !strings.Contains(out, "three") || !strings.Contains(out, "four") {
		t.Fatalf("get_conversation missing most recent messages: %q", out)
	}
}

func TestConversationToolSchemas(t *testing.T) {
	store := fakeConversationStore{}
	tests := []struct {
		name string
		tool Tool
		want []string
	}{
		{"list_conversations", NewListConversations(store, 0), nil},
		{"get_conversation", NewGetConversation(store, 0), []string{"id"}},
		{"search_conversations", NewSearchConversations(store, 0, 0), []string{"q"}},
	}
	for _, tt := range tests {
		if tt.tool.Name() == "" || tt.tool.Description() == "" {
			t.Fatalf("tool %q missing name or description", tt.tool.Name())
		}
		schema := tt.tool.Schema()
		if schema["type"] != "object" {
			t.Fatalf("%s schema type = %v, want object", tt.tool.Name(), schema["type"])
		}
		if tt.want == nil {
			if _, present := schema["required"]; present {
				t.Fatalf("%s schema should not require arguments", tt.tool.Name())
			}
			continue
		}
		got, ok := schema["required"].([]string)
		if !ok || strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Fatalf("%s required = %v, want %v", tt.tool.Name(), schema["required"], tt.want)
		}
	}
}
