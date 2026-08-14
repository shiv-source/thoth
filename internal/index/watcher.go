package index

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shiv-source/thoth/internal/wiki"
)

// Watch keeps ix in sync with root until ctx is cancelled. Directories
// created after startup are watched and rescanned immediately, so notes
// written into them at creation time are indexed as well.
func Watch(ctx context.Context, root string, ix *Index, log *slog.Logger) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer func() { _ = w.Close() }()

	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(p)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("watch tree: %w", err)
	}

	pending := map[string]bool{}
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	defer debounce.Stop()

	flush := func() {
		for p := range pending {
			apply(ix, root, p, log)
			delete(pending, p)
		}
	}

	// watchNewDir watches a directory created after startup and indexes its
	// contents. The immediate rescan covers files written before the watch
	// registration took effect, which would otherwise never fire an event.
	watchNewDir := func(p string) {
		if err := w.Add(p); err != nil {
			log.Warn("index: cannot watch new directory", "path", p, "err", err)
			return
		}
		if err := filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Watch every directory in the new tree, not just the top one,
			// or notes created later inside nested subdirs stay invisible.
			if d.IsDir() {
				if err := w.Add(q); err != nil {
					log.Warn("index: cannot watch nested directory", "path", q, "err", err)
				}
				return nil
			}
			if filepath.Ext(q) != ".md" {
				return nil
			}
			apply(ix, root, q, log)
			return nil
		}); err != nil {
			log.Warn("index: cannot rescan new directory", "path", p, "err", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-debounce.C:
			flush()
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				if ev.Op&fsnotify.Create != 0 {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						watchNewDir(ev.Name)
					}
				}
				pending[ev.Name] = true
				debounce.Reset(200 * time.Millisecond)
			}
		case <-w.Errors:
			// individual watch errors are non-fatal; keep serving
		}
	}
}

func apply(ix *Index, root, p string, log *slog.Logger) {
	if filepath.Ext(p) != ".md" {
		// Removing a directory delivers no per-file events, so a path that
		// no longer exists and is not a note is a removed subtree: clear it
		// from the index in one go.
		if _, err := os.Stat(p); err != nil {
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return
			}
			if err := ix.DeletePrefix(filepath.ToSlash(rel)); err != nil {
				log.Warn("index: delete prefix failed", "path", p, "err", err)
			}
		}
		return
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return
		}
		if err := ix.Delete(filepath.ToSlash(rel)); err != nil {
			log.Warn("index: delete failed", "path", p, "err", err)
		}
		return
	}
	if err != nil {
		log.Warn("index: cannot read note", "path", p, "err", err)
		return
	}
	meta, _, err := wiki.ParseNote(b)
	if err != nil {
		log.Warn("index: skipping malformed note", "path", p, "err", err)
		return
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return
	}
	info, err := os.Stat(p)
	if err != nil {
		return
	}
	if err := ix.Upsert(Note{
		Path: filepath.ToSlash(rel), Title: meta.Title, Kind: meta.Kind,
		Tags: meta.Tags, Body: string(b), UpdatedAt: info.ModTime(),
	}); err != nil {
		log.Warn("index: upsert failed", "path", p, "err", err)
	}
}
