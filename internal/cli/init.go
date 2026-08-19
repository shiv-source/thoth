package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [path]",
		Short: "Scaffold a Thoth wiki directory (default ~/.thoth/wiki)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			} else {
				var err error
				target, err = config.ExpandHome(settings.DefaultWikiPath)
				if err != nil {
					return err
				}
			}
			expanded, err := config.ExpandHome(target)
			if err != nil {
				return err
			}
			if err := wiki.ScaffoldWithOptions(expanded, wiki.ScaffoldOptions{Folders: initFolders(), GitInit: true}); err != nil {
				return err
			}
			if err := wiki.EnsureReservedDir(expanded); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, "wiki scaffolded at %s\n", expanded)
			return nil
		},
	}
}

// initFolders returns the configured scaffold folder set from the settings
// table when it exists; a missing database or key falls back to the defaults.
func initFolders() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dbPath := filepath.Join(home, ".thoth", "thoth.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		return nil
	}
	defer func() { _ = r.Close() }()
	folders, err := r.Folders()
	if err != nil {
		return nil
	}
	return folders
}
