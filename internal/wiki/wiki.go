package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Children []Node `json:"children"`
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
// skipping dotfiles.
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
		if e.Name()[0] == '.' {
			continue // .git, .gitkeep, dotfiles
		}
		childRel := filepath.Join(rel, e.Name())
		n := Node{Name: e.Name(), Path: filepath.ToSlash(childRel), IsDir: e.IsDir()}
		if e.IsDir() {
			n.Children, err = tree(base, childRel)
			if err != nil {
				return nil, err
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
