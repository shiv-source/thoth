package agent

import (
	"strings"
	"testing"
)

func TestSearchQuery(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{"plain words", "how do notes work", `"how" OR "do" OR "notes" OR "work"`},
		{"hyphens are separators, not operators", "pre-searched notes", `"pre" OR "searched" OR "notes"`},
		{"quotes cannot break the match", `"unterminated phrase`, `"unterminated" OR "phrase"`},
		{"punctuation and case are normalized", "What is FTS5, bm25?", `"what" OR "is" OR "fts5" OR "bm25"`},
		{"duplicate words collapse", "search the search index", `"search" OR "the" OR "index"`},
		{"no words yields empty", "??!!()", ""},
		{"unicode words are kept", "café notes", `"café" OR "notes"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchQuery(tt.prompt); got != tt.want {
				t.Fatalf("searchQuery(%q) = %q, want %q", tt.prompt, got, tt.want)
			}
		})
	}
}

func TestSearchQueryProducesValidFTS5ForPunctuation(t *testing.T) {
	// Every sanitized query is a quoted OR list, so it must never contain an
	// FTS5 operator that a raw prompt could smuggle in.
	for _, prompt := range []string{`"`, `a-b`, `a AND b`, `(a)`, `a:b`, `a*`} {
		q := searchQuery(prompt)
		if strings.ContainsAny(q, `()*:`) {
			t.Fatalf("searchQuery(%q) = %q, contains FTS5 operator syntax", prompt, q)
		}
	}
}
