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
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/wiki"
)

const (
	claudeCheckTimeout = 5 * time.Second
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
// check, in order: wiki, claude, claude login, database, index, api,
// websocket. Every check runs even when an earlier one fails. addr is the
// host:port the api/websocket checks probe ("" → 127.0.0.1:8333); tests
// inject a free port. ctx bounds the claude probes; log is reserved for
// future diagnostics.
func Run(ctx context.Context, dir string, addr string, log *slog.Logger) []Check {
	thDir := dir
	if thDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return []Check{{Name: "home", OK: false, Message: fmt.Sprintf("cannot resolve home directory: %v", err)}}
		}
		thDir = filepath.Join(home, ".thoth")
	}
	if addr == "" {
		addr = defaultAddr()
	}
	dbPath := filepath.Join(thDir, "thoth.db")

	results := make([]Check, 0, 7)

	// 1. wiki — the path lives in the settings table; a missing database
	// falls back to the default (the doctor must not create the DB itself).
	expanded, err := config.ExpandHome(wikiPath(dbPath))
	if err != nil {
		results = append(results, Check{Name: "wiki", OK: false, Message: fmt.Sprintf("cannot expand wiki path %q: %v", wikiPath(dbPath), err)})
		expanded = wikiPath(dbPath)
	}
	results = append(results, checkWiki(expanded))

	// 2. claude CLI
	results = append(results, checkClaude(ctx)...)

	// 3. database
	results = append(results, checkDatabase(dbPath))

	// 4. index sync
	results = append(results, checkIndex(dbPath, expanded))

	// 5. api — REST health at the configured address. Pre-launch (CLI) it
	// reports whether the port is free, occupied by a running Thoth, or by
	// something else; in-flight (GET /api/doctor) it self-checks the very
	// server answering the request.
	apiCheck, apiReachable := checkAPI(addr)
	results = append(results, apiCheck)

	// 6. websocket — the chat upgrade; reported separately, and only probed
	// when the REST check reached a Thoth server.
	results = append(results, checkWebsocket(addr, apiReachable))

	return results
}

// wikiPath reads the settings table when the database exists; a missing or
// unreadable database (or an empty value) falls back to the default.
func wikiPath(dbPath string) string {
	if !fileExists(dbPath) {
		return settings.DefaultWikiPath
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		return settings.DefaultWikiPath
	}
	defer func() { _ = r.Close() }()
	value, found, err := r.Setting(settings.KeyWikiPath)
	if err != nil || !found || value == "" {
		return settings.DefaultWikiPath
	}
	return value
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

// checkClaude resolves "claude" on PATH, reports its version, and probes the
// login state. The login result is only reported when the binary was found.
func checkClaude(ctx context.Context) []Check {
	bin, notFound := resolveClaudeBin()
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

// resolveClaudeBin returns the claude binary to probe: "claude" from PATH.
// The second return value is a failure message, or "" when resolved.
func resolveClaudeBin() (string, string) {
	if p, err := exec.LookPath("claude"); err == nil {
		return p, ""
	}
	return "", "claude CLI not found on PATH — install it"
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
func checkAPI(addr string) (Check, bool) {
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
func checkWebsocket(addr string, apiReachable bool) Check {
	if !apiReachable {
		return Check{Name: "websocket", OK: true, Message: fmt.Sprintf("skipped — api not reachable at %s", addr)}
	}
	if !wsUpgradeOK(addr) {
		return Check{Name: "websocket", OK: false, Message: fmt.Sprintf("chat websocket did not connect at %s", addr)}
	}
	return Check{Name: "websocket", OK: true, Message: fmt.Sprintf("chat websocket connects at %s", addr)}
}

// defaultAddr is the fixed server address (host/port are code constants).
func defaultAddr() string {
	return net.JoinHostPort(config.DefaultHost, strconv.Itoa(config.DefaultPort))
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

// (removed with config.toml: the claude binary always comes from PATH)
