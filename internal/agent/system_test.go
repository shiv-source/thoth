package agent

import (
	"testing"

	"github.com/shiv-source/thoth/internal/wiki"
)

func TestSystemPromptReadsRulebook(t *testing.T) {
	w := newWiki(t, "rulebook v1")
	got, err := SystemPrompt(w, wiki.Folders())
	if err != nil {
		t.Fatalf("SystemPrompt: %v", err)
	}
	if got != "rulebook v1" {
		t.Fatalf("got %q, want rulebook v1", got)
	}

	if err := writeRulebook(t, w, "rulebook v2"); err != nil {
		t.Fatal(err)
	}
	got, err = SystemPrompt(w, wiki.Folders())
	if err != nil {
		t.Fatalf("SystemPrompt: %v", err)
	}
	if got != "rulebook v2" {
		t.Fatalf("got %q, want rulebook v2", got)
	}
}

func TestSystemPromptFallsBackToRulebook(t *testing.T) {
	w := wiki.New(t.TempDir()) // no CLAUDE.md
	folders := []string{"inbox", "knowledge"}
	got, err := SystemPrompt(w, folders)
	if err != nil {
		t.Fatalf("SystemPrompt: %v", err)
	}
	if want := wiki.RulebookFor(folders); got != want {
		t.Fatalf("got %q, want rulebook fallback", got)
	}
}
