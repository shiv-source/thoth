package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/doctor"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

// errUnhealthy is returned by thoth doctor when at least one check fails.
// main prints it and exits with status 1.
var errUnhealthy = errors.New("doctor: one or more checks failed")

// doctorRunner runs the checks (via the shared internal/doctor package) and, with
// --fix, the repairs. The paths the repair pass needs are captured here
// because the shared checks resolve them internally.
type doctorRunner struct {
	log             *slog.Logger
	fixes           []string
	dbPath          string
	wikiPath        string
	wikiResolved    bool
	wikiFolders     []string
	providerBaseURL string
}

func newDoctorCmd() *cobra.Command {
	var dir string
	var fix bool
	var providerBaseURL string
	cmd := &cobra.Command{
		Use:          "doctor",
		Short:        "Check the health of the Thoth installation",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(dir, fix, providerBaseURL)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "repair what can be fixed automatically")
	cmd.Flags().StringVar(&dir, "dir", "", "thoth directory to check (default ~/.thoth)")
	cmd.Flags().StringVar(&providerBaseURL, "provider-base-url", "", "provider base URL the provider check probes (default: the provider's public endpoint)")
	_ = cmd.Flags().MarkHidden("dir")
	_ = cmd.Flags().MarkHidden("provider-base-url")
	return cmd
}

func runDoctor(dir string, fix bool, providerBaseURL string) error {
	d := &doctorRunner{log: slog.New(slog.NewTextHandler(os.Stderr, nil)), providerBaseURL: providerBaseURL}
	checks := d.checks(dir, fix)
	for _, f := range d.fixes {
		_, _ = fmt.Fprintf(os.Stdout, "fixed: %s\n", f)
	}
	allOK := true
	for _, c := range checks {
		mark := "✓ "
		if !c.OK {
			mark = "✗ "
			allOK = false
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s%s: %s\n", mark, c.Name, c.Message)
	}
	if !allOK {
		return errUnhealthy
	}
	return nil
}

// checks runs every check, then applies --fix repairs and re-runs the suite so
// the report reflects the repaired state.
func (d *doctorRunner) checks(dir string, fix bool) []doctor.Check {
	results := d.runChecks(dir)
	if fix && d.repair(results) {
		results = d.runChecks(dir)
	}
	return results
}

// runChecks runs the shared check suite and captures the resolved paths the
// repair pass needs. The wiki path comes from the settings table; a missing
// database falls back to the default, so repairs target the default wiki.
func (d *doctorRunner) runChecks(dir string) []doctor.Check {
	thDir, err := resolveThothDir(dir)
	if err != nil {
		return []doctor.Check{{Name: "home", OK: false, Message: err.Error()}}
	}
	d.dbPath = filepath.Join(thDir, "thoth.db")
	wikiPath := settings.DefaultWikiPath
	if fileExists(d.dbPath) {
		if r, err := settings.OpenRepo(d.dbPath); err == nil {
			if value, found, err := r.Setting(settings.KeyWikiPath); err == nil && found && value != "" {
				wikiPath = value
			}
			if folders, err := r.Folders(); err == nil {
				d.wikiFolders = folders
			}
			_ = r.Close()
		}
	}
	expanded, err := config.ExpandHome(wikiPath)
	d.wikiResolved = err == nil
	if err != nil {
		d.wikiPath = wikiPath
	} else {
		d.wikiPath = expanded
	}
	return doctor.Run(context.Background(), doctor.Options{Dir: dir, Log: d.log, BaseURL: d.providerBaseURL})
}

// resolveThothDir returns dir, or ~/.thoth when dir is empty. It mirrors the
// resolution inside doctor.Run so the repair pass knows the real paths.
func resolveThothDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve home directory: %v", err)
	}
	return filepath.Join(home, ".thoth"), nil
}

// repair fixes what --fix is allowed to touch: a missing wiki and an
// out-of-sync index. It never touches provider connectivity or api keys. It
// returns whether any fix was applied.
func (d *doctorRunner) repair(results []doctor.Check) bool {
	fixed := false
	if d.wikiResolved && failed(results, "wiki") {
		if err := wiki.ScaffoldWithOptions(d.wikiPath, wiki.ScaffoldOptions{Folders: d.wikiFolders, GitInit: true}); err != nil {
			d.fixes = append(d.fixes, fmt.Sprintf("wiki: could not scaffold %s: %v", d.wikiPath, err))
		} else {
			d.fixes = append(d.fixes, fmt.Sprintf("wiki: scaffolded %s", d.wikiPath))
			fixed = true
		}
	}
	if failed(results, "index") {
		if err := d.syncIndex(); err != nil {
			d.fixes = append(d.fixes, fmt.Sprintf("index: sync failed: %v", err))
		} else {
			d.fixes = append(d.fixes, fmt.Sprintf("index: synced from %s", d.wikiPath))
			fixed = true
		}
	}
	return fixed
}

func (d *doctorRunner) syncIndex() error {
	ix, err := index.Open(d.dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = ix.Close() }()
	return ix.Sync(d.wikiPath, d.log)
}

func failed(results []doctor.Check, name string) bool {
	for _, r := range results {
		if r.Name == name && !r.OK {
			return true
		}
	}
	return false
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
