package agent

import "context"

// Summarizer condenses a span of dropped history into a short stand-in the
// loop may inject when it caps a conversation. No default implementation is
// provided; hosts opt in when they want dropped turns to survive as a summary
// instead of vanishing.
type Summarizer interface {
	// Summarize returns a concise summary of msgs. The loop decides whether
	// to call it for the turns it drops and where to place the result.
	Summarize(ctx context.Context, msgs []Message) (string, error)
}

// Cap trims a conversation to its last n user-initiated turns. The cut always
// falls on a user turn, so a trimmed history starts with a user message and
// never splits an assistant tool_use from its tool_result: a dropped turn
// takes its whole tool exchange with it. When n <= 0, or there are at most n
// user turns, messages is returned unchanged. Cap is pure and deterministic —
// it neither reads message contents nor allocates beyond the result.
func Cap(messages []Message, n int) []Message {
	if n <= 0 || len(messages) == 0 {
		return messages
	}
	start := nthUserTurn(messages, n)
	if start < 0 {
		return messages // fewer than n user turns: nothing to drop
	}
	// Never keep a tool_result whose tool_use was dropped with an earlier
	// turn; back the cut up until the window starts on a self-contained turn.
	for start > 0 && hasOrphanedResult(messages, start) {
		prev, ok := previousUser(messages, start)
		if !ok {
			return messages
		}
		start = prev
	}
	if start == 0 {
		return messages
	}
	return messages[start:]
}

// nthUserTurn returns the index of the n-th-from-last user turn, or -1 when
// there are fewer than n user turns.
func nthUserTurn(messages []Message, n int) int {
	seen := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			seen++
			if seen == n {
				return i
			}
		}
	}
	return -1
}

// previousUser returns the index of the user turn before start, if any.
func previousUser(messages []Message, start int) (int, bool) {
	for i := start - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			return i, true
		}
	}
	return 0, false
}

// hasOrphanedResult reports whether the window beginning at start contains a
// tool_result whose tool_use was dropped with an earlier turn.
func hasOrphanedResult(messages []Message, start int) bool {
	seen := make(map[string]bool)
	for i := start; i < len(messages); i++ {
		for _, b := range messages[i].Content {
			switch b.Type {
			case BlockToolUse:
				seen[b.ID] = true
			case BlockToolResult:
				if !seen[b.ToolUseID] {
					return true
				}
			}
		}
	}
	return false
}

// SystemMarker is the marker index CacheMarkers reports for the system
// prompt, which is always a prompt-cache breakpoint.
const SystemMarker = -1

// CacheMarkers returns the prompt-cache breakpoint indices for the messages
// of one request. The system prompt is always a breakpoint, reported as
// SystemMarker; the stable history prefix — every message before the final
// user turn — contributes the index of its last message. The result is
// deterministic and holds at most two indices, and markers never land
// mid-conversation. A request with no stable prefix reports only the system
// marker. Providers map the returned indices onto their own cache_control
// wire markers.
func CacheMarkers(messages []Message) []int {
	last := lastUserTurn(messages)
	if last <= 0 {
		return []int{SystemMarker}
	}
	return []int{SystemMarker, last - 1}
}

// lastUserTurn returns the index of the final user turn, or -1 when there is
// none.
func lastUserTurn(messages []Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			return i
		}
	}
	return -1
}
