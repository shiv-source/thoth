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
	log          *slog.Logger
	fixes        []string
	cfgPath      string
	dbPath       string
	wikiPath     string
	wikiResolved bool
}

func newDoctorCmd() *cobra.Command {
	var dir string
	var fix bool
	cmd := &cobra.Command{
		Use:          "doctor",
		Short:        "Check the health of the Thoth installation",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(dir, fix)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "repair what can be fixed automatically")
	cmd.Flags().StringVar(&dir, "dir", "", "thoth directory to check (default ~/.thoth)")
	_ = cmd.Flags().MarkHidden("dir")
	return cmd
}

func runDoctor(dir string, fix bool) error {
	d := &doctorRunner{log: slog.New(slog.NewTextHandler(os.Stderr, nil))}
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
// repair pass needs. The config fallback mirrors internal/doctor: a missing or
// unparsable config yields defaults, so repairs target the default wiki path.
func (d *doctorRunner) runChecks(dir string) []doctor.Check {
	thDir, err := resolveThothDir(dir)
	if err != nil {
		return []doctor.Check{{Name: "home", OK: false, Message: err.Error()}}
	}
	d.cfgPath = filepath.Join(thDir, "config.toml")
	d.dbPath = filepath.Join(thDir, "thoth.db")
	cfg, _ := config.Load(d.cfgPath)
	expanded, err := config.ExpandHome(cfg.WikiPath)
	d.wikiResolved = err == nil
	if err != nil {
		d.wikiPath = cfg.WikiPath
	} else {
		d.wikiPath = expanded
	}
	return doctor.Run(context.Background(), dir, d.log)
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

// repair fixes what --fix is allowed to touch: a missing config file, a
// missing wiki, and an out-of-sync index. It never touches the claude login.
// It returns whether any fix was applied.
func (d *doctorRunner) repair(results []doctor.Check) bool {
	fixed := false
	if !fileExists(d.cfgPath) {
		if err := config.Save(d.cfgPath, config.Default()); err != nil {
			d.fixes = append(d.fixes, fmt.Sprintf("config: could not write %s: %v", d.cfgPath, err))
		} else {
			d.fixes = append(d.fixes, fmt.Sprintf("config: wrote default config to %s", d.cfgPath))
			fixed = true
		}
	}
	if d.wikiResolved && failed(results, "wiki") {
		if err := wiki.Scaffold(d.wikiPath); err != nil {
			d.fixes = append(d.fixes, fmt.Sprintf("wiki: could not scaffold %s: %v", d.wikiPath, err))
		} else {
			d.fixes = append(d.fixes, fmt.Sprintf("wiki: scaffolded %s", d.wikiPath))
			fixed = true
		}
	}
	if failed(results, "index") {
		if err := d.rebuildIndex(); err != nil {
			d.fixes = append(d.fixes, fmt.Sprintf("index: rebuild failed: %v", err))
		} else {
			d.fixes = append(d.fixes, fmt.Sprintf("index: rebuilt from %s", d.wikiPath))
			fixed = true
		}
	}
	return fixed
}

func (d *doctorRunner) rebuildIndex() error {
	ix, err := index.Open(d.dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = ix.Close() }()
	return ix.Rebuild(d.wikiPath, d.log)
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
