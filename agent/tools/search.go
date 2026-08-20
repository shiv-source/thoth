package tools

import (
	"context"
	"fmt"
	"strings"
)

// searchDefaultLimit is the default cap on search results when NewSearch is
// given a non-positive limit.
const searchDefaultLimit = 10

// Result is one match returned by a SearchFunc.
type Result struct {
	Path    string // stable identifier for the match (e.g. a workspace-relative path)
	Snippet string // short human-readable context for the match
}

// SearchFunc finds up to limit matches for query. Results are returned in rank
// order; the search tool formats only the first limit of them, so a host can
// inject any backing implementation (FTS index, plain scan, remote search).
type SearchFunc func(ctx context.Context, query string, limit int) ([]Result, error)

// Search is the "search" tool: it runs a host-injected SearchFunc over the
// workspace and returns the top results formatted as deterministic text.
type Search struct {
	search SearchFunc
	limit  int
}

// NewSearch returns the search tool backed by fn, returning at most limit
// results. A non-positive limit falls back to the default of 10.
func NewSearch(fn SearchFunc, limit int) *Search {
	if limit <= 0 {
		limit = searchDefaultLimit
	}
	return &Search{search: fn, limit: limit}
}

// Name implements Tool.
func (t *Search) Name() string { return "search" }

// Description implements Tool.
func (t *Search) Description() string {
	return "Search the workspace for query and return the top matching paths with context."
}

// Schema implements Tool.
func (t *Search) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The text to search for.",
			},
		},
		"required": []string{"query"},
	}
}

// Run implements Tool. The injected limit is enforced both when calling the
// SearchFunc and defensively on its return value, so the output is bounded no
// matter what the backing implementation does.
func (t *Search) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	query, err := stringArg(args, "query")
	if err != nil {
		return "", err
	}
	results, err := t.search(ctx, query, t.limit)
	if err != nil {
		return "", fmt.Errorf("search: %w", err)
	}
	if len(results) > t.limit {
		results = results[:t.limit]
	}
	var sb strings.Builder
	for i, r := range results {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%d. %s: %s", i+1, r.Path, r.Snippet)
	}
	return sb.String(), nil
}
