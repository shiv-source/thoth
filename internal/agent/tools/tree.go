package tools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/wiki"
)

// ListTree is the "list_tree" tool: a dirs-first recursive tree of the wiki
// with each note annotated by its frontmatter title. Output is size-capped.
type ListTree struct {
	fs       agenttools.FS
	maxBytes int
	maxNodes int
}

// NewListTree returns the list_tree tool backed by fs. A non-positive maxBytes
// falls back to the default read cap; a non-positive maxNodes to the default
// node cap.
func NewListTree(fs agenttools.FS, maxBytes, maxNodes int) *ListTree {
	if maxBytes <= 0 {
		maxBytes = agenttools.MaxReadBytes
	}
	if maxNodes <= 0 {
		maxNodes = 5000
	}
	return &ListTree{fs: fs, maxBytes: maxBytes, maxNodes: maxNodes}
}

// Name implements agenttools.Tool.
func (t *ListTree) Name() string { return "list_tree" }

// Description implements agenttools.Tool.
func (t *ListTree) Description() string {
	return "List the full note tree, directories first, with each note's frontmatter title. Path is relative to the wiki root."
}

// Schema implements agenttools.Tool.
func (t *ListTree) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the directory to list. Defaults to the wiki root.",
			},
		},
	}
}

// Run implements agenttools.Tool.
func (t *ListTree) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := agenttools.StringArgDefault(args, "path", ".")
	if err != nil {
		return "", err
	}
	rel, err := agenttools.CleanRel(path)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	count := 0
	writeNode := func(depth int, name, title string, isDir bool) {
		if title != "" {
			fmt.Fprintf(&sb, "%s%s — %s\n", strings.Repeat("  ", depth), name, title)
		} else {
			fmt.Fprintf(&sb, "%s%s\n", strings.Repeat("  ", depth), name)
		}
		count++
	}
	err = t.walk(rel, 0, writeNode, &count)
	if errors.Is(err, agenttools.ErrTreeTooLarge) {
		fmt.Fprintf(&sb, "\n[tree truncated: more than %d nodes]", t.maxNodes)
	}
	out := sb.String()
	if len(out) > t.maxBytes {
		return out[:t.maxBytes] + agenttools.TruncationMarker(t.maxBytes), nil
	}
	if count == 0 {
		return "", nil
	}
	return out, nil
}

// walk renders the subtree at rel into sb via writeNode, dirs first, sorted by
// name, with each note annotated by its frontmatter title. It follows the
// wiki's Visible rules — dotfiles, the root rulebook and the attachments
// subtree are skipped. It returns agenttools.ErrTreeTooLarge when the node cap
// is exceeded.
func (t *ListTree) walk(rel string, depth int, writeNode func(depth int, name, title string, isDir bool), count *int) error {
	if *count >= t.maxNodes {
		return agenttools.ErrTreeTooLarge
	}
	entries, err := t.fs.ReadDir(rel)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		child := filepath.Join(rel, e.Name())
		slashChild := filepath.ToSlash(child)
		if !wiki.Visible(slashChild, e.IsDir()) {
			continue
		}
		if e.IsDir() {
			writeNode(depth, e.Name()+"/", "", true)
			if err := t.walk(child, depth+1, writeNode, count); err != nil {
				return err
			}
			continue
		}
		title := ""
		if data, err := t.fs.ReadFile(child); err == nil {
			if meta, _, err := wiki.ParseNote(data); err == nil {
				title = meta.Title
			}
		}
		writeNode(depth, e.Name(), title, false)
	}
	return nil
}
