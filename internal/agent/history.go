package agent

import (
	"context"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/store"
)

// History returns the history provider for one conversation: the store's
// prior user/assistant messages mapped to agent messages. The trailing user
// message is dropped — the Hub persists the in-flight prompt with AddMessage
// before Start, and the agent loop appends it again, so keeping it would
// duplicate the prompt. The loop caps the result to HistoryCap turns.
func History(st *store.Store) func(ctx context.Context, convID string) ([]agentlib.Message, error) {
	return func(ctx context.Context, convID string) ([]agentlib.Message, error) {
		msgs, err := st.Messages(convID)
		if err != nil {
			return nil, err
		}
		out := make([]agentlib.Message, 0, len(msgs))
		for i, m := range msgs {
			if m.Role != "user" && m.Role != "assistant" {
				continue
			}
			if i == len(msgs)-1 && m.Role == "user" {
				continue // the in-flight prompt the loop appends itself
			}
			out = append(out, agentlib.Message{
				Role:    m.Role,
				Content: []agentlib.Block{agentlib.NewTextBlock(m.Content)},
			})
		}
		return out, nil
	}
}
