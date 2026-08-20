package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-warehouse/events"
	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/api"
	"github.com/shiv-source/thoth/internal/assets"
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
	var dev bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the Thoth server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, dev)
		},
	}
	cmd.Flags().BoolVar(&dev, "dev", false, "run on the dev port (8334) with isolated data in ~/.thoth/dev — leaves 8333 free for a running instance")
	return cmd
}

func runServe(cmd *cobra.Command, dev bool) error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// The event bus carries wiki change batches from the index watcher to
	// the API layer, which pushes them to connected clients. Close drains
	// subscribers before shutdown; publish failures after Close are
	// expected (ErrClosed) and swallowed by the watcher's publisher.
	bus := events.New(events.WithLogger(log))
	defer func() {
		bus.Close()
		_ = bus.Wait(context.Background())
	}()

	dir, err := thothDir(dev)
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
	if err := ensureModels(st); err != nil {
		return err
	}

	// The wiki path lives in the settings table; failing to read it aborts
	// boot rather than silently falling back to a default (a fallback would
	// scaffold a second wiki while the user's data lives elsewhere).
	wikiPath, err := settleWikiPath(stg, dev, dir)
	if err != nil {
		return err
	}
	// The model setting enforces --model on every CLI spawn; empty keeps
	// the CLI's own default. Read at boot — a change applies on next start.
	model, _, err := stg.Setting(settings.KeyModel)
	if err != nil {
		return err
	}
	// The api key setting becomes ANTHROPIC_API_KEY on every CLI spawn;
	// empty inherits the server's own environment. Read at boot like the
	// model — a change applies on next start.
	apiKey, _, err := stg.Setting(settings.KeyAPIKey)
	if err != nil {
		return err
	}
	w, err := ensureWiki(wikiPath, stg, log)
	if err != nil {
		return err
	}
	ix, err := openIndex(dbPath, w.Root, log)
	if err != nil {
		return err
	}
	defer func() { _ = ix.Close() }()

	root := newRootHolder(w.Root)
	// One long-lived CLI process per conversation: the first turn of each
	// conversation pays the CLI boot, later turns reuse the process. Close
	// guarantees no CLI process outlives the server.
	pc := claude.NewPersistent(resolveClaudeBin(log), w.Root, claude.WithDirProvider(root.get), claude.WithModel(model), claude.WithAPIKey(apiKey), claude.WithDebugStream(filepath.Join(dir, "stream-dump.json")))
	defer func() { _ = pc.Close() }()

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
		go watchWiki(wctx, root, ix, log, bus, pc.Flush)
	}
	startWatcher(w.Root)

	gh, err := github.OpenRepo(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = gh.Close() }()

	// Eagerly spawn the pooled process for the most recently active
	// conversation so resuming it after a restart skips the CLI boot (~1.6 s
	// of spawn + init). The pool's idle-eviction timer reaps it if the user
	// never resumes that conversation.
	prewarmPool(log, st, pc)
	// The dev banner shows the commit the dev server runs from; prod has no
	// banner, so the lookup is dev-only.
	commit := ""
	if dev {
		commit = devCommit("")
	}
	e := api.New(api.Deps{
		Log:             log,
		Store:           st,
		Claude:          pc,
		GitHub:          &github.Service{Client: github.New(http.DefaultClient), Repo: gh},
		Settings:        stg,
		DataDir:         dir,
		Version:         Version(),
		Dev:             dev,
		Commit:          commit,
		DefaultWikiPath: config.ToTilde(defaultWikiPath(dev, dir)),
		Wiki:            w,
		Index:           ix,
		OnSettingsSaved: onSettingsSaved(log, stg, root, w, ix, startWatcher, pc.Flush),
		Ctx:             ctx,
		// When the user leaves the chat page (socket closed or tab hidden),
		// flush idle pooled CLI processes RelaxTimeout later instead of letting
		// them idle for their full 10-minute timer; a busy process finishes its
		// turn and is evicted.
		RelaxTimeout: chatRelaxTimeout,
		OnChatAway:   pc.Flush,
		Events:       bus,
	})

	host, port := config.DefaultHost, servePort(dev)
	// The banner owns its trailing newline — Fprint, not Fprintln, so the
	// panel ends flush with the next prompt line.
	fmt.Fprint(os.Stderr, startupBanner(Version(), host, port, w.Root, isTerminal(os.Stderr)))
	return serveUntilShutdown(e, net.JoinHostPort(host, strconv.Itoa(port)), ctx)
}

// servePort returns the listen port: DevPort for serve --dev (make dev),
// DefaultPort otherwise.
func servePort(dev bool) int {
	if dev {
		return config.DevPort
	}
	return config.DefaultPort
}

// watchWiki runs the index watcher for a wiki root until ctx is cancelled:
// tree changes publish onto the event bus (serve forwards them to clients as
// wiki_changed frames), and a rulebook (wiki-root CLAUDE.md) edit calls
// onRulebook — serve wires it to the pool's Flush, so idle pooled CLI
// processes die now and the next turn of every conversation respawns under
// the new rules. A watcher that stops of its own accord is logged, not fatal.
func watchWiki(ctx context.Context, root string, ix *index.Index, log *slog.Logger, bus *events.Bus, onRulebook func()) {
	publish := index.WithPublisher(func(pctx context.Context, changed wiki.Changed) {
		if err := bus.Publish(pctx, changed); err != nil && !errors.Is(err, events.ErrClosed) {
			log.Warn("publish wiki change", "err", err)
		}
	})
	if err := index.Watch(ctx, root, ix, log, publish, index.WithRulebookChange(onRulebook)); err != nil && ctx.Err() == nil {
		log.Error("watcher stopped", "err", err)
	}
}

// chatRelaxTimeout is how long idle pooled CLI processes survive after the
// user leaves the chat page (socket closed or tab hidden). Shorter than the
// per-process idle timer, so an away user does not hold heavyweight Claude
// processes for minutes; resuming after that pays the CLI boot again.
const chatRelaxTimeout = time.Minute

// thothDir returns the server's data dir: ~/.thoth, or ~/.thoth/dev when
// serve --dev keeps its database and wiki isolated from production data.
func thothDir(dev bool) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home: %w", err)
	}
	dir := filepath.Join(home, ".thoth")
	if dev {
		dir = filepath.Join(dir, "dev")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// defaultWikiPath is the fallback wiki location when the settings table has
// none: prod keeps the shared default, dev resolves inside the dev data dir
// so a fresh dev database can never fall back onto the production wiki.
func defaultWikiPath(dev bool, dir string) string {
	if dev {
		return filepath.Join(dir, "wiki")
	}
	return settings.DefaultWikiPath
}

// resolveWikiPath picks the wiki root: the stored setting wins, except a
// missing/empty value and — in dev — the seeded prod default. The 0007 seed
// stores the prod default in every fresh database, so without the override a
// fresh dev database would follow the seed onto the production wiki.
func resolveWikiPath(dev bool, dir, stored string, found bool) string {
	if !found || stored == "" || (dev && stored == settings.DefaultWikiPath) {
		return defaultWikiPath(dev, dir)
	}
	return stored
}

// settleWikiPath reads and resolves the wiki root, and persists the
// resolution when it differs from the stored value — in dev this rewrites
// the seeded prod default to the dev wiki, so every reader (the settings
// UI, the next boot) agrees with the wiki the server actually serves.
// Returns the resolved path in expanded form, ready for ensureWiki.
func settleWikiPath(stg *settings.Repo, dev bool, dir string) (string, error) {
	stored, found, err := stg.Setting(settings.KeyWikiPath)
	if err != nil {
		return "", err
	}
	resolved := resolveWikiPath(dev, dir, stored, found)
	if resolved != stored {
		if err := stg.SetSetting(settings.KeyWikiPath, config.ToTilde(resolved)); err != nil {
			return "", err
		}
	}
	return resolved, nil
}

// ensureModels seeds llm_models from assets/models.json whenever the table
// is empty, so every startup self-heals an empty registry (fresh install,
// deleted database, or a user who removed every model). A table with rows —
// even just one user-added model — is never overwritten. models.json stays
// the single source for the built-in list.
func ensureModels(st *store.Store) error {
	models, err := st.ListModels()
	if err != nil {
		return err
	}
	if len(models) > 0 {
		return nil
	}
	opts, err := assets.ModelOptions()
	if err != nil {
		return err
	}
	for _, o := range opts {
		if _, err := st.CreateModel(o.Value, o.Name, o.Tag, o.Provider); err != nil {
			return fmt.Errorf("seed model %s: %w", o.Value, err)
		}
	}
	return nil
}

// scaffoldFolders returns the configured scaffold folder set, or nil (the
// defaults) when unset or unreadable.
func scaffoldFolders(stg *settings.Repo, log *slog.Logger) []string {
	folders, err := stg.Folders()
	if err != nil {
		log.Warn("read wiki_folders", "err", err)
		return nil
	}
	return folders
}

// ensureWiki returns a wiki for path, scaffolding it if it does not exist.
func ensureWiki(path string, stg *settings.Repo, log *slog.Logger) (*wiki.Wiki, error) {
	wikiPath, err := config.ExpandHome(path)
	if err != nil {
		return nil, err
	}
	w := wiki.New(wikiPath)
	if !w.Exists() {
		if err := wiki.ScaffoldWithOptions(wikiPath, wiki.ScaffoldOptions{Folders: scaffoldFolders(stg, log), GitInit: true}); err != nil {
			return nil, err
		}
		log.Info("scaffolded wiki", "path", wikiPath)
	}
	// The attachments directory is reserved and app-managed: recreate it on
	// every startup if it went missing.
	if err := wiki.EnsureReservedDir(wikiPath); err != nil {
		return nil, err
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

// devCommit returns the full commit id of the checkout in dir (the server's
// working directory under `make dev`); empty when dir is not a git repo or
// git is unavailable, so the banner degrades to no commit.
func devCommit(dir string) string {
	if dir == "" {
		dir = "."
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

// prewarmStore is the slice of the store prewarmPool needs, kept as a small
// consumer-side interface so every defensive failure branch is testable.
type prewarmStore interface {
	RecentConversation() (store.Conversation, error)
	ClaudeSessionID(convID string) (string, error)
}

// prewarmPool spawns the pooled CLI process for the most recently created
// conversation, so resuming it after a restart skips the CLI boot. It warms
// the exact session a turn on that conversation would use (store.ClaudeSessionID).
// Best-effort: an empty database or any lookup/spawn failure only logs a
// warning — the first send on that session spawns normally.
func prewarmPool(log *slog.Logger, st prewarmStore, pc *claude.PersistentClient) {
	recent, err := st.RecentConversation()
	if err != nil {
		log.Warn("prewarm: list conversations", "err", err)
		return
	}
	if recent.ID == "" {
		return // no conversations: nothing to pre-warm
	}
	sid, err := st.ClaudeSessionID(recent.ID)
	if err != nil {
		log.Warn("prewarm: read session id", "err", err)
		return
	}
	if err := pc.Warm(sid); err != nil {
		log.Warn("prewarm: spawn claude", "session_id", sid, "err", err)
		return
	}
	log.Info("prewarmed claude process", "conversation_id", recent.ID, "session_id", sid)
}

// onSettingsSaved rebuilds the index and switches the live wiki root when
// the wiki path changes. Nothing is mutated until every step that can fail
// has succeeded: scaffold, then rebuild, and only then the root swap, the
// watcher restart, and the pooled-CLI flush (idle processes die now, a busy
// one is evicted at its turn's end — same semantics as the per-Start cwd).
func onSettingsSaved(log *slog.Logger, stg *settings.Repo, root *rootHolder, w *wiki.Wiki, ix *index.Index, startWatcher func(string), flush func()) func(string) error {
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
			if err := wiki.ScaffoldWithOptions(newPath, wiki.ScaffoldOptions{Folders: scaffoldFolders(stg, log), GitInit: true}); err != nil {
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
