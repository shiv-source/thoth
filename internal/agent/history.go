package agent

import (
	"context"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/store"
)

// History returns the history provider for one conversation: the store's
// prior user/assistant messages mapped to agent messages. Every trailing user
// message is dropped — the Hub persists the in-flight prompt with AddMessage
// before Start, and the agent loop appends it again, so keeping them would
// duplicate the prompt. A failed or cancelled turn leaves an orphaned user
// message (no assistant reply) behind; dropping all of them keeps the history
// from replaying dangling prompts. The loop caps the result to HistoryCap
// turns.
func History(st *store.Store) func(ctx context.Context, convID string) ([]agentlib.Message, error) {
	return func(ctx context.Context, convID string) ([]agentlib.Message, error) {
		msgs, err := st.Messages(convID)
		if err != nil {
			return nil, err
		}
		out := make([]agentlib.Message, 0, len(msgs))
		for _, m := range msgs {
			if m.Role != "user" && m.Role != "assistant" {
				continue
			}
			out = append(out, agentlib.Message{
				Role:    m.Role,
				Content: []agentlib.Block{agentlib.NewTextBlock(m.Content)},
			})
		}
		for len(out) > 0 && out[len(out)-1].Role == "user" {
			out = out[:len(out)-1]
		}
		return out, nil
	}
}
