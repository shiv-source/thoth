package tools

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// grepMaxMatches caps the number of grep results a single call returns, so a
// broad pattern cannot flood the model context.
const grepMaxMatches = 100

// Grep is the "grep" tool: it matches a regular expression against note
// bodies across the wiki and returns path:line matches, size-capped.
type Grep struct {
	fs    FS
	limit int
}

// NewGrep returns the grep tool backed by fs. A non-positive limit falls back
// to the default result cap.
func NewGrep(fs FS, limit int) *Grep {
	if limit <= 0 {
		limit = grepMaxMatches
	}
	return &Grep{fs: fs, limit: limit}
}

// Name implements Tool.
func (t *Grep) Name() string { return "grep" }

// Description implements Tool.
func (t *Grep) Description() string {
	return "Search note bodies for a regular expression pattern and return matching lines with their paths and line numbers. Path is relative to the wiki root."
}

// Schema implements Tool.
func (t *Grep) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regular expression to search for.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional directory to restrict the search to. Defaults to the wiki root.",
			},
		},
		"required": []string{"pattern"},
	}
}

// Run implements Tool.
func (t *Grep) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	pattern, err := stringArg(args, "pattern")
	if err != nil {
		return "", err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("grep: invalid pattern: %w", err)
	}
	path, err := stringArgDefault(args, "path", ".")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	matches := 0
	truncated := false
	err = walkNotes(t.fs, rel, func(noteRel string, _ Note, body []byte) error {
		if matches >= t.limit {
			return errTreeTooLarge
		}
		lines := strings.Split(string(body), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				if matches >= t.limit {
					truncated = true
					return errTreeTooLarge
				}
				if matches > 0 {
					sb.WriteByte('\n')
				}
				fmt.Fprintf(&sb, "%s:%d:%s", noteRel, i+1, line)
				matches++
			}
		}
		return nil
	})
	if errors.Is(err, errTreeTooLarge) {
		truncated = true
	} else if err != nil {
		return "", fmt.Errorf("grep: %w", err)
	}
	if truncated {
		fmt.Fprintf(&sb, "\n[grep truncated: more than %d matches]", t.limit)
	}
	return sb.String(), nil
}
