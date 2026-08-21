package agent

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

// wikiFS adapts the live wiki directory to the tools.FS seam. Every operation
// resolves its relative name through wiki.SafePath against the wiki's current
// root, so reads, writes, mkdir and list can never escape the wiki root — even
// through a symlink — and always agree with the root the rulebook is read
// from, even after a settings wiki-path change.
type wikiFS struct {
	wiki *wiki.Wiki
}

// ReadFile implements tools.FS.
func (f wikiFS) ReadFile(name string) ([]byte, error) {
	p, err := wiki.SafePath(f.wiki.Root(), name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return data, nil
}

// WriteFile implements tools.FS. The write is atomic — temp file plus rename
// in the target directory — so a failed write never leaves a partial note.
func (f wikiFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	p, err := wiki.SafePath(f.wiki.Root(), name)
	if err != nil {
		return err
	}
	if err := agenttools.AtomicWrite(p, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// MkdirAll implements tools.FS.
func (f wikiFS) MkdirAll(path string, perm fs.FileMode) error {
	p, err := wiki.SafePath(f.wiki.Root(), path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p, perm); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

// ReadDir implements tools.FS.
func (f wikiFS) ReadDir(name string) ([]fs.DirEntry, error) {
	p, err := wiki.SafePath(f.wiki.Root(), name)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", name, err)
	}
	return entries, nil
}

// indexSearch adapts the FTS5 index to the search tool's SearchFunc. Note
// paths are already wiki-relative, matching the tool's result contract.
func indexSearch(ix *index.Index) agenttools.SearchFunc {
	return func(ctx context.Context, query string, limit int) ([]agenttools.Result, error) {
		res, err := ix.Search(query, limit)
		if err != nil {
			return nil, err
		}
		out := make([]agenttools.Result, 0, len(res))
		for _, r := range res {
			out = append(out, agenttools.Result{Path: r.Path, Snippet: r.Snippet})
		}
		return out, nil
	}
}

// registry builds the wiki-bounded tool registry: read_file, write_file and
// list over a wikiFS bound to the live wiki root (so a settings wiki-path
// change moves the tools with the rulebook), plus search over the FTS index
// when one is supplied. It is registered once and read-only afterwards, so
// concurrent Start calls may share it.
func registry(w *wiki.Wiki, ix *index.Index) (*agenttools.Registry, error) {
	fsys := wikiFS{wiki: w}
	reg := agenttools.NewRegistry()
	for _, t := range []agenttools.Tool{
		agenttools.NewReadFile(fsys, 0),
		agenttools.NewWriteFile(fsys),
		agenttools.NewList(fsys),
	} {
		if err := reg.Register(t); err != nil {
			return nil, err
		}
	}
	if ix != nil {
		if err := reg.Register(agenttools.NewSearch(indexSearch(ix), 0)); err != nil {
			return nil, err
		}
	}
	return reg, nil
}
