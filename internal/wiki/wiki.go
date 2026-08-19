package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// Visible reports whether a wiki-relative, slash-separated path appears in
// the tree: everything except dotfiles (any hidden path segment) and the
// root rulebook. Tree() relies on it, and so does the index watcher when
// deciding which changes are worth an event.
func Visible(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") {
			return false
		}
	}
	return rel != "CLAUDE.md"
}

type Wiki struct {
	Root string
}

func New(root string) *Wiki { return &Wiki{Root: root} }

func (w *Wiki) Exists() bool {
	fi, err := os.Stat(w.Root)
	return err == nil && fi.IsDir()
}

func (w *Wiki) Read(rel string) ([]byte, error) {
	p, err := SafePath(w.Root, rel)
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
// skipping dotfiles. Directories that cannot be read (permissions, …) are
// kept as nodes carrying an Error instead of failing the whole walk; only
// an unreadable root aborts.
func (w *Wiki) Tree() ([]Node, error) {
	return tree(w.Root, "")
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
		if !Visible(filepath.ToSlash(childRel)) {
			continue // dotfiles and the root rulebook are not notes
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
