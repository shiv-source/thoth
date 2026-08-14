package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/api"
	"github.com/shiv-source/thoth/internal/claude"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

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
	cfgPath := filepath.Join(dir, "config.toml")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	w, err := ensureWiki(cfg.WikiPath, log)
	if err != nil {
		return err
	}
	st, ix, err := openStores(filepath.Join(dir, "thoth.db"), w.Root, log)
	if err != nil {
		return err
	}
	defer st.Close()
	defer ix.Close()

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := index.Watch(ctx, w.Root, ix, log); err != nil {
			log.Error("watcher stopped", "err", err)
		}
	}()

	e := api.New(api.Deps{
		Log:             log,
		Config:          &cfg,
		ConfigPath:      cfgPath,
		Store:           st,
		Claude:          claude.New(resolveClaudeBin(cfg, log), w.Root, claude.WithPermissionMode(cfg.PermissionMode), claude.WithModel(cfg.Model)),
		Wiki:            w,
		Index:           ix,
		OnSettingsSaved: onSettingsSaved(log, w, ix),
	})

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	log.Info("thoth listening", "addr", addr, "wiki", w.Root)
	return serveUntilShutdown(e, addr, ctx)
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

// loadConfig reads the config at path, persisting defaults on first run.
func loadConfig(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, err
	}
	if err := config.Save(path, cfg); err != nil {
		return config.Config{}, err // persist defaults on first run
	}
	return cfg, nil
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

// openStores opens the chat store and the note index and rebuilds the index
// from wikiPath.
func openStores(dbPath, wikiPath string, log *slog.Logger) (*store.Store, *index.Index, error) {
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}
	ix, err := index.Open(dbPath)
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	if err := ix.Rebuild(wikiPath, log); err != nil {
		ix.Close()
		st.Close()
		return nil, nil, err
	}
	return st, ix, nil
}

// resolveClaudeBin returns the configured claude binary, falling back to the
// one on PATH and finally to a bare "claude" that will fail loudly at chat
// time.
func resolveClaudeBin(cfg config.Config, log *slog.Logger) string {
	if cfg.ClaudeBin != "" {
		return cfg.ClaudeBin
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p
	}
	log.Warn("claude CLI not found on PATH — chat will fail until configured", "setting", "claude_bin")
	return "claude"
}

// onSettingsSaved rebuilds the index when the wiki path changes.
func onSettingsSaved(log *slog.Logger, w *wiki.Wiki, ix *index.Index) func(config.Config) error {
	return func(c config.Config) error {
		newPath, err := config.ExpandHome(c.WikiPath)
		if err != nil {
			return err
		}
		if newPath == w.Root {
			return nil
		}
		log.Info("wiki path changed, rebuilding index", "path", newPath)
		w.Root = newPath
		if !w.Exists() {
			if err := wiki.Scaffold(newPath); err != nil {
				return err
			}
		}
		return ix.Rebuild(newPath, log)
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
