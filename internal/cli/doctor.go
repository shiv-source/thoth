package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
	"github.com/spf13/cobra"
)

// errUnhealthy is returned by thoth doctor when at least one check fails.
// main prints it and exits with status 1.
var errUnhealthy = errors.New("doctor: one or more checks failed")

const (
	claudeCheckTimeout = 2 * time.Second
	portDialTimeout    = 300 * time.Millisecond
)

type checkResult struct {
	name string
	ok   bool
	msg  string
}

// doctor runs the checks and (with --fix) the repairs. State shared between
// the check pass and the repair pass lives on the struct.
type doctor struct {
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
	d := &doctor{log: slog.New(slog.NewTextHandler(os.Stderr, nil))}
	checks := d.checks(dir, fix)
	for _, f := range d.fixes {
		fmt.Fprintf(os.Stdout, "fixed: %s\n", f)
	}
	allOK := true
	for _, c := range checks {
		mark := "✓ "
		if !c.ok {
			mark = "✗ "
			allOK = false
		}
		fmt.Fprintf(os.Stdout, "%s%s: %s\n", mark, c.name, c.msg)
	}
	if !allOK {
		return errUnhealthy
	}
	return nil
}

// checks runs every check, then applies --fix repairs and re-runs the suite so
// the report reflects the repaired state.
func (d *doctor) checks(dir string, fix bool) []checkResult {
	results := d.runChecks(dir)
	if fix && d.repair(results) {
		results = d.runChecks(dir)
	}
	return results
}

// runChecks runs all six checks in order; every check runs even when an
// earlier one fails.
func (d *doctor) runChecks(dir string) []checkResult {
	thDir := dir
	if thDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return []checkResult{{name: "home", ok: false, msg: fmt.Sprintf("cannot resolve home directory: %v", err)}}
		}
		thDir = filepath.Join(home, ".thoth")
	}
	d.cfgPath = filepath.Join(thDir, "config.toml")
	d.dbPath = filepath.Join(thDir, "thoth.db")

	results := make([]checkResult, 0, 7)

	// 1. config
	cfg, res := checkConfig(d.cfgPath)
	results = append(results, res)

	// 2. wiki
	expanded, err := config.ExpandHome(cfg.WikiPath)
	d.wikiResolved = err == nil
	if err != nil {
		results = append(results, checkResult{name: "wiki", ok: false, msg: fmt.Sprintf("cannot expand wiki path %q: %v", cfg.WikiPath, err)})
		expanded = cfg.WikiPath
	}
	d.wikiPath = expanded
	results = append(results, checkWiki(expanded))

	// 3. claude CLI
	results = append(results, checkClaude(cfg)...)

	// 4. database
	results = append(results, checkDatabase(d.dbPath))

	// 5. index sync
	results = append(results, checkIndex(d.dbPath, expanded))

	// 6. port
	results = append(results, checkPort(cfg))

	return results
}

// repair fixes what --fix is allowed to touch: a missing config file, a
// missing wiki, and an out-of-sync index. It never touches the claude login.
// It returns whether any fix was applied.
func (d *doctor) repair(results []checkResult) bool {
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

func (d *doctor) rebuildIndex() error {
	ix, err := index.Open(d.dbPath)
	if err != nil {
		return err
	}
	defer ix.Close()
	return ix.Rebuild(d.wikiPath, d.log)
}

func failed(results []checkResult, name string) bool {
	for _, r := range results {
		if r.name == name && !r.ok {
			return true
		}
	}
	return false
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// checkConfig verifies that ~/.thoth/config.toml exists and parses.
func checkConfig(path string) (config.Config, checkResult) {
	if !fileExists(path) {
		return config.Default(), checkResult{name: "config", ok: false, msg: fmt.Sprintf("%s is missing — run thoth doctor --fix to create a default config", path)}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Default(), checkResult{name: "config", ok: false, msg: fmt.Sprintf("%s cannot be parsed: %v — fix the syntax or recreate the file", path, err)}
	}
	return cfg, checkResult{name: "config", ok: true, msg: fmt.Sprintf("%s parses", path)}
}

// checkWiki verifies that the wiki directory exists with the scaffold folders
// and CLAUDE.md.
func checkWiki(root string) checkResult {
	if !isDir(root) {
		return checkResult{name: "wiki", ok: false, msg: fmt.Sprintf("%s does not exist — run thoth doctor --fix to scaffold it", root)}
	}
	var missing []string
	for _, f := range wiki.Folders() {
		if !isDir(filepath.Join(root, f)) {
			missing = append(missing, f)
		}
	}
	if !fileExists(filepath.Join(root, "CLAUDE.md")) {
		missing = append(missing, "CLAUDE.md")
	}
	if len(missing) > 0 {
		return checkResult{name: "wiki", ok: false, msg: fmt.Sprintf("%s is missing %s — run thoth doctor --fix to repair it", root, strings.Join(missing, ", "))}
	}
	return checkResult{name: "wiki", ok: true, msg: fmt.Sprintf("%s exists with the 8 scaffold folders and CLAUDE.md", root)}
}

// checkClaude resolves the claude binary (config.ClaudeBin or "claude" on
// PATH), reports its version, and probes the login state. The login result is
// only reported when the binary was found.
func checkClaude(cfg config.Config) []checkResult {
	bin, notFound := resolveDoctorClaudeBin(cfg)
	if notFound != "" {
		return []checkResult{{name: "claude", ok: false, msg: notFound}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), claudeCheckTimeout)
	defer cancel()
	ver, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return []checkResult{{name: "claude", ok: false, msg: fmt.Sprintf("%s --version failed: %v — is the CLI installed correctly?", bin, err)}}
	}
	results := []checkResult{{name: "claude", ok: true, msg: fmt.Sprintf("%s (version %s)", bin, strings.TrimSpace(string(ver)))}}
	ctx, cancel = context.WithTimeout(context.Background(), claudeCheckTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, bin, "auth", "status").Run(); err != nil {
		results = append(results, checkResult{name: "claude login", ok: false, msg: "login status unknown (claude auth status did not exit 0)"})
	} else {
		results = append(results, checkResult{name: "claude login", ok: true, msg: "login confirmed (claude auth status exited 0)"})
	}
	return results
}

// resolveDoctorClaudeBin returns the claude binary to probe: config.ClaudeBin
// when configured, otherwise "claude" from PATH. The second return value is a
// failure message, or "" when a binary was resolved.
func resolveDoctorClaudeBin(cfg config.Config) (string, string) {
	if bin := cfg.ClaudeBin; bin != "" {
		if isExecutable(bin) {
			return bin, ""
		}
		return "", fmt.Sprintf("configured claude_bin %q not found — fix claude_bin in the config", bin)
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p, ""
	}
	return "", "claude CLI not found on PATH — install it or set claude_bin in the config"
}

func isExecutable(bin string) bool {
	if strings.ContainsRune(bin, os.PathSeparator) {
		fi, err := os.Stat(bin)
		return err == nil && !fi.IsDir()
	}
	_, err := exec.LookPath(bin)
	return err == nil
}

// checkDatabase verifies that thoth.db exists, opens in WAL mode, and has the
// notes and notes_fts tables.
func checkDatabase(path string) checkResult {
	if !fileExists(path) {
		return checkResult{name: "database", ok: false, msg: fmt.Sprintf("%s does not exist — start thoth serve to create it", path)}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return checkResult{name: "database", ok: false, msg: fmt.Sprintf("%s cannot be opened: %v", path, err)}
	}
	defer db.Close()
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode;`).Scan(&mode); err != nil {
		return checkResult{name: "database", ok: false, msg: fmt.Sprintf("%s is not a usable sqlite database: %v", path, err)}
	}
	if !strings.EqualFold(mode, "wal") {
		return checkResult{name: "database", ok: false, msg: fmt.Sprintf("%s journal mode is %q, want \"wal\"", path, mode)}
	}
	var tables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('notes', 'notes_fts')`).Scan(&tables); err != nil {
		return checkResult{name: "database", ok: false, msg: fmt.Sprintf("cannot inspect %s: %v", path, err)}
	}
	if tables != 2 {
		return checkResult{name: "database", ok: false, msg: fmt.Sprintf("%s is missing the notes/notes_fts tables — run thoth doctor --fix", path)}
	}
	return checkResult{name: "database", ok: true, msg: fmt.Sprintf("%s opens in WAL with notes + notes_fts tables", path)}
}

// checkIndex compares the number of notes in the index against a rescan of the
// wiki's .md files with valid frontmatter.
func checkIndex(dbPath, wikiRoot string) checkResult {
	onDisk, err := countNotes(wikiRoot)
	if err != nil {
		return checkResult{name: "index", ok: false, msg: fmt.Sprintf("cannot scan %s: %v — run thoth doctor --fix", wikiRoot, err)}
	}
	if !fileExists(dbPath) {
		return checkResult{name: "index", ok: false, msg: fmt.Sprintf("%s does not exist — run thoth doctor --fix to create and populate it", dbPath)}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return checkResult{name: "index", ok: false, msg: fmt.Sprintf("cannot open %s: %v", dbPath, err)}
	}
	defer db.Close()
	var indexed int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&indexed); err != nil {
		return checkResult{name: "index", ok: false, msg: fmt.Sprintf("cannot read the index: %v — run thoth doctor --fix to rebuild it", err)}
	}
	if indexed != onDisk {
		return checkResult{name: "index", ok: false, msg: fmt.Sprintf("index has %d notes but %d notes on disk — run thoth doctor --fix to rebuild", indexed, onDisk)}
	}
	return checkResult{name: "index", ok: true, msg: fmt.Sprintf("in sync: %d notes indexed, %d on disk", indexed, onDisk)}
}

// countNotes counts .md files with valid frontmatter under root, mirroring
// index.Rebuild's definition of an indexable note.
func countNotes(root string) (int, error) {
	n := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".md" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil // unreadable notes are skipped by Rebuild too
		}
		if _, _, err := wiki.ParseNote(b); err != nil {
			return nil
		}
		n++
		return nil
	})
	return n, err
}

// checkPort verifies that nothing is listening on the configured address.
func checkPort(cfg config.Config) checkResult {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 8333
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, portDialTimeout)
	if err == nil {
		conn.Close()
		return checkResult{name: "port", ok: false, msg: fmt.Sprintf("%s is already in use — stop the other process or change the port in the config", addr)}
	}
	return checkResult{name: "port", ok: true, msg: fmt.Sprintf("%s is free", addr)}
}
