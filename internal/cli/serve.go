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
	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

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
	// that name, falling back to the shared api_key and the provider's
	// default endpoint. Read at boot like the model — a change applies on
	// next start.
	providerName, err := modelProvider(st, model)
	if err != nil {
		return err
	}
	apiKey, baseURL, err := stg.ProviderConfig(providerName)
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

	gh, err := github.OpenRepo(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = gh.Close() }()

	// The native agent host drives every turn — no CLI subprocess exists
	// anywhere in the chat path. Model, provider config, wiki (tools +
	// rulebook), store (history) and index (search) are all read at boot.
	// The git tools follow the live wiki root and are guarded by the sync
	// switch: commit/push run only after the user opted into sync, with the
	// stored GitHub connection supplying identity and push credentials.
	ac, err := agent.New(model, apiKey, w, st, ix,
		agent.WithProviderConfig(providerName, baseURL),
		agent.WithLogger(log),
		agent.WithFolders(scaffoldFolders(stg, log)),
		agent.WithGitOptions(gitToolOptions(w, stg, gh)),
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
		GitHub:          &github.Service{Client: github.New(http.DefaultClient), Repo: gh},
		Settings:        stg,
		DataDir:         dir,
		Version:         Version(),
		Dev:             dev,
		Commit:          commit,
		DefaultWikiPath: config.ToTilde(defaultWikiPath(dev, dir)),
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
	// A pre-existing but unversioned wiki is git-inited on startup (the agent's
	// git_commit/git_push tools also auto-init), so the git tools always have a
	// repo to act on. Idempotent.
	if err := wiki.EnsureGitRepo(wikiPath); err != nil {
		return nil, err
	}
	return w, nil
}

// gitToolOptions wires the agent's git tools to the live wiki root and the
// stored GitHub connection. The guard gates the mutating tools on the sync
// switch; identity and push credentials come from the github_auth row, read
// lazily per call so a token is never held and a connection change applies
// without restart.
func gitToolOptions(w *wiki.Wiki, stg *settings.Repo, gh *github.Repo) agenttools.GitOptions {
	return agenttools.GitOptions{
		RepoPath: func() string { return w.Root() },
		Guard: func() error {
			on, err := stg.SyncEnabled()
			if err != nil {
				return err
			}
			if !on {
				return errors.New("git sync is disabled — enable it in Settings to commit or push")
			}
			return nil
		},
		Auth: func() (agenttools.GitAuth, error) {
			a, ok, err := gh.Get()
			if err != nil {
				return agenttools.GitAuth{}, err
			}
			if !ok {
				return agenttools.GitAuth{}, errors.New("no GitHub connection — connect one in Settings to push")
			}
			return agenttools.GitAuth{Username: a.Username, Token: a.Token}, nil
		},
		Identity: func() (agenttools.GitIdentity, error) {
			a, ok, err := gh.Get()
			if err != nil {
				return agenttools.GitIdentity{}, err
			}
			if !ok {
				return agenttools.GitIdentity{}, errors.New("no GitHub connection — connect one in Settings to commit")
			}
			name := a.DisplayName
			if name == "" {
				name = a.Username
			}
			return agenttools.GitIdentity{Name: name, Email: a.Email}, nil
		},
	}
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
// result leaves the shared api_key as the credential and lets agent.New fall
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
