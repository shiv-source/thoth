package agent

import (
	"fmt"
	"strings"
	"unicode"
)

// contextInjectionLimit is the number of pre-searched notes fed into a turn's
// first prompt when context injection is on. bm25 already ranks the index's
// results, so a small cap with the snippet window keeps the injected block's
// token cost bounded.
const contextInjectionLimit = 5

// contextInjectionHeader prefixes the pre-searched block so the model
// distinguishes provided context from the live question. The rulebook tells
// the model to answer from it before reaching for the retrieval tools.
const contextInjectionHeader = "Relevant wiki notes (pre-searched):"

// enrichPrompt pre-searches the wiki for the user prompt and, when matches
// exist, prepends a bounded context block so the model can answer from
// provided context instead of driving a retrieval tool loop. The result is
// part of the first provider turn's user message. An empty search (no
// matches) returns the prompt unchanged.
func (c *Client) enrichPrompt(prompt string) (string, error) {
	if !c.contextInjection || c.index == nil {
		return prompt, nil
	}
	q := searchQuery(prompt)
	if q == "" {
		return prompt, nil
	}
	results, err := c.index.SearchClean(q, contextInjectionLimit)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return prompt, nil
	}
	var sb strings.Builder
	sb.WriteString(contextInjectionHeader)
	sb.WriteByte('\n')
	for _, r := range results {
		fmt.Fprintf(&sb, "- %s: %s\n", r.Path, r.Snippet)
	}
	return sb.String() + prompt, nil
}

// searchQuery reduces a free-form prompt to an FTS5-safe OR query: the words
// are quoted and joined, so sentence punctuation (hyphens, quotes, colons)
// can never be misparsed as FTS5 operators, and a sentence's filler words
// never AND every match away. Returns "" when the prompt has no words.
func searchQuery(prompt string) string {
	words := strings.FieldsFunc(prompt, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if len(words) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(words))
	parts := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ToLower(w)
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		parts = append(parts, `"`+w+`"`)
	}
	return strings.Join(parts, " OR ")
}
