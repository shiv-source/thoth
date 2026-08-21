package agent

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

// wikiFS adapts the live wiki directory to the tools.FS seam. Every operation
// resolves its relative name through wiki.SafePath against the wiki's current
// root, so reads, writes, mkdir, list, stat, remove and rename can never
// escape the wiki root — even through a symlink — and always agree with the
// root the rulebook is read from, even after a settings wiki-path change.
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

// Stat implements tools.FS.
func (f wikiFS) Stat(name string) (fs.FileInfo, error) {
	p, err := wiki.SafePath(f.wiki.Root(), name)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", name, err)
	}
	return fi, nil
}

// Remove implements tools.FS.
func (f wikiFS) Remove(name string) error {
	p, err := wiki.SafePath(f.wiki.Root(), name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	return nil
}

// Rename implements tools.FS. Both names resolve through SafePath against the
// same live root, so a move can never escape the wiki.
func (f wikiFS) Rename(oldPath, newPath string) error {
	op, err := wiki.SafePath(f.wiki.Root(), oldPath)
	if err != nil {
		return err
	}
	np, err := wiki.SafePath(f.wiki.Root(), newPath)
	if err != nil {
		return err
	}
	if err := os.Rename(op, np); err != nil {
		return fmt.Errorf("rename %s: %w", oldPath, err)
	}
	return nil
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

// RegistryOptions tunes the tool registry a host builds. Wiki is required;
// Index adds the FTS search tool when non-nil. The path-configurable tools
// (todos/inbox/memory) use the scaffolded wiki defaults unless overridden.
type RegistryOptions struct {
	Wiki  *wiki.Wiki
	Index *index.Index
	// TodosPath overrides the get_todos/update_todos path (default
	// todos/TODO.md); InboxDir overrides get_inbox/file_inbox (default inbox);
	// MemoryPath overrides remember (default knowledge/memory.md).
	TodosPath  string
	InboxDir   string
	MemoryPath string
}

// registry builds the wiki-bounded tool registry: the file/note/edit/search
// tools over a wikiFS bound to the live wiki root (so a settings wiki-path
// change moves the tools with the rulebook), plus search over the FTS index
// when one is supplied. It is registered once and read-only afterwards, so
// concurrent Start calls may share it.
func registry(opts RegistryOptions) (*agenttools.Registry, error) {
	if opts.Wiki == nil {
		return nil, errors.New("agent: registry requires a wiki")
	}
	fsys := wikiFS{wiki: opts.Wiki}
	reg := agenttools.NewRegistry()
	now := time.Now
	for _, t := range []agenttools.Tool{
		agenttools.NewReadFile(fsys, 0),
		agenttools.NewWriteFile(fsys),
		agenttools.NewList(fsys),
		agenttools.NewGetTime(now),
		agenttools.NewWriteNote(fsys, now),
		agenttools.NewReadNote(fsys, 0),
		agenttools.NewEditFile(fsys),
		agenttools.NewAppendFile(fsys),
		agenttools.NewRenameFile(fsys),
		agenttools.NewDeleteFile(fsys),
		agenttools.NewListTree(fsys, 0),
		agenttools.NewGrep(fsys, 0),
		agenttools.NewListRecent(fsys, 0),
		agenttools.NewSearchByTag(fsys, 0),
		agenttools.NewGetTodos(fsys, opts.TodosPath),
		agenttools.NewUpdateTodos(fsys, opts.TodosPath),
		agenttools.NewGetInbox(fsys, opts.InboxDir),
		agenttools.NewFileInbox(fsys, opts.InboxDir),
		agenttools.NewRemember(fsys, opts.MemoryPath, now),
	} {
		if err := reg.Register(t); err != nil {
			return nil, err
		}
	}
	if opts.Index != nil {
		if err := reg.Register(agenttools.NewSearch(indexSearch(opts.Index), 0)); err != nil {
			return nil, err
		}
	}
	return reg, nil
}
