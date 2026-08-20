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

// WatchOption tunes Watch; see WithPublisher.
type WatchOption func(*watchConfig)

// fileWatcher is the subset of fsnotify.Watcher the watch loop needs,
// declared here so tests can inject a controllable fake. The real watcher
// is wrapped in fsnotifyAdapter.
type fileWatcher interface {
	Add(name string) error
	Close() error
	Events() <-chan fsnotify.Event
	Errors() <-chan error
}

// fsnotifyAdapter adapts *fsnotify.Watcher (whose channels are fields) to
// the fileWatcher interface.
type fsnotifyAdapter struct{ w *fsnotify.Watcher }

func (a *fsnotifyAdapter) Add(name string) error         { return a.w.Add(name) }
func (a *fsnotifyAdapter) Close() error                  { return a.w.Close() }
func (a *fsnotifyAdapter) Events() <-chan fsnotify.Event { return a.w.Events }
func (a *fsnotifyAdapter) Errors() <-chan error          { return a.w.Errors }

type watchConfig struct {
	publisher func(context.Context, wiki.Changed)
	watcher   fileWatcher
}

// withWatcher injects a pre-built watcher (tests only) so watch failures on
// the Errors channel can be exercised deterministically.
func withWatcher(w fileWatcher) WatchOption {
	return func(c *watchConfig) { c.watcher = w }
}

// WithPublisher registers a hook that receives one wiki.Changed batch per
// debounce flush — only paths the wiki tree would display — plus one empty
// batch when watching starts (so a watcher restart after a wiki-path change
// also refreshes clients). A nil hook disables publishing.
func WithPublisher(p func(context.Context, wiki.Changed)) WatchOption {
	return func(c *watchConfig) { c.publisher = p }
}

// Watch keeps ix in sync with root until ctx is cancelled. Directories
// created after startup are watched and rescanned immediately, so notes
// written into them at creation time are indexed as well.
func Watch(ctx context.Context, root string, ix *Index, log *slog.Logger, opts ...WatchOption) error {
	cfg := &watchConfig{}
	for _, o := range opts {
		o(cfg)
	}

	w := cfg.watcher
	if w == nil {
		nw, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("create watcher: %w", err)
		}
		w = &fsnotifyAdapter{w: nw}
	}
	defer func() { _ = w.Close() }()

	// dirs tracks every watched directory, so a removed path can still be
	// recognized as a directory (removing one delivers no per-file events).
	dirs := map[string]struct{}{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		// Hidden directories (.git, …) are not part of the wiki: never watch
		// or descend into them, so git activity doesn't churn the watcher.
		if rel != "." && wiki.Hidden(filepath.ToSlash(rel)) {
			return filepath.SkipDir
		}
		dirs[p] = struct{}{}
		return w.Add(p)
	})
	if err != nil {
		return fmt.Errorf("watch tree: %w", err)
	}
	if cfg.publisher != nil {
		cfg.publisher(ctx, wiki.Changed{})
	}

	pending := map[string]fsnotify.Op{}
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	defer debounce.Stop()

	flush := func() {
		changes := make([]wiki.Change, 0, len(pending))
		for p, mask := range pending {
			apply(ix, root, p, log)
			if rel, err := filepath.Rel(root, p); err == nil {
				rel = filepath.ToSlash(rel)
				isDir := false
				if fi, err := os.Stat(p); err == nil {
					isDir = fi.IsDir()
				} else if _, wasDir := dirs[p]; wasDir {
					// Removed directories deliver no per-file events; the
					// watcher tracked them so removals still publish.
					isDir = true
					delete(dirs, p)
				}
				if wiki.Visible(rel, isDir) {
					changes = append(changes, wiki.Change{Op: opName(mask), Path: rel})
				}
			}
			delete(pending, p)
		}
		if len(changes) > 0 && cfg.publisher != nil {
			cfg.publisher(ctx, wiki.Changed{Changes: changes})
		}
	}

	// watchNewDir watches a directory created after startup and indexes its
	// contents. The immediate rescan covers files written before the watch
	// registration took effect, which would otherwise never fire an event.
	watchNewDir := func(p string) {
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil || (rel != "." && wiki.Hidden(filepath.ToSlash(rel))) {
			return // hidden directories are never watched
		}
		if err := w.Add(p); err != nil {
			log.Warn("index: cannot watch new directory", "path", p, "err", err)
			return
		}
		dirs[p] = struct{}{}
		if err := filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Watch every directory in the new tree, not just the top one,
			// or notes created later inside nested subdirs stay invisible.
			if d.IsDir() {
				rq, rerr := filepath.Rel(root, q)
				if rerr != nil {
					return rerr
				}
				if wiki.Hidden(filepath.ToSlash(rq)) {
					return filepath.SkipDir
				}
				if err := w.Add(q); err != nil {
					log.Warn("index: cannot watch nested directory", "path", q, "err", err)
				}
				dirs[q] = struct{}{}
				return nil
			}
			apply(ix, root, q, log)
			// The file may have been created before the new directory's
			// watch took effect, so its own event never fired: record it so
			// the pending flush also publishes it as a tree change.
			pending[q] |= fsnotify.Create
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
		case ev, ok := <-w.Events():
			if !ok {
				return nil
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				if ev.Op&fsnotify.Create != 0 {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						watchNewDir(ev.Name)
					}
				}
				pending[ev.Name] |= ev.Op
				debounce.Reset(200 * time.Millisecond)
			}
		case err, ok := <-w.Errors():
			// individual watch errors are non-fatal, but they must be heard:
			// a silent drop hides that a directory is no longer being watched
			if !ok {
				return nil
			}
			log.Warn("index: watcher error", "err", err)
		}
	}
}

func apply(ix *Index, root, p string, log *slog.Logger) {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	info, err := os.Stat(p)
	if err != nil {
		// Removing a directory delivers no per-file events, so a path that
		// no longer exists is a removed file or a removed subtree: clear it
		// from the index in one go.
		if derr := ix.DeletePrefix(rel); derr != nil {
			log.Warn("index: delete failed", "path", p, "err", derr)
		}
		return
	}
	if info.IsDir() || !wiki.Indexable(rel) {
		return // directories and hidden paths are not notes
	}
	if !wiki.IsMarkdownPath(rel) {
		// Attachment (image, script, …): index the filename only so search
		// can find it; the tree hides it (see wiki.Visible).
		if err := ix.Upsert(Note{
			Path: rel, Title: filepath.Base(rel), Kind: "file", UpdatedAt: info.ModTime(),
		}); err != nil {
			log.Warn("index: upsert failed", "path", p, "err", err)
		}
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		log.Warn("index: cannot read note", "path", p, "err", err)
		return
	}
	meta, _, err := wiki.ParseNote(b)
	if err != nil {
		for _, pr := range wiki.Validate(rel, b) {
			log.Warn("index: skipping malformed note", "path", p, "rule", pr.Rule, "msg", pr.Msg)
		}
		return
	}
	for _, pr := range wiki.Validate(rel, b) {
		log.Warn("index: save-protocol violation", "path", p, "rule", pr.Rule, "msg", pr.Msg)
	}
	if err := ix.Upsert(Note{
		Path: rel, Title: meta.Title, Kind: meta.Kind,
		Tags: meta.Tags, Body: string(b), UpdatedAt: info.ModTime(),
	}); err != nil {
		log.Warn("index: upsert failed", "path", p, "err", err)
	}
}

// opName maps an fsnotify mask onto the wiki change operations. Masks are
// OR'd across events within one debounce window; the more specific ops win.
func opName(mask fsnotify.Op) string {
	switch {
	case mask&fsnotify.Create != 0:
		return wiki.OpCreate
	case mask&fsnotify.Rename != 0:
		return wiki.OpRename
	case mask&fsnotify.Remove != 0:
		return wiki.OpRemove
	default:
		return wiki.OpWrite
	}
}
