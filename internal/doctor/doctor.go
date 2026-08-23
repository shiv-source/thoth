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
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
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
	providerCheckTimeout = 5 * time.Second
	apiCheckTimeout      = 2 * time.Second
)

// Check is one named health check: what was probed and the human-readable
// outcome (ok or a message explaining what is wrong and how to fix it).
type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// Options configure Run.
type Options struct {
	Dir  string       // thoth dir to check ("" → ~/.thoth)
	Addr string       // host:port the api/websocket checks probe ("" → 127.0.0.1:8333)
	Log  *slog.Logger // diagnostics (reserved for future checks)
	HTTP *http.Client // HTTP client for the provider probe (nil → http.DefaultClient); tests stub the endpoint
	// BaseURL overrides the provider base URL the provider probe targets
	// ("" → the provider's public endpoint); tests point it at a stub so the
	// probe never touches the live network.
	BaseURL string
}

// Run probes the installation and returns every check, in order: wiki,
// provider, api key, model, database, index, malformed, api, websocket. Every
// check runs even when an earlier one fails. opts.Dir ("" → ~/.thoth) is the
// install to check; opts.Addr is the host:port the api/websocket checks probe
// ("" → 127.0.0.1:8333; tests inject a free port). ctx bounds the provider
// probe; opts.HTTP and opts.BaseURL stub its endpoint in tests.
func Run(ctx context.Context, opts Options) []Check {
	dir := opts.Dir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return []Check{{Name: "home", OK: false, Message: fmt.Sprintf("cannot resolve home directory: %v", err)}}
		}
		dir = filepath.Join(home, ".thoth")
	}
	addr := opts.Addr
	if addr == "" {
		addr = defaultAddr()
	}
	dbPath := filepath.Join(dir, "thoth.db")

	results := make([]Check, 0, 9)

	// 1. wiki — the path lives in the settings table; a missing database
	// falls back to the default (the doctor must not create the DB itself).
	expanded, err := config.ExpandHome(wikiPath(dbPath))
	if err != nil {
		results = append(results, Check{Name: "wiki", OK: false, Message: fmt.Sprintf("cannot expand wiki path %q: %v", wikiPath(dbPath), err)})
		expanded = wikiPath(dbPath)
	}
	results = append(results, checkWiki(expanded, wikiFolders(dbPath)))

	// 2. provider — native backend connectivity, probed against the provider
	// the selected model maps to.
	results = append(results, checkProvider(ctx, dbPath, opts))

	// 3. setup state — api key and model, from the settings table.
	results = append(results, checkSettings(dbPath)...)

	// 4. database
	results = append(results, checkDatabase(dbPath))

	// 5. index sync
	results = append(results, checkIndex(dbPath, expanded))

	// 6. malformed — notes the index silently skips, surfaced by name.
	results = append(results, checkMalformed(expanded))

	// 7. api — REST health at the configured address. Pre-launch (CLI) it
	// reports whether the port is free, occupied by a running Thoth, or by
	// something else; in-flight (GET /api/doctor) it self-checks the very
	// server answering the request.
	apiCheck, apiReachable := checkAPI(addr)
	results = append(results, apiCheck)

	// 8. websocket — the chat upgrade; reported separately, and only probed
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

// wikiFolders reads the configured scaffold folder set from the settings
// table; a missing or unreadable database falls back to nil (the defaults).
func wikiFolders(dbPath string) []string {
	if !fileExists(dbPath) {
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

// checkWiki verifies that the wiki directory exists with the scaffold folders
// (the configured set, or the defaults when none) and CLAUDE.md.
func checkWiki(root string, folders []string) Check {
	if !isDir(root) {
		return Check{Name: "wiki", OK: false, Message: fmt.Sprintf("%s does not exist — run thoth doctor --fix to scaffold it", root)}
	}
	if len(folders) == 0 {
		folders = wiki.Folders()
	}
	var missing []string
	for _, f := range folders {
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
	return Check{Name: "wiki", OK: true, Message: fmt.Sprintf("%s exists with the %d scaffold folders and CLAUDE.md", root, len(folders))}
}

// providerProbe is the models-style request the connectivity check makes
// against one provider family: the cheapest authenticated probe the provider
// offers (it consumes no tokens), whose response status distinguishes the
// failure modes the native agent actually hits.
type providerProbe struct {
	name    string // display name, e.g. "Anthropic"
	baseURL string // provider origin; the models request path is appended
	path    string // e.g. /v1/models
	key     string // request header the API key rides in
	bearer  bool   // true → Authorization: Bearer, false → a bare header
	version string // anthropic-version header (Anthropic only)
}

// providerProbeFor maps a model to the provider probe that serves it,
// mirroring internal/agent's provider selection: the model's llm_models
// provider label wins — "Anthropic" probes Anthropic, every other name the
// OpenAI-compatible models endpoint (DeepSeek, Qwen, GLM, … all speak the
// OpenAI wire shape); an empty label falls back to the model id prefixes
// (claude-* Anthropic, gpt-* OpenAI-compatible). baseURL overrides the
// probe's default origin when non-empty (the configured per-provider base
// url). Families without a concrete provider here error, exactly as the
// agent would when asked to run them.
func providerProbeFor(model, providerName, baseURL string) (providerProbe, error) {
	var probe providerProbe
	switch {
	case providerName == "Anthropic" || (providerName == "" && strings.HasPrefix(model, "claude-")):
		probe = providerProbe{name: "Anthropic", baseURL: "https://api.anthropic.com", path: "/v1/models", key: "x-api-key", version: "2023-06-01"}
	case providerName != "":
		probe = providerProbe{name: providerName, baseURL: "https://api.openai.com", path: "/v1/models", key: "Authorization", bearer: true}
	case strings.HasPrefix(model, "gpt-"):
		probe = providerProbe{name: "OpenAI", baseURL: "https://api.openai.com", path: "/v1/models", key: "Authorization", bearer: true}
	default:
		return providerProbe{}, fmt.Errorf("no provider for model %q", model)
	}
	if baseURL != "" {
		probe.baseURL = baseURL
	}
	return probe, nil
}

// checkProvider probes the configured provider's models endpoint with the
// resolved API key and reports reachability, keeping auth failure distinct
// from a down endpoint or provider-side error. The provider and its
// credentials resolve the way serve resolves them at boot: the selected
// model's llm_models row names the provider, whose per-provider api key and
// base url are used (empty base url → the provider default). The probe is
// bounded by ctx and opts.HTTP (nil → http.DefaultClient); opts.BaseURL
// overrides the resolved base url so tests stub the endpoint.
func checkProvider(ctx context.Context, dbPath string, opts Options) Check {
	model, err := selectedModel(dbPath)
	if err != nil {
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("cannot read the settings table: %v", err)}
	}
	if model == "" {
		return Check{Name: "provider", OK: false, Message: "no model selected — cannot determine which provider to probe"}
	}
	providerName := modelProvider(dbPath, model)
	key, baseURL, err := providerConfig(dbPath, providerName)
	if err != nil {
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("cannot read the settings table: %v", err)}
	}
	probe, err := providerProbeFor(model, providerName, baseURL)
	if err != nil {
		return Check{Name: "provider", OK: false, Message: err.Error()}
	}
	if opts.BaseURL != "" {
		probe.baseURL = opts.BaseURL
	}
	pctx, cancel := context.WithTimeout(ctx, providerCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(pctx, http.MethodGet, strings.TrimRight(probe.baseURL, "/")+probe.path, nil)
	if err != nil {
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("cannot build provider probe: %v", err)}
	}
	if probe.bearer {
		req.Header.Set("authorization", "Bearer "+key)
	} else {
		req.Header.Set(probe.key, key)
	}
	if probe.version != "" {
		req.Header.Set("anthropic-version", probe.version)
	}
	client := opts.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		if pctx.Err() != nil {
			return Check{Name: "provider", OK: false, Message: fmt.Sprintf("%s probe timed out", probe.name)}
		}
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("%s endpoint unreachable: %v", probe.name, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode == http.StatusOK:
		return Check{Name: "provider", OK: true, Message: fmt.Sprintf("%s reachable at %s", probe.name, probe.baseURL)}
	case resp.StatusCode == http.StatusUnauthorized:
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("%s rejected the API key (401) — set a valid one in Settings → Providers", probe.name)}
	case resp.StatusCode == http.StatusTooManyRequests:
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("%s rate limited (429) — retry later", probe.name)}
	case resp.StatusCode >= 500:
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("%s returned a server error (%s)", probe.name, resp.Status)}
	default:
		return Check{Name: "provider", OK: false, Message: fmt.Sprintf("%s returned %s", probe.name, resp.Status)}
	}
}

// selectedModel reads the model from the settings table; a missing database
// yields an empty value and no error, so callers fall through to their
// unset-state messages.
func selectedModel(dbPath string) (string, error) {
	if !fileExists(dbPath) {
		return "", nil
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()
	model, _, err := r.Setting(settings.KeyModel)
	return model, err
}

// modelProvider returns the providers row's name for a model value, or ""
// when the model is absent (an empty value, an unreadable registry, or no
// matching row) — the same fallback serve's boot uses.
func modelProvider(dbPath, value string) string {
	if value == "" || !fileExists(dbPath) {
		return ""
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return ""
	}
	defer func() { _ = db.Close() }()
	var provider string
	if err := db.QueryRow(`
		SELECT p.name FROM llm_models m
		LEFT JOIN providers p ON p.id = m.provider_id
		WHERE m.value = ?`, value).Scan(&provider); err != nil {
		return ""
	}
	return provider
}

// providerConfig resolves the per-provider api key and base url override for
// a provider name, mirroring settings.Repo.ProviderConfig: keys live in
// thoth.db only (no shared fallback, nothing from the environment), and an
// empty base url means the provider's default endpoint. An empty provider
// name (no registry row) resolves to an empty key. A missing database yields
// empty values and no error.
func providerConfig(dbPath, providerName string) (apiKey, baseURL string, err error) {
	if !fileExists(dbPath) {
		return "", "", nil
	}
	r, err := settings.OpenRepo(dbPath)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = r.Close() }()
	return r.ProviderConfig(providerName)
}

// unreadableSettings builds the failed api key + model checks for a settings
// table that cannot be read.
func unreadableSettings(err error) []Check {
	msg := fmt.Sprintf("cannot read the settings table: %v", err)
	return []Check{{Name: "api key", OK: false, Message: msg}, {Name: "model", OK: false, Message: msg}}
}

// checkSettings reports the user-facing setup state: whether the API key and
// the selected model are configured. Both are optional (an unset model keeps
// the default), so an unset value is a failed check whose message explains
// what to do. A model whose value is not in the llm_models registry is
// reported as unknown.
func checkSettings(dbPath string) []Check {
	unset := func(name, message string) Check { return Check{Name: name, OK: false, Message: message} }
	model, err := selectedModel(dbPath)
	if err != nil {
		return unreadableSettings(err)
	}
	// The credential the agent actually uses is the selected provider's own
	// key — read from the DB only, no shared fallback.
	var resolvedKey string
	if model != "" {
		if resolvedKey, _, err = providerConfig(dbPath, modelProvider(dbPath, model)); err != nil {
			return unreadableSettings(err)
		}
	}
	checks := make([]Check, 0, 2)
	if resolvedKey != "" {
		checks = append(checks, Check{Name: "api key", OK: true, Message: "API key configured"})
	} else {
		checks = append(checks, unset("api key", "no API key configured — set the provider's API key in Settings → Providers"))
	}
	if model != "" {
		if modelKnown(dbPath, model) {
			checks = append(checks, Check{Name: "model", OK: true, Message: fmt.Sprintf("model %q selected", model)})
		} else {
			checks = append(checks, Check{Name: "model", OK: false, Message: fmt.Sprintf("unknown model %q — it is not in the model registry; add or pick one in Settings → Providers", model)})
		}
	} else {
		checks = append(checks, unset("model", "no model selected — the default model is used; pick one in Settings → General"))
	}
	return checks
}

// modelKnown reports whether value exists in the llm_models registry. An
// unreadable registry (or an absent table) counts as unknown.
func modelKnown(dbPath, value string) bool {
	if value == "" || !fileExists(dbPath) {
		return false
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return false
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM llm_models WHERE value = ?`, value).Scan(&n); err != nil {
		return false
	}
	return n > 0
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
// wiki, mirroring index.Sync's definition: every non-hidden file, markdown
// notes requiring valid frontmatter and attachments counted by filename.
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
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("cannot read the index: %v — run thoth doctor --fix to sync it", err)}
	}
	if indexed != onDisk {
		return Check{Name: "index", OK: false, Message: fmt.Sprintf("index has %d notes but %d notes on disk — run thoth doctor --fix to sync", indexed, onDisk)}
	}
	return Check{Name: "index", OK: true, Message: fmt.Sprintf("in sync: %d notes indexed, %d on disk", indexed, onDisk)}
}

// countNotes counts every file index.Sync would index: all non-hidden files,
// where markdown notes must parse with valid frontmatter. Attachments are
// indexed by filename regardless of content.
func countNotes(root string) (int, error) {
	n := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		if !wiki.Indexable(filepath.ToSlash(rel)) {
			return nil
		}
		if !wiki.IsMarkdownPath(p) {
			n++ // attachment: indexed by filename only
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil // unreadable notes are skipped by Sync too
		}
		if _, _, err := wiki.ParseNote(b); err != nil {
			return nil
		}
		n++
		return nil
	})
	return n, err
}

// checkMalformed walks the wiki and reports markdown notes that the index
// silently skips — notes whose frontmatter fails to parse (see index.Sync).
// Advisory save-protocol warnings (missing type, non-kebab filename) still
// index, so they are surfaced by the index's own logs, not this check.
func checkMalformed(root string) Check {
	bad := []string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // an unreadable wiki (or subtree) is a real problem
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !wiki.Indexable(rel) || !wiki.IsMarkdownPath(rel) {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		if _, _, perr := wiki.ParseNote(b); perr != nil {
			bad = append(bad, rel+" ("+perr.Error()+")")
		}
		return nil
	})
	if err != nil {
		return Check{Name: "malformed", OK: false, Message: fmt.Sprintf("cannot scan %s: %v", root, err)}
	}
	if len(bad) == 0 {
		return Check{Name: "malformed", OK: true, Message: "every note parses as a valid note"}
	}
	shown, rest := bad, 0
	if len(bad) > 5 {
		shown, rest = bad[:5], len(bad)-5
	}
	msg := "notes the index skips: " + strings.Join(shown, "; ")
	if rest > 0 {
		msg += fmt.Sprintf("; and %d more", rest)
	}
	return Check{Name: "malformed", OK: false, Message: msg}
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
	resp, err := client.Get("http://" + addr + "/api/v1/health")
	if err != nil {
		return Check{Name: "api", OK: false, Message: fmt.Sprintf("port %s is occupied by a non-thoth process", addr)}, false
	}
	defer func() { _ = resp.Body.Close() }()
	var h struct {
		Status string `json:"status"`
		Wiki   struct {
			Exists bool `json:"exists"`
		} `json:"wiki"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&h) != nil || h.Status != "ok" {
		return Check{Name: "api", OK: false, Message: fmt.Sprintf("port %s is occupied by a non-thoth process", addr)}, false
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
	conn, _, err := dialer.Dial("ws://"+addr+"/ws/v1", nil)
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
