package main

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
		RunE: func(cmd *cobra.Command, args []string) error {
			log := slog.New(slog.NewTextHandler(os.Stderr, nil))

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("find home: %w", err)
			}
			thothDir := filepath.Join(home, ".thoth")
			if err := os.MkdirAll(thothDir, 0o755); err != nil {
				return fmt.Errorf("create ~/.thoth: %w", err)
			}

			cfgPath := filepath.Join(thothDir, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			if err := config.Save(cfgPath, cfg); err != nil {
				return err // persist defaults on first run
			}

			wikiPath, err := config.ExpandHome(cfg.WikiPath)
			if err != nil {
				return err
			}
			w := wiki.Open(wikiPath)
			if !w.Exists() {
				if err := wiki.Scaffold(wikiPath); err != nil {
					return err
				}
				log.Info("scaffolded wiki", "path", wikiPath)
			}

			dbPath := filepath.Join(thothDir, "thoth.db")
			st, err := store.Open(dbPath)
			if err != nil {
				return err
			}
			defer st.Close()
			ix, err := index.Open(dbPath)
			if err != nil {
				return err
			}
			defer ix.Close()
			if err := ix.Rebuild(wikiPath, log); err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			go func() {
				if err := index.Watch(ctx, wikiPath, ix, log); err != nil {
					log.Error("watcher stopped", "err", err)
				}
			}()

			bin := cfg.ClaudeBin
			if bin == "" {
				if p, err := exec.LookPath("claude"); err == nil {
					bin = p
				} else {
					log.Warn("claude CLI not found on PATH — chat will fail until configured", "setting", "claude_bin")
					bin = "claude"
				}
			}
			client := claude.New(bin, wikiPath,
				claude.WithPermissionMode(cfg.PermissionMode),
				claude.WithModel(cfg.Model))

			e := api.New(api.Deps{
				Log:        log,
				Config:     &cfg,
				ConfigPath: cfgPath,
				Store:      st,
				Claude:     client,
				Wiki:       w,
				Index:      ix,
				OnSettingsSaved: func(c config.Config) error {
					newPath, err := config.ExpandHome(c.WikiPath)
					if err != nil {
						return err
					}
					if newPath != w.Root {
						log.Info("wiki path changed, rebuilding index", "path", newPath)
						w.Root = newPath
						if !w.Exists() {
							if err := wiki.Scaffold(newPath); err != nil {
								return err
							}
						}
						return ix.Rebuild(newPath, log)
					}
					return nil
				},
			})

			addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
			log.Info("thoth listening", "addr", addr, "wiki", wikiPath)
			// Run the server until it fails or the signal context is done
			// (Ctrl+C / SIGTERM), then shut down cleanly. Echo does not
			// install its own signal handlers.
			errCh := make(chan error, 1)
			go func() { errCh <- e.Start(addr) }()
			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				return e.Shutdown(context.Background())
			}
		},
	}
}
