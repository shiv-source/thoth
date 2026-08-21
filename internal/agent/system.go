package agent

import (
	"github.com/shiv-source/thoth/internal/wiki"
)

// SystemPrompt returns the system prompt for one turn: the wiki's CLAUDE.md
// (the user-editable rulebook, the single source) when present, otherwise the
// scaffolded rulebook rendered for folders. It is read per Start so rulebook
// edits apply without restart; an absent or unreadable rulebook falls back to
// the template rather than failing the turn.
func SystemPrompt(w *wiki.Wiki, folders []string) (string, error) {
	b, err := w.Read("CLAUDE.md")
	if err == nil {
		return string(b), nil
	}
	return wiki.RulebookFor(folders), nil
}
