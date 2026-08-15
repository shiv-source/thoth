// Package doctor runs the Thoth installation health checks. It is shared by
// the `thoth doctor` CLI (which adds --fix repairs on top) and the
// `GET /api/doctor` endpoint (which renders the same checks in the settings
// modal). Check messages are the single source of truth for both surfaces.
package doctor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/wiki"
)

const (
	claudeCheckTimeout = 2 * time.Second
	apiCheckTimeout    = 2 * time.Second
)

// Check is one named health check: what was probed and the human-readable
// outcome (ok or a message explaining what is wrong and how to fix it).
type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// Run probes the installation under dir ("" means ~/.thoth) and returns every
// check, in order: config, wiki, claude, claude login, database, index, api,
// websocket. Every check runs even when an earlier one fails. ctx bounds the
// claude probes; log is reserved for future diagnostics.
func Run(ctx context.Context, dir string, log *slog.Logger) []Check {
	thDir := dir
	if thDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return []Check{{Name: "home", OK: false, Message: fmt.Sprintf("cannot resolve home directory: %v", err)}}
		}
		thDir = filepath.Join(home, ".thoth")
	}
	cfgPath := filepath.Join(thDir, "config.toml")
	dbPath := filepath.Join(thDir, "thoth.db")

	results := make([]Check, 0, 7)

	// 1. config
	cfg, res := checkConfig(cfgPath)
	results = append(results, res)

	// 2. wiki
	expanded, err := config.ExpandHome(cfg.WikiPath)
	if err != nil {
		results = append(results, Check{Name: "wiki", OK: false, Message: fmt.Sprintf("cannot expand wiki path %q: %v", cfg.WikiPath, err)})
		expanded = cfg.WikiPath
	}
	results = append(results, checkWiki(expanded))

	// 3. claude CLI
	results = append(results, checkClaude(ctx, cfg)...)

	// 4. database
	results = append(results, checkDatabase(dbPath))

	// 5. index sync
	results = append(results, checkIndex(dbPath, expanded))

	// 6. api — REST health at the configured address. Pre-launch (CLI) it
	// reports whether the port is free, occupied by a running Thoth, or by
	// something else; in-flight (GET /api/doctor) it self-checks the very
	// server answering the request.
	apiCheck, apiReachable := checkAPI(cfg)
	results = append(results, apiCheck)

	// 7. websocket — the chat upgrade; reported separately, and only probed
	// when the REST check reached a Thoth server.
	results = append(results, checkWebsocket(cfg, apiReachable))

	return results
}

// checkConfig verifies that ~/.thoth/config.toml exists and parses. On any
// failure the caller proceeds with defaults so later checks still run.
func checkConfig(path string) (config.Config, Check) {
	if !fileExists(path) {
		return config.Default(), Check{Name: "config", OK: false, Message: fmt.Sprintf("%s is missing — run thoth doctor --fix to create a default config", path)}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Default(), Check{Name: "config", OK: false, Message: fmt.Sprintf("%s cannot be parsed: %v — fix the syntax or recreate the file", path, err)}
	}
	return cfg, Check{Name: "config", OK: true, Message: fmt.Sprintf("%s parses", path)}
}

// checkWiki verifies that the wiki directory exists with the scaffold folders
// and CLAUDE.md.
func checkWiki(root string) Check {
	if !isDir(root) {
		return Check{Name: "wiki", OK: false, Message: fmt.Sprintf("%s does not exist — run thoth doctor --fix to scaffold it", root)}
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
		return Check{Name: "wiki", OK: false, Message: fmt.Sprintf("%s is missing %s — run thoth doctor --fix to repair it", root, strings.Join(missing, ", "))}
	}
	return Check{Name: "wiki", OK: true, Message: fmt.Sprintf("%s exists with the 8 scaffold folders and CLAUDE.md", root)}
}

// checkClaude resolves the claude binary (config.ClaudeBin or "claude" on
// PATH), reports its version, and probes the login state. The login result is
// only reported when the binary was found.
func checkClaude(ctx context.Context, cfg config.Config) []Check {
	bin, notFound := resolveClaudeBin(cfg)
	if notFound != "" {
		return []Check{{Name: "claude", OK: false, Message: notFound}}
	}
	cctx, cancel := context.WithTimeout(ctx, claudeCheckTimeout)
	defer cancel()
	ver, err := exec.CommandContext(cctx, bin, "--version").Output()
	if err != nil {
		return []Check{{Name: "claude", OK: false, Message: fmt.Sprintf("%s --version failed: %v — is the CLI installed correctly?", bin, err)}}
	}
	results := []Check{{Name: "claude", OK: true, Message: fmt.Sprintf("%s (version %s)", bin, strings.TrimSpace(string(ver)))}}
	cctx, cancel = context.WithTimeout(ctx, claudeCheckTimeout)
	defer cancel()
	if err := exec.CommandContext(cctx, bin, "auth", "status").Run(); err != nil {
		results = append(results, Check{Name: "claude login", OK: false, Message: "login status unknown (claude auth status did not exit 0)"})
	} else {
		results = append(results, Check{Name: "claude login", OK: true, Message: "login confirmed (claude auth status exited 0)"})
	}
	return results
}

// resolveClaudeBin returns the claude binary to probe: config.ClaudeBin when
// configured, otherwise "claude" from PATH. The second return value is a
// failure message, or "" when a binary was resolved.
func resolveClaudeBin(cfg config.Config) (string, string) {
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

// checkDatabase verifies that thoth.db exists, opens in WAL mode, and has the
// notes and notes_fts tables.
func checkDatabase(path string) Check {
	if !fileExists(path) {
		return Check{Name: "database", OK: false, Message: fmt.Sprintf("%s does not exist — start thoth serve to create it", path)}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return Check{Name: "database", OK: false, Message: fmt.Sprintf("%s cannot be opened: %v", path, err)}
	}
	defer func() { _ = db.Close() }()
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode;`).Scan(&mode); err != nil {
		return Check{Name: "database", OK: false, Message: fmt.Sprintf("%s is not a usable sqlite database: %v", path, err)}
	}
	if !strings.EqualFold(mode, "wal") {
		return Check{Name: "database", OK: false, Message: fmt.Sprintf("%s journal mode is %q, want \"wal\"", path, mode)}
	}
	var tables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('notes', 'notes_fts')`).Scan(&tables); err != nil {
		return Check{Name: "database", OK: false, Message: fmt.Sprintf("cannot inspect %s: %v", path, err)}
	}
	if tables != 2 {
		return Check{Name: "database", OK: false, Message: fmt.Sprintf("%s is missing the notes/notes_fts tables — run thoth doctor --fix", path)}
	}
	return Check{Name: "database", OK: true, Message: fmt.Sprintf("%s opens in WAL with notes + notes_fts tables", path)}
}

// checkIndex compares the number of notes in the index against a rescan of the
// wiki's .md files with valid frontmatter.
func checkIndex(dbPath, wikiRoot string) Check {
	onDisk, err := countNotes(wikiRoot)
	if err != nil {
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("cannot scan %s: %v — run thoth doctor --fix", wikiRoot, err)}
	}
	if !fileExists(dbPath) {
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("%s does not exist — run thoth doctor --fix to create and populate it", dbPath)}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("cannot open %s: %v", dbPath, err)}
	}
	defer func() { _ = db.Close() }()
	var indexed int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&indexed); err != nil {
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("cannot read the index: %v — run thoth doctor --fix to rebuild it", err)}
	}
	if indexed != onDisk {
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("index has %d notes but %d notes on disk — run thoth doctor --fix to rebuild", indexed, onDisk)}
	}
	return Check{Name: "index", OK: true, Message: fmt.Sprintf("in sync: %d notes indexed, %d on disk", indexed, onDisk)}
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

// checkAPI probes the REST health endpoint at the configured address and
// reports whether a Thoth server answered (the bool) alongside the check.
// The same check runs pre-launch and in-flight; see Run.
func checkAPI(cfg config.Config) (Check, bool) {
	addr := configAddr(cfg)

	// A TCP dial distinguishes "nothing there" from "something there that
	// does not speak the Thoth protocol".
	conn, err := net.DialTimeout("tcp", addr, apiCheckTimeout)
	if err != nil {
		return Check{Name: "api", OK: true, Message: fmt.Sprintf("api not running at %s", addr)}, false
	}
	_ = conn.Close()

	client := &http.Client{Timeout: apiCheckTimeout}
	resp, err := client.Get("http://" + addr + "/api/health")
	if err != nil {
		return Check{Name: "api", OK: false, Message: fmt.Sprintf("port %s is occupied by a non-thoth process", addr)}, false
	}
	defer func() { _ = resp.Body.Close() }()
	var h struct {
		Status string `json:"status"`
		Claude struct {
			Found bool   `json:"found"`
			Path  string `json:"path"`
		} `json:"claude"`
		Wiki struct {
			Exists bool `json:"exists"`
		} `json:"wiki"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&h) != nil || h.Status != "ok" {
		return Check{Name: "api", OK: false, Message: fmt.Sprintf("port %s is occupied by a non-thoth process", addr)}, false
	}
	if !h.Claude.Found {
		return Check{Name: "api", OK: false, Message: fmt.Sprintf("api healthy at %s but the claude CLI was not found", addr)}, true
	}
	if !h.Wiki.Exists {
		return Check{Name: "api", OK: false, Message: fmt.Sprintf("api healthy at %s but the wiki path does not exist", addr)}, true
	}
	return Check{Name: "api", OK: true, Message: fmt.Sprintf("api healthy at %s — REST", addr)}, true
}

// checkWebsocket probes the chat upgrade at the configured address. It only
// runs when the REST check reached a Thoth server; otherwise it reports a
// skip so a pre-launch run stays clean.
func checkWebsocket(cfg config.Config, apiReachable bool) Check {
	addr := configAddr(cfg)
	if !apiReachable {
		return Check{Name: "websocket", OK: true, Message: fmt.Sprintf("skipped — api not reachable at %s", addr)}
	}
	if !wsUpgradeOK(addr) {
		return Check{Name: "websocket", OK: false, Message: fmt.Sprintf("chat websocket did not connect at %s", addr)}
	}
	return Check{Name: "websocket", OK: true, Message: fmt.Sprintf("chat websocket connects at %s", addr)}
}

// configAddr returns the configured host:port with defaults applied.
func configAddr(cfg config.Config) string {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 8333
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// wsUpgradeOK dials the chat websocket and reports whether the upgrade
// succeeds. The connection is closed immediately — no frames are sent.
func wsUpgradeOK(addr string) bool {
	dialer := websocket.Dialer{HandshakeTimeout: apiCheckTimeout}
	conn, _, err := dialer.Dial("ws://"+addr+"/ws", nil)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isExecutable(bin string) bool {
	if strings.ContainsRune(bin, os.PathSeparator) {
		fi, err := os.Stat(bin)
		return err == nil && !fi.IsDir()
	}
	_, err := exec.LookPath(bin)
	return err == nil
}
