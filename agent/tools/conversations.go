package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// conversationDefaultLimit is the default result cap for list_conversations
// when NewListConversations is given a non-positive limit.
const conversationDefaultLimit = 20

// conversationMessageLimit is the default cap on messages rendered by
// get_conversation and on the number of messages scanned per conversation by
// search_conversations.
const conversationMessageLimit = 100

// conversationSearchLimit is the default result cap for search_conversations.
const conversationSearchLimit = 10

// conversationMessageMaxRunes caps each message's rendered content, so one
// long assistant reply cannot blow up the transcript.
const conversationMessageMaxRunes = 2000

// Conversation is one conversation's metadata, as exposed by a
// ConversationStore.
type Conversation struct {
	ID        string
	Title     string
	CreatedAt time.Time
}

// ConversationMessage is one message in a conversation.
type ConversationMessage struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// ConversationStore is the host-injected seam the conversation-memory tools
// recall through. It exposes exactly what the tools need — the conversation
// list and one conversation's messages — so a host can back it with any
// storage (Thoth implements it over internal/store).
type ConversationStore interface {
	ListConversations(ctx context.Context) ([]Conversation, error)
	Messages(ctx context.Context, convID string) ([]ConversationMessage, error)
}

// ListConversations is the "list_conversations" tool: it lists the most
// recent conversations, newest first, optionally filtered to those whose
// title contains q.
type ListConversations struct {
	store ConversationStore
	limit int
}

// NewListConversations returns the list_conversations tool backed by store. A
// non-positive limit falls back to the default of 20.
func NewListConversations(store ConversationStore, limit int) *ListConversations {
	if limit <= 0 {
		limit = conversationDefaultLimit
	}
	return &ListConversations{store: store, limit: limit}
}

// Name implements Tool.
func (t *ListConversations) Name() string { return "list_conversations" }

// Description implements Tool.
func (t *ListConversations) Description() string {
	return "List the most recent conversations, newest first, with their titles and creation time. Optionally filter to conversations whose title contains q. Read-only."
}

// Schema implements Tool.
func (t *ListConversations) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q": map[string]any{
				"type":        "string",
				"description": "Optional text to filter conversations by title.",
			},
		},
	}
}

// Run implements Tool.
func (t *ListConversations) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.store == nil {
		return "", errors.New("list_conversations: no conversation store is configured")
	}
	q, err := StringArgDefault(args, "q", "")
	if err != nil {
		return "", err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	convs, err := t.store.ListConversations(ctx)
	if err != nil {
		return "", fmt.Errorf("list_conversations: %w", err)
	}
	var sb strings.Builder
	n := 0
	for _, c := range convs {
		if n >= t.limit {
			break
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) {
			continue
		}
		if n > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%s\t%s", c.CreatedAt.UTC().Format(time.RFC3339), c.ID)
		if c.Title != "" {
			fmt.Fprintf(&sb, " — %s", c.Title)
		}
		n++
	}
	return sb.String(), nil
}

// GetConversation is the "get_conversation" tool: it returns one
// conversation's messages as a transcript, most recent last, so the agent can
// recall what was discussed.
type GetConversation struct {
	store ConversationStore
	limit int
}

// NewGetConversation returns the get_conversation tool backed by store. A
// non-positive limit falls back to the default of 100 messages.
func NewGetConversation(store ConversationStore, limit int) *GetConversation {
	if limit <= 0 {
		limit = conversationMessageLimit
	}
	return &GetConversation{store: store, limit: limit}
}

// Name implements Tool.
func (t *GetConversation) Name() string { return "get_conversation" }

// Description implements Tool.
func (t *GetConversation) Description() string {
	return "Return one conversation's messages as a transcript, oldest first. Pass the id from list_conversations. Read-only."
}

// Schema implements Tool.
func (t *GetConversation) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "The conversation id to read.",
			},
		},
		"required": []string{"id"},
	}
}

// Run implements Tool.
func (t *GetConversation) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.store == nil {
		return "", errors.New("get_conversation: no conversation store is configured")
	}
	id, err := StringArg(args, "id")
	if err != nil {
		return "", err
	}
	msgs, err := t.store.Messages(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get_conversation: %w", err)
	}
	if len(msgs) > t.limit {
		msgs = msgs[len(msgs)-t.limit:]
	}
	var sb strings.Builder
	for i, m := range msgs {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%s: %s", m.Role, truncateRunes(m.Content, conversationMessageMaxRunes))
	}
	if sb.Len() == 0 {
		return "no messages", nil
	}
	return sb.String(), nil
}

// SearchConversations is the "search_conversations" tool: it finds
// conversations whose title or message content contains q, returning each
// matching conversation with a snippet of the first matching message. Read-only.
type SearchConversations struct {
	store   ConversationStore
	matches int
	msgs    int
}

// NewSearchConversations returns the search_conversations tool backed by
// store. A non-positive matches cap falls back to 10; a non-positive per-
// conversation message cap falls back to 100.
func NewSearchConversations(store ConversationStore, matches, msgs int) *SearchConversations {
	if matches <= 0 {
		matches = conversationSearchLimit
	}
	if msgs <= 0 {
		msgs = conversationMessageLimit
	}
	return &SearchConversations{store: store, matches: matches, msgs: msgs}
}

// Name implements Tool.
func (t *SearchConversations) Name() string { return "search_conversations" }

// Description implements Tool.
func (t *SearchConversations) Description() string {
	return "Search across conversation titles and message content for q and return the matching conversations with a snippet of the matching message. Read-only."
}

// Schema implements Tool.
func (t *SearchConversations) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q": map[string]any{
				"type":        "string",
				"description": "The text to search for in conversation titles and messages.",
			},
		},
		"required": []string{"q"},
	}
}

// Run implements Tool.
func (t *SearchConversations) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.store == nil {
		return "", errors.New("search_conversations: no conversation store is configured")
	}
	q, err := StringArg(args, "q")
	if err != nil {
		return "", err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return "", errors.New("search_conversations: q must not be empty")
	}
	convs, err := t.store.ListConversations(ctx)
	if err != nil {
		return "", fmt.Errorf("search_conversations: %w", err)
	}
	var sb strings.Builder
	n := 0
	for _, c := range convs {
		if n >= t.matches {
			break
		}
		snippet, ok, err := t.match(ctx, c, q)
		if err != nil {
			return "", fmt.Errorf("search_conversations: %w", err)
		}
		if !ok {
			continue
		}
		if n > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%s\t%s", c.CreatedAt.UTC().Format(time.RFC3339), c.ID)
		if c.Title != "" {
			fmt.Fprintf(&sb, " — %s", c.Title)
		}
		if snippet != "" {
			fmt.Fprintf(&sb, "\n%s", snippet)
		}
		n++
	}
	return sb.String(), nil
}

// match reports whether conversation c matches q and, for content matches, a
// snippet of the first matching message. A title match wins without scanning
// messages. q is expected already lowercased.
func (t *SearchConversations) match(ctx context.Context, c Conversation, q string) (string, bool, error) {
	if strings.Contains(strings.ToLower(c.Title), q) {
		return "", true, nil
	}
	msgs, err := t.store.Messages(ctx, c.ID)
	if err != nil {
		return "", false, err
	}
	if len(msgs) > t.msgs {
		msgs = msgs[len(msgs)-t.msgs:]
	}
	for _, m := range msgs {
		lower := strings.ToLower(m.Content)
		if i := strings.Index(lower, q); i >= 0 {
			// Convert the byte offset in the lowercased content to a rune
			// index so the snippet window never splits a multi-byte rune.
			matchRune := utf8.RuneCountInString(lower[:i])
			return snippet(m.Content, matchRune, conversationMessageMaxRunes), true, nil
		}
	}
	return "", false, nil
}

// snippet extracts a bounded window of content around the match at rune
// index matchRune, prefixed with an ellipsis when earlier content is cut.
func snippet(content string, matchRune, maxRunes int) string {
	const context = 60
	runes := []rune(content)
	start, head := 0, ""
	if matchRune > context {
		start = matchRune - context
		head = "… "
	}
	runes = runes[start:]
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return head + string(runes)
}

// truncateRunes returns the first n runes of s, marking truncation.
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "… [truncated]"
}
