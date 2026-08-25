package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-warehouse/events"
	"github.com/labstack/echo/v4"
	agentgit "github.com/shiv-source/thoth/agent/git"
	agenttools "github.com/shiv-source/thoth/agent/tools"
	agent "github.com/shiv-source/thoth/internal/agent"
	"github.com/shiv-source/thoth/internal/api"
	"github.com/shiv-source/thoth/internal/assets"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/retention"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	syncsvc "github.com/shiv-source/thoth/internal/sync"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var dev bool
	var noAPIDocs bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the Thoth server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, dev, noAPIDocs)
		},
	}
	cmd.Flags().BoolVar(&dev, "dev", false, "run on the dev port (8334) with isolated data in ~/.thoth/dev — leaves 8333 free for a running instance")
	cmd.Flags().BoolVar(&noAPIDocs, "no-api-docs", false, "exclude API docs even in --dev mode")
	return cmd
}

func runServe(cmd *cobra.Command, dev bool, noAPIDocs bool) error {
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
	if err := ensureSyncProviders(st); err != nil {
		return err
	}
	if err := ensureLocalBackup(st); err != nil {
		return err
	}

	// The wiki path lives in the settings table; failing to read it aborts
	// boot rather than silently falling back to a default (a fallback would
	// scaffold a second wiki while the user's data lives elsewhere).
	wikiPath, err := settleWikiPath(stg, dev, dir)
	if err != nil {
		return err
	}
	// The model setting selects the provider model for every turn; empty
	// falls back to the first seeded claude model (the family the CLI itself
	// defaulted to). Read at boot — a change applies on next start.
	model, _, err := stg.Setting(settings.KeyModel)
	if err != nil {
		return err
	}
	if model == "" {
		model = defaultModel(st)
	}
	// The selected model's llm_models row names its provider; the matching
	// per-provider config (its own api key + base URL override) resolves from
	// that name, with an empty base URL meaning the provider's default
	// endpoint. Read at boot like the model — a change applies on next start.
	providerName, err := modelProvider(st, model)
	if err != nil {
		return err
	}
	apiKey, baseURL, err := stg.ProviderConfig(providerName)
	if err != nil {
		return err
	}
	// The provider's custom request headers (e.g. Portkey's x-portkey-*
	// routing headers) are read at boot like the key and base URL, and sent
	// on every request the agent makes through that provider.
	customHeaders, err := stg.ProviderHeaders(providerName)
	if err != nil {
		return err
	}
	w, err := ensureWiki(wikiPath, stg, log)
	if err != nil {
		return err
	}
	ix, err := openIndex(dbPath, w.Root(), log)
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
			publish := index.WithPublisher(func(pctx context.Context, changed wiki.Changed) {
				if err := bus.Publish(pctx, changed); err != nil && !errors.Is(err, events.ErrClosed) {
					log.Warn("publish wiki change", "err", err)
				}
			})
			if err := index.Watch(wctx, root, ix, log, publish); err != nil && wctx.Err() == nil {
				log.Error("watcher stopped", "err", err)
			}
		}()
	}
	startWatcher(w.Root())

	// The multi-provider sync engine reads the same thoth.db: connections
	// (credentials + targets), the provider catalog, and the local backup
	// (both seeded above) live on sync_providers/sync_connections.
	syncSvc := syncsvc.NewService(st, http.DefaultClient)

	// The auto-sync scheduler pushes enabled connections whose configured
	// interval has elapsed. It shares the Service.Push path with the API, and
	// reports each result on the event bus so the API layer can surface a
	// notification. It stops on server shutdown (ctx), so no goroutine
	// outlives serve.
	syncScheduler := syncsvc.NewScheduler(syncSvc, w.Root(), log, func(r syncsvc.Result) {
		if err := bus.Publish(ctx, r); err != nil && !errors.Is(err, events.ErrClosed) {
			log.Warn("publish sync result", "err", err)
		}
	})
	go syncScheduler.Start(ctx)

	// The chat-retention scheduler deletes conversations older than the
	// configured window (default 7 days) every hour. It reads the window from
	// the settings table on each sweep, so a change in Settings → General
	// applies within the hour without a restart. It stops on ctx, so no
	// goroutine outlives serve.
	retentionScheduler := retention.NewScheduler(st, stg, log)
	go retentionScheduler.Start(ctx)

	// The native agent host drives every turn — no CLI subprocess exists
	// anywhere in the chat path. Model, provider config, wiki (tools +
	// rulebook), store (history) and index (search) are all read at boot;
	// the context_injection setting opts the host into pre-searching the
	// wiki into each turn's first prompt.
	// The git tools follow the live wiki root and are guarded by the active
	// git sync connection: commit/push run only when an enabled git-kind
	// connection exists, with that connection supplying identity and push
	// credentials. The conversation-memory tools (list/get/search_conversations)
	// are wired to the store, and system_health to the same doctor checks the
	// CLI runs.
	contextInjection, err := stg.ContextInjection()
	if err != nil {
		return err
	}
	ac, err := agent.New(model, apiKey, w, st, ix,
		agent.WithProviderConfig(providerName, baseURL),
		agent.WithCustomHeaders(customHeaders),
		agent.WithLogger(log),
		agent.WithFolders(scaffoldFolders(stg, log)),
		agent.WithGitOptions(gitToolOptions(w, stg, syncSvc)),
		agent.WithHealthFunc(agent.DoctorHealth(dir)),
		agent.WithContextInjection(contextInjection),
	)
	if err != nil {
		return err
	}
	// The dev banner shows the commit the dev server runs from; prod has no
	// banner, so the lookup is dev-only.
	commit := ""
	if dev {
		commit = devCommit("")
	}
	e := api.New(api.Deps{
		Log:             log,
		Store:           st,
		Claude:          ac,
		Sync:            syncSvc,
		Settings:        stg,
		DataDir:         dir,
		Version:         Version(),
		Dev:             dev,
		Commit:          commit,
		DefaultWikiPath: config.ToTilde(defaultWikiPath(dev, dir)),
		ServeAPIDocs:    apiDocsEnabled(dev, noAPIDocs),
		Wiki:            w,
		Index:           ix,
		OnSettingsSaved: onSettingsSaved(log, stg, w, ix, startWatcher),
		Ctx:             ctx,
		Events:          bus,
	})

	host, port := config.DefaultHost, servePort(dev)
	// The banner owns its trailing newline — Fprint, not Fprintln, so the
	// panel ends flush with the next prompt line.
	fmt.Fprint(os.Stderr, startupBanner(Version(), host, port, w.Root(), isTerminal(os.Stderr)))
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

// apiDocsEnabled reports whether the /swagger.json route is served: only
// serve --dev exposes it, and --no-api-docs opts back out of that one dev
// convenience. serve without --dev is always off — the flag is a no-op there.
func apiDocsEnabled(dev, noAPIDocs bool) bool {
	return dev && !noAPIDocs
}

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

// ensureModels seeds llm_models from assets/llm-providers.json whenever the
// table is empty, so every startup self-heals an empty registry (fresh
// install, deleted database, or a user who removed every model). A table with
// rows — even just one user-added model — is never overwritten.
// llm-providers.json stays the single source for the built-in list.
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
		// Providers are first-class rows now: ensure the model's provider
		// exists (creating it with empty credentials), then attach the model
		// to it.
		p, err := st.EnsureProvider(o.Provider)
		if err != nil {
			return fmt.Errorf("seed provider %s: %w", o.Provider, err)
		}
		if _, err := st.CreateModel(o.Value, o.Name, o.Tag, p.ID); err != nil {
			return fmt.Errorf("seed model %s: %w", o.Value, err)
		}
	}
	return nil
}

// ensureSyncProviders seeds sync_providers from assets/sync-providers.json
// whenever the table is empty — the ensureModels self-heal: a fresh or wiped
// database gets the built-ins back, a table with rows is never touched.
// Migration 0012 seeds the same four rows on a fresh database so the
// github_auth cutover has its provider to attach to; the JSON stays the
// single source for the built-in list (a store test pins the two in sync).
func ensureSyncProviders(st *store.Store) error {
	providers, err := st.ListSyncProviders()
	if err != nil {
		return err
	}
	if len(providers) > 0 {
		return nil
	}
	opts, err := assets.SyncProviderOptions()
	if err != nil {
		return err
	}
	for _, o := range opts {
		if _, err := st.EnsureSyncProvider(o.Slug, o.Name, o.Driver, o.BaseURL, o.Protected); err != nil {
			return fmt.Errorf("seed sync provider %s: %w", o.Slug, err)
		}
	}
	return nil
}

// ensureLocalBackup guarantees the first-class local backup connection on
// every boot: one protected connection under the local provider, created with
// no folder configured (the user picks it in Settings). Idempotent.
func ensureLocalBackup(st *store.Store) error {
	if _, err := st.EnsureLocalBackup(); err != nil {
		return err
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
	// A pre-existing but unversioned wiki is git-inited on startup (the agent's
	// git_commit/git_push tools also auto-init), so the git tools always have a
	// repo to act on. Idempotent.
	if err := wiki.EnsureGitRepo(wikiPath); err != nil {
		return nil, err
	}
	return w, nil
}

// gitToolOptions wires the agent's git tools to the live wiki root and the
// active git-kind sync connection. The guard gates the mutating tools on an
// enabled git connection existing; identity and push credentials come from
// that connection, read lazily per call so a token is never held and a
// connection change applies without restart.
func gitToolOptions(w *wiki.Wiki, stg *settings.Repo, svc *syncsvc.Service) agenttools.GitOptions {
	return agenttools.GitOptions{
		RepoPath: func() string { return w.Root() },
		Guard: func() error {
			_, ok, err := activeGitConnection(svc, stg)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("no enabled git sync connection — enable one in Settings to commit or push")
			}
			return nil
		},
		Auth: func() (agenttools.GitAuth, error) {
			c, ok, err := activeGitConnection(svc, stg)
			if err != nil {
				return agenttools.GitAuth{}, err
			}
			if !ok {
				return agenttools.GitAuth{}, errors.New("no git connection — connect one in Settings to push")
			}
			cfg, err := syncsvc.DecodeConfig(c.Config)
			if err != nil {
				return agenttools.GitAuth{}, err
			}
			ident, err := syncsvc.DecodeIdentity(c.Identity)
			if err != nil {
				return agenttools.GitAuth{}, err
			}
			username := ident.Username
			if username == "" {
				username = "oauth2"
			}
			return agenttools.GitAuth{Username: username, Token: cfg["token"]}, nil
		},
		Identity: func() (agenttools.GitIdentity, error) {
			c, ok, err := activeGitConnection(svc, stg)
			if err != nil {
				return agenttools.GitIdentity{}, err
			}
			if !ok {
				return agenttools.GitIdentity{}, errors.New("no git connection — connect one in Settings to commit")
			}
			ident, err := syncsvc.DecodeIdentity(c.Identity)
			if err != nil {
				return agenttools.GitIdentity{}, err
			}
			name := ident.DisplayName
			if name == "" {
				name = ident.Username
			}
			return agenttools.GitIdentity{Name: name, Email: ident.Email}, nil
		},
	}
}

// activeGitConnection returns the agent-tools' sync connection: the one named
// by sync_active_connection when it is an enabled git-kind connection,
// otherwise the first enabled git-kind connection. ok is false when none
// exists.
func activeGitConnection(svc *syncsvc.Service, stg *settings.Repo) (store.Connection, bool, error) {
	if svc == nil || svc.Store == nil {
		return store.Connection{}, false, nil
	}
	active, found, err := stg.Setting(settings.KeyActiveConnection)
	if err != nil {
		return store.Connection{}, false, err
	}
	if found && active != "" {
		if id, err := strconv.ParseInt(active, 10, 64); err == nil {
			if c, err := svc.Store.Connection(id); err == nil && c.Enabled && isGitConnection(svc, c) {
				return c, true, nil
			}
		}
	}
	conns, err := svc.Store.ListConnections()
	if err != nil {
		return store.Connection{}, false, err
	}
	for _, c := range conns {
		if c.Enabled && isGitConnection(svc, c) {
			return c, true, nil
		}
	}
	return store.Connection{}, false, nil
}

// isGitConnection reports whether a connection's provider resolves to a
// git-kind driver.
func isGitConnection(svc *syncsvc.Service, c store.Connection) bool {
	p, err := svc.Store.SyncProvider(c.ProviderID)
	if err != nil {
		return false
	}
	d, err := svc.Driver(p)
	if err != nil {
		return false
	}
	return d.Kind() == syncsvc.KindGit
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
// working directory under `make dev`); empty when dir is not a git repo, so
// the banner degrades to no commit.
func devCommit(dir string) string {
	if dir == "" {
		dir = "."
	}
	repo, err := agentgit.Open(dir)
	if err != nil {
		return ""
	}
	head, err := repo.Head()
	if err != nil {
		return ""
	}
	return head
}

// defaultModel returns the model a fresh install runs on when the settings
// model key is unset: the first seeded claude-family model — the family the
// CLI itself defaulted to, so an unset model keeps working after the cutover.
// An empty return (no claude model in the registry) lets agent.New surface
// its own "model is required" error.
func defaultModel(st *store.Store) string {
	models, err := st.ListModels()
	if err != nil {
		return ""
	}
	for _, m := range models {
		if strings.HasPrefix(m.Value, "claude-") {
			return m.Value
		}
	}
	return ""
}

// modelProvider returns the llm_models row's provider label for a model
// value, or "" when the value is empty or not in the registry. An empty
// result leaves the agent without a per-provider key and lets agent.New fall
// back to its model-id routing, preserving the pre-registry behavior.
func modelProvider(st *store.Store, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	models, err := st.ListModels()
	if err != nil {
		return "", err
	}
	for _, m := range models {
		if m.Value == value {
			return m.Provider, nil
		}
	}
	return "", nil
}

// onSettingsSaved rebuilds the index and switches the live wiki root when
// the wiki path changes. Nothing is mutated until every step that can fail
// has succeeded: scaffold, then rebuild, and only then the root swap and the
// watcher restart. The rulebook the agent reads each turn follows the new
// root immediately; the tool registry's root is fixed at boot.
func onSettingsSaved(log *slog.Logger, stg *settings.Repo, w *wiki.Wiki, ix *index.Index, startWatcher func(string)) func(string) error {
	return func(wikiPath string) error {
		newPath, err := config.ExpandHome(wikiPath)
		if err != nil {
			return err
		}
		if newPath == w.Root() {
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
		w.SetRoot(newPath)
		startWatcher(newPath)
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
