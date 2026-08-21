package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Children []Node `json:"children"`
	// Error is set when the directory itself exists but could not be read
	// (e.g. permissions); its children are omitted. The node stays in the
	// tree so one locked folder never hides every other note.
	Error string `json:"error,omitempty"`
}

// Change is a single filesystem change inside the wiki directory. Op is one
// of the Op* constants; Path is wiki-relative and slash-separated.
type Change struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

// Changed is the batch of changes the index watcher reports per debounce
// flush, pushed to the UI so it can refetch the wiki tree.
type Changed struct {
	Changes []Change `json:"changes,omitempty"`
}

// Filesystem change operations.
const (
	OpCreate = "create"
	OpWrite  = "write"
	OpRemove = "remove"
	OpRename = "rename"
)

// IsMarkdownPath reports whether p names a markdown note — .md or
// .markdown, case-insensitive. The tree and the index share it so both agree
// on what a note is.
func IsMarkdownPath(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

// IsImagePath reports whether p names a previewable image attachment — .png,
// .jpg, .jpeg, .gif, .svg, or .webp, case-insensitive. The /api/notes handler
// uses it to serve images inline (raw bytes) while every other attachment is
// served as a download; the dashboard mirrors it so the preview/download
// decision matches the wire format.
func IsImagePath(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return true
	}
	return false
}

// Hidden reports whether any segment of a wiki-relative, slash-separated path
// is a dotfile.
func Hidden(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// isReservedPath reports whether rel is inside the reserved attachments
// subtree: the top-level AttachmentsDir directory and everything under it.
func isReservedPath(rel string) bool {
	return rel == AttachmentsDir || strings.HasPrefix(rel, AttachmentsDir+"/")
}

// Visible reports whether a wiki-relative, slash-separated path appears in
// the tree: directories and markdown notes, excluding dotfiles (any hidden
// path segment), the root rulebook, and the reserved attachments subtree.
// Attachments remain searchable by filename (see Indexable). Tree() relies
// on Visible, and so does the index watcher when deciding which changes are
// worth an event.
func Visible(rel string, isDir bool) bool {
	if Hidden(rel) || rel == "CLAUDE.md" || isReservedPath(rel) {
		return false
	}
	return isDir || IsMarkdownPath(rel)
}

// Indexable reports whether a file path is indexed: everything except
// dotfiles and the root rulebook. Markdown notes are parsed for their
// frontmatter; other files (attachments, anywhere in the wiki) are indexed by
// filename only so search can find them. The index sync, the watcher, and the
// doctor's index check all share this so the tree and index agree on what is
// indexed.
func Indexable(rel string) bool {
	return !Hidden(rel) && rel != "CLAUDE.md"
}

type Wiki struct {
	mu   sync.RWMutex
	root string
}

func New(root string) *Wiki { return &Wiki{root: root} }

// Root returns the current wiki root directory. It may change when the
// settings wiki path is updated (SetRoot), so callers read it once per
// filesystem operation to keep rulebook and tools on the same root.
func (w *Wiki) Root() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.root
}

// SetRoot switches the wiki root directory, e.g. after a settings change.
// It is safe to call concurrently with Root: readers see either the old or
// the new root, never a torn value.
func (w *Wiki) SetRoot(root string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.root = root
}

func (w *Wiki) Exists() bool {
	fi, err := os.Stat(w.Root())
	return err == nil && fi.IsDir()
}

func (w *Wiki) Read(rel string) ([]byte, error) {
	p, err := SafePath(w.Root(), rel)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	return b, nil
}

// Tree returns the full directory tree with dirs first, sorted by name,
// showing directories and markdown notes only — dotfiles and non-markdown
// attachments (see Visible) are skipped. Directories that cannot be read
// (permissions, …) are kept as nodes carrying an Error instead of failing
// the whole walk; only an unreadable root aborts.
func (w *Wiki) Tree() ([]Node, error) {
	return tree(w.Root(), "")
}

func tree(base, rel string) ([]Node, error) {
	dir := filepath.Join(base, rel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", rel, err)
	}
	nodes := make([]Node, 0, len(entries))
	for _, e := range entries {
		childRel := filepath.Join(rel, e.Name())
		if !Visible(filepath.ToSlash(childRel), e.IsDir()) {
			continue // dotfiles, the root rulebook, and attachments are not notes
		}
		n := Node{Name: e.Name(), Path: filepath.ToSlash(childRel), IsDir: e.IsDir()}
		if e.IsDir() {
			children, err := tree(base, childRel)
			if err != nil {
				n.Error = err.Error()
			} else {
				n.Children = children
			}
		}
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return nodes[i].Name < nodes[j].Name
	})
	return nodes, nil
}
