package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/api"
	"github.com/shiv-source/thoth/internal/claude"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

// rootHolder is the single source of truth for the current wiki path. The
// claude client reads it on every Start and the settings callback writes it,
// so a wiki-path change takes effect for new turns without restarting.
type rootHolder struct {
	mu   sync.RWMutex
	root string
}

func newRootHolder(root string) *rootHolder { return &rootHolder{root: root} }

func (r *rootHolder) get() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.root
}

func (r *rootHolder) set(root string) {
	r.mu.Lock()
	r.root = root
	r.mu.Unlock()
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the Thoth server",
		RunE:  runServe,
	}
}

func runServe(cmd *cobra.Command, _ []string) error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	dir, err := thothDir()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "thoth.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	if err := st.EnsureMetadata(); err != nil {
		return err
	}
	stg, err := settings.OpenRepo(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = stg.Close() }()

	// The wiki path lives in the settings table; failing to read it aborts
	// boot rather than silently falling back to a default (a fallback would
	// scaffold a second wiki while the user's data lives elsewhere).
	wikiPath, found, err := stg.Setting(settings.KeyWikiPath)
	if err != nil {
		return err
	}
	if !found || wikiPath == "" {
		wikiPath = settings.DefaultWikiPath
	}
	// The model setting enforces --model on every CLI spawn; empty keeps
	// the CLI's own default. Read at boot — a change applies on next start.
	model, _, err := stg.Setting(settings.KeyModel)
	if err != nil {
		return err
	}
	w, err := ensureWiki(wikiPath, log)
	if err != nil {
		return err
	}
	ix, err := openIndex(dbPath, w.Root, log)
	if err != nil {
		return err
	}
	defer func() { _ = ix.Close() }()

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The watcher tracks the current wiki root; a settings change cancels the
	// old watcher and starts a fresh one under the new root.
	var watcherMu sync.Mutex
	var watcherCancel context.CancelFunc
	startWatcher := func(root string) {
		watcherMu.Lock()
		if watcherCancel != nil {
			watcherCancel()
		}
		wctx, cancel := context.WithCancel(ctx)
		watcherCancel = cancel
		watcherMu.Unlock()
		go func() {
			if err := index.Watch(wctx, root, ix, log); err != nil && wctx.Err() == nil {
				log.Error("watcher stopped", "err", err)
			}
		}()
	}
	startWatcher(w.Root)

	gh, err := github.OpenRepo(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = gh.Close() }()

	root := newRootHolder(w.Root)
	// One long-lived CLI process per conversation: the first turn of each
	// conversation pays the CLI boot, later turns reuse the process. Close
	// guarantees no CLI process outlives the server.
	pc := claude.NewPersistent(resolveClaudeBin(log), w.Root, claude.WithDirProvider(root.get), claude.WithModel(model), claude.WithDebugStream(filepath.Join(dir, "stream-dump.json")))
	defer func() { _ = pc.Close() }()
	e := api.New(api.Deps{
		Log:             log,
		Store:           st,
		Claude:          pc,
		GitHub:          &github.Service{Client: github.New(http.DefaultClient), Repo: gh},
		Settings:        stg,
		DataDir:         dir,
		Version:         Version(),
		Wiki:            w,
		Index:           ix,
		OnSettingsSaved: onSettingsSaved(log, root, w, ix, startWatcher, pc.Flush),
		Ctx:             ctx,
	})

	host, port := config.DefaultHost, config.DefaultPort
	// The banner owns its trailing newline — Fprint, not Fprintln, so the
	// panel ends flush with the next prompt line.
	fmt.Fprint(os.Stderr, startupBanner(Version(), host, port, w.Root, isTerminal(os.Stderr)))
	return serveUntilShutdown(e, net.JoinHostPort(host, strconv.Itoa(port)), ctx)
}

// thothDir returns ~/.thoth, creating it if needed.
func thothDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home: %w", err)
	}
	dir := filepath.Join(home, ".thoth")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create ~/.thoth: %w", err)
	}
	return dir, nil
}

// ensureWiki returns a wiki for path, scaffolding it if it does not exist.
func ensureWiki(path string, log *slog.Logger) (*wiki.Wiki, error) {
	wikiPath, err := config.ExpandHome(path)
	if err != nil {
		return nil, err
	}
	w := wiki.New(wikiPath)
	if !w.Exists() {
		if err := wiki.Scaffold(wikiPath); err != nil {
			return nil, err
		}
		log.Info("scaffolded wiki", "path", wikiPath)
	}
	return w, nil
}

// openIndex opens the note index and syncs it from wikiPath.
func openIndex(dbPath, wikiPath string, log *slog.Logger) (*index.Index, error) {
	ix, err := index.Open(dbPath)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	if err := ix.Sync(wikiPath, log); err != nil {
		_ = ix.Close()
		return nil, err
	}
	log.Info("index synced", "path", wikiPath, "dur", time.Since(start))
	return ix, nil
}

// resolveClaudeBin returns the claude binary from PATH, falling back to a
// bare "claude" that will fail loudly at chat time.
func resolveClaudeBin(log *slog.Logger) string {
	if p, err := exec.LookPath("claude"); err == nil {
		return p
	}
	log.Warn("claude CLI not found on PATH — chat will fail until it is installed")
	return "claude"
}

// onSettingsSaved rebuilds the index and switches the live wiki root when
// the wiki path changes. Nothing is mutated until every step that can fail
// has succeeded: scaffold, then rebuild, and only then the root swap, the
// watcher restart, and the pooled-CLI flush (idle processes die now, a busy
// one is evicted at its turn's end — same semantics as the per-Start cwd).
func onSettingsSaved(log *slog.Logger, root *rootHolder, w *wiki.Wiki, ix *index.Index, startWatcher func(string), flush func()) func(string) error {
	return func(wikiPath string) error {
		newPath, err := config.ExpandHome(wikiPath)
		if err != nil {
			return err
		}
		if newPath == root.get() {
			return nil // already current (e.g. a retry after a failed save)
		}
		log.Info("wiki path changed, syncing index", "path", newPath)
		// Check the new path itself: w still points at the old root until
		// every fallible step below has succeeded.
		if !wiki.New(newPath).Exists() {
			if err := wiki.Scaffold(newPath); err != nil {
				return err
			}
		}
		if err := ix.Sync(newPath, log); err != nil {
			return err
		}
		// All fallible steps done: commit the new root atomically-ish.
		root.set(newPath)
		w.Root = newPath
		startWatcher(newPath)
		flush()
		return nil
	}
}

// serveUntilShutdown runs e until it fails or ctx is done (Ctrl+C / SIGTERM),
// then shuts down cleanly. Echo does not install its own signal handlers.
func serveUntilShutdown(e *echo.Echo, addr string, ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- e.Start(addr) }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return e.Shutdown(context.Background())
	}
}
