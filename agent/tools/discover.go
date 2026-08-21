package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// recentDefaultLimit is the default result cap when NewListRecent is given a
// non-positive limit.
const recentDefaultLimit = 10

// tagDefaultLimit is the default result cap for search_by_tag.
const tagDefaultLimit = 20

// ListRecent is the "list_recent" tool: it lists the most recently modified
// notes, newest first, with each note's title. Modification time comes from
// the FS seam's Stat, so hosts control what "modified" means.
type ListRecent struct {
	fs    FS
	limit int
}

// NewListRecent returns the list_recent tool backed by fs. A non-positive
// limit falls back to the default of 10.
func NewListRecent(fs FS, limit int) *ListRecent {
	if limit <= 0 {
		limit = recentDefaultLimit
	}
	return &ListRecent{fs: fs, limit: limit}
}

// Name implements Tool.
func (t *ListRecent) Name() string { return "list_recent" }

// Description implements Tool.
func (t *ListRecent) Description() string {
	return "List the most recently modified notes, newest first, with their titles."
}

// Schema implements Tool.
func (t *ListRecent) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of notes to return. Defaults to 10.",
			},
		},
	}
}

// Run implements Tool.
func (t *ListRecent) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	limit, err := intArgDefault(args, "limit", t.limit)
	if err != nil {
		return "", err
	}
	type entry struct {
		rel   string
		mod   time.Time
		title string
	}
	var entries []entry
	err = walkNotes(t.fs, ".", func(rel string, note Note, _ []byte) error {
		fi, err := t.fs.Stat(rel)
		if err != nil {
			return nil // unreadable notes are skipped, not fatal
		}
		entries = append(entries, entry{rel: rel, mod: fi.ModTime(), title: note.Title})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("list_recent: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].mod.After(entries[j].mod) })
	if len(entries) > limit {
		entries = entries[:limit]
	}
	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%s\t%s", e.mod.UTC().Format(time.RFC3339), e.rel)
		if e.title != "" {
			fmt.Fprintf(&sb, " — %s", e.title)
		}
	}
	return sb.String(), nil
}

// SearchByTag is the "search_by_tag" tool: it returns notes whose frontmatter
// tags include the given tag, sorted by path.
type SearchByTag struct {
	fs    FS
	limit int
}

// NewSearchByTag returns the search_by_tag tool backed by fs. A non-positive
// limit falls back to the default of 20.
func NewSearchByTag(fs FS, limit int) *SearchByTag {
	if limit <= 0 {
		limit = tagDefaultLimit
	}
	return &SearchByTag{fs: fs, limit: limit}
}

// Name implements Tool.
func (t *SearchByTag) Name() string { return "search_by_tag" }

// Description implements Tool.
func (t *SearchByTag) Description() string {
	return "Return the paths of notes whose frontmatter tags include the given tag."
}

// Schema implements Tool.
func (t *SearchByTag) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tag": map[string]any{
				"type":        "string",
				"description": "The tag to search for.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of notes to return. Defaults to 20.",
			},
		},
		"required": []string{"tag"},
	}
}

// Run implements Tool.
func (t *SearchByTag) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	tag, err := stringArg(args, "tag")
	if err != nil {
		return "", err
	}
	limit, err := intArgDefault(args, "limit", t.limit)
	if err != nil {
		return "", err
	}
	var paths []string
	err = walkNotes(t.fs, ".", func(rel string, note Note, _ []byte) error {
		for _, t := range note.Tags {
			if t == tag {
				paths = append(paths, filepath.ToSlash(rel))
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("search_by_tag: %w", err)
	}
	sort.Strings(paths)
	if len(paths) > limit {
		paths = paths[:limit]
	}
	return strings.Join(paths, "\n"), nil
}
