package agent

import (
	"testing"

	"github.com/shiv-source/thoth/internal/wiki"
)

func TestSystemPromptReadsRulebook(t *testing.T) {
	w := newWiki(t, "rulebook v1")
	if got := SystemPrompt(w, wiki.Folders()); got != "rulebook v1" {
		t.Fatalf("got %q, want rulebook v1", got)
	}

	if err := writeRulebook(t, w, "rulebook v2"); err != nil {
		t.Fatal(err)
	}
	if got := SystemPrompt(w, wiki.Folders()); got != "rulebook v2" {
		t.Fatalf("got %q, want rulebook v2", got)
	}
}

func TestSystemPromptFallsBackToRulebook(t *testing.T) {
	w := wiki.New(t.TempDir()) // no CLAUDE.md
	folders := []string{"inbox", "knowledge"}
	if want := wiki.RulebookFor(folders); SystemPrompt(w, folders) != want {
		t.Fatalf("got %q, want rulebook fallback", SystemPrompt(w, folders))
	}
}
