// Package tools holds the Thoth-wiki-specific agent tools. Unlike the
// reusable library in agent/tools (generic file operations), these understand
// the wiki's note contract: frontmatter, the folder-to-type rule, the
// scaffolded layout (todos/TODO.md, inbox/, knowledge/memory.md). They run
// over the same agent/tools.FS seam, so every path stays bounded to the wiki
// root, and they share the note contract with the wiki via internal/wiki
// (ParseNote/FormatNote) — the single source, never a fork.
package tools

import (
	"path/filepath"
	"strings"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/wiki"
)

// DefaultTodosPath is the scaffolded location of the wiki's TODO list.
const DefaultTodosPath = "todos/TODO.md"

// DefaultInboxDir is the scaffolded inbox folder.
const DefaultInboxDir = "inbox"

// DefaultMemoryPath is the default memory note for the remember tool.
const DefaultMemoryPath = "knowledge/memory.md"

// noteTypeFor derives the default frontmatter type for a note path: the
// top-level folder's type per the wiki's mechanical rule (meetings/x.md ->
// meeting). A path at the root has no folder, so its type stays empty.
func noteTypeFor(rel string) string {
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		return wiki.NoteType(rel[:i])
	}
	return ""
}

// walkNotes walks the note tree rooted at rel (relative to the FS root) and
// calls fn for every markdown note with its parsed metadata and body. It
// follows the wiki's Visible rules — skipping dotfiles, the root rulebook and
// the attachments subtree — so the agent and the dashboard agree on what a
// note is. Notes that fail to parse are skipped, not fatal. Paths are
// slash-separated wiki-relative, matching the wiki and the index.
func walkNotes(fs agenttools.FS, rel string, fn func(rel string, meta wiki.NoteMeta, body []byte) error) error {
	return agenttools.WalkFiles(fs, rel, func(fileRel string) error {
		slashRel := filepath.ToSlash(fileRel)
		if !wiki.Visible(slashRel, false) {
			return nil
		}
		data, err := fs.ReadFile(slashRel)
		if err != nil {
			return err
		}
		meta, body, err := wiki.ParseNote(data)
		if err != nil {
			return nil
		}
		return fn(slashRel, meta, body)
	})
}
