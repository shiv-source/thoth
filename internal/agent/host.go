package agent

import (
	"context"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/doctor"
	"github.com/shiv-source/thoth/internal/store"
)

// DoctorHealth adapts doctor.Run to the tools.HealthFunc seam: the agent's
// system_health tool runs the same install checks as `thoth doctor` and the
// Settings → Doctor tab. dir is the data dir the checks probe ("" → ~/.thoth);
// serve passes its own, so a dev run reports against ~/.thoth/dev.
func DoctorHealth(dir string) agenttools.HealthFunc {
	return func(ctx context.Context) ([]agenttools.HealthReport, error) {
		checks := doctor.Run(ctx, doctor.Options{Dir: dir})
		out := make([]agenttools.HealthReport, 0, len(checks))
		for _, c := range checks {
			out = append(out, agenttools.HealthReport{Name: c.Name, OK: c.OK, Message: c.Message})
		}
		return out, nil
	}
}

// conversationStore adapts store.Store to the tools.ConversationStore seam the
// conversation-memory tools recall through, mapping the store's types to the
// wiki-agnostic agent types. The host conversation/message tables are the only
// place the tools look, so the adapter is a pure field-for-field copy.
type conversationStore struct {
	st *store.Store
}

// ListConversations implements agenttools.ConversationStore.
func (a conversationStore) ListConversations(ctx context.Context) ([]agenttools.Conversation, error) {
	convs, err := a.st.ListConversations()
	if err != nil {
		return nil, err
	}
	out := make([]agenttools.Conversation, 0, len(convs))
	for _, c := range convs {
		out = append(out, agenttools.Conversation{ID: c.ID, Title: c.Title, CreatedAt: c.CreatedAt})
	}
	return out, nil
}

// Messages implements agenttools.ConversationStore.
func (a conversationStore) Messages(ctx context.Context, convID string) ([]agenttools.ConversationMessage, error) {
	msgs, err := a.st.Messages(convID)
	if err != nil {
		return nil, err
	}
	out := make([]agenttools.ConversationMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, agenttools.ConversationMessage{Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt})
	}
	return out, nil
}
