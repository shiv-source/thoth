package tools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// maxTreeNodes caps the number of nodes list_tree and walkNotes visit, so a
// pathological directory tree cannot exhaust memory. It matches the wiki tree
// convention of bounding output size.
const maxTreeNodes = 5000

// walkNotes walks the note tree rooted at rel (relative to the FS root) and
// calls fn for every markdown note. Hidden paths — any segment starting with a
// dot, mirroring the wiki's Visible rule — are skipped so the agent and the
// dashboard agree on what a note is. fn is given the note's relative path and
// its parsed frontmatter plus body.
func walkNotes(fs FS, rel string, fn func(rel string, note Note, body []byte) error) error {
	count := 0
	return walkNotesBounded(fs, rel, fn, &count, maxTreeNodes)
}

// walkNotesBounded walks like walkNotes but stops after visiting max nodes,
// returning errTreeTooLarge so callers can truncate output. A max of 0 means
// no node limit. count is shared across recursion so the cap is enforced on
// the whole walk, not per directory.
func walkNotesBounded(fs FS, rel string, fn func(rel string, note Note, body []byte) error, count *int, max int) error {
	if max > 0 && *count >= max {
		return errTreeTooLarge
	}
	entries, err := fs.ReadDir(rel)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		child := filepath.Join(rel, e.Name())
		if e.IsDir() {
			*count++
			if err := walkNotesBounded(fs, child, fn, count, max); err != nil {
				return err
			}
			continue
		}
		if !HasMarkdownExt(e.Name()) {
			continue
		}
		data, err := fs.ReadFile(child)
		if err != nil {
			return err
		}
		note, body, err := ParseNote(data)
		if err != nil {
			// Non-notes (missing frontmatter) are walked as markdown files but
			// surfaced with an empty Note so tree/grep can still show them.
			continue
		}
		if err := fn(child, note, body); err != nil {
			return err
		}
		*count++
		if max > 0 && *count >= max {
			return errTreeTooLarge
		}
	}
	return nil
}

// errTreeTooLarge is returned by walkNotesBounded when the node cap is hit, so
// a size-capped tool can emit a truncation marker.
var errTreeTooLarge = fmt.Errorf("tools: tree too large")

// hiddenPath reports whether any segment of the slash-separated relative path
// is a dotfile, mirroring the wiki's Visible rule.
func hiddenPath(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// noteVisible reports whether a path shows in the note tree: directories and
// markdown notes, excluding dotfiles, the root rulebook, and the reserved
// attachments subtree — the same rules the wiki tree and the dashboard use.
func noteVisible(rel string, isDir bool) bool {
	if hiddenPath(rel) || rel == "CLAUDE.md" || rel == "attachments" || strings.HasPrefix(rel, "attachments/") {
		return false
	}
	return isDir || HasMarkdownExt(rel)
}

// ListTree is the "list_tree" tool: a dirs-first recursive tree of the wiki
// with each note annotated by its frontmatter title. Output is size-capped.
type ListTree struct {
	fs       FS
	maxBytes int
}

// NewListTree returns the list_tree tool backed by fs. A non-positive maxBytes
// falls back to the default read cap.
func NewListTree(fs FS, maxBytes int) *ListTree {
	if maxBytes <= 0 {
		maxBytes = maxReadBytes
	}
	return &ListTree{fs: fs, maxBytes: maxBytes}
}

// Name implements Tool.
func (t *ListTree) Name() string { return "list_tree" }

// Description implements Tool.
func (t *ListTree) Description() string {
	return "List the full note tree, directories first, with each note's frontmatter title. Path is relative to the wiki root."
}

// Schema implements Tool.
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

// Run implements Tool.
func (t *ListTree) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
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
	if errors.Is(err, errTreeTooLarge) {
		fmt.Fprintf(&sb, "\n[tree truncated: more than %d nodes]", maxTreeNodes)
	}
	out := sb.String()
	if len(out) > t.maxBytes {
		return out[:t.maxBytes] + truncationMarker(t.maxBytes), nil
	}
	if count == 0 {
		return "", nil
	}
	return out, nil
}

// walk renders the subtree at rel into sb via writeNode, dirs first, sorted by
// name, with each note annotated by its frontmatter title. It returns
// errTreeTooLarge when the node cap is exceeded.
func (t *ListTree) walk(rel string, depth int, writeNode func(depth int, name, title string, isDir bool), count *int) error {
	if *count >= maxTreeNodes {
		return errTreeTooLarge
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
		if !noteVisible(slashChild, e.IsDir()) {
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
			if note, _, err := ParseNote(data); err == nil {
				title = note.Title
			}
		}
		writeNode(depth, e.Name(), title, false)
	}
	return nil
}
