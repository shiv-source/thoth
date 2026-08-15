# Components

The Go backend is organized as small packages with one purpose each, communicating through narrow interfaces. Everything lives under `internal/` except the binary entrypoint.

| Package | Responsibility | Key exports |
|---|---|---|
| `cmd/thoth` | Thin `main` — calls the CLI and exits | `main` |
| `internal/cli` | Cobra commands: serve, init, version, doctor | `Execute()` |
| `internal/claude` | **The blast wall** — the only package that knows CLI flags, stream parsing, and process kill mechanics | `Client`, `CLIClient`, `PersistentClient`, `Event`, `ParseLine`, `FakeClient` |
| `internal/wiki` | The file contract: scaffolding, parsing, path safety, tree | `Scaffold`, `ParseNote`, `SafePath`, `Wiki`, `Rulebook` |
| `internal/index` | SQLite + FTS5 + watcher | `Index`, `Sync`, `Watch`, `Search` |
| `internal/assets` | Static data files served by the API (embedded) | `models.json` → `ModelOptions` (the Settings model picker list) |
| `internal/store` | Conversations and messages (same db file) | `Store` |
| `internal/api` | Echo server: routes, WS hub, handlers | `Deps`, `New`, `Hub` |
| `internal/config` | Localhost bind constants (`127.0.0.1:8333`) + `ExpandHome` path helper | `DefaultHost`, `DefaultPort`, `ExpandHome` |
| `internal/doctor` | Shared install checks (the `thoth doctor` CLI and the Settings → Doctor tab run the same suite) | `Check`, `Run` |
| `internal/github` | GitHub identity (PAT storage) and git sync | `Client`, `Auth`, `Repo`, `Service` |
| `internal/settings` | The settings KV table (thoth.db) — the single source for user-facing settings | `Repo`, `OpenRepo` |
| `internal/webui` | Embedded frontend | `Register` |

## internal/claude — the blast wall

Everything version-sensitive about the Claude Code CLI lives in exactly two files:

- `client.go` — the flag lists (per-turn `-p --output-format stream-json --verbose --session-id …` and persistent-mode `-p --input-format stream-json … --autocompact auto`; plus `--dangerously-skip-permissions` by default, or `--permission-mode <mode>` when configured, plus optional `--model`), spawn, stream scanning, cancel; stderr is captured and appended to exit errors
- `persistent.go` — `PersistentClient`, a pool of long-lived CLI processes keyed by session id: lazily spawned, one dispatcher goroutine per process turns stdout lines into events for the in-flight turn and ends it at the CLI's `result` line; cancel kills the process and the next turn respawns; idle processes evict after 10 min; `Flush` on wiki-path change, `Close` on shutdown
- `events.go` — tolerant parsing of `stream-json` lines into typed events: `assistant_delta`, `thinking` (thinking-only assistant blocks — the UI shows "Thinking…" with the block text), `tool_activity`, `turn_done`, `error`; the raw stream is also appended to `~/.thoth/stream-dump.json` for debugging (rotated past 10MB)

```go
type Client interface {
    Start(ctx context.Context, sessionID, prompt string, w EventWriter) error
}
```

Cancelling `ctx` kills the CLI's process group (unix) or direct child (windows) — `proc_unix.go` / `proc_windows.go` are build-tagged for that; for a pooled process the kill evicts it and the next turn respawns (there is no per-turn interrupt in the plain CLI). A CLI upgrade can only ever break this package; everything else is stable.

`FakeClient` replays scripted events and records calls — every consumer's tests use it, so no test ever touches the real CLI.

## internal/wiki — the file contract

- `Scaffold(dir)` — creates the 8 folders + rulebook; never overwrites an existing `CLAUDE.md`
- `Rulebook()` — the single source of the rulebook text (embedded template)
- `ParseNote(content)` — splits frontmatter, requires `title`, returns `NoteMeta` + body
- `SafePath(root, rel)` — rejects absolute paths and `..` escapes; every filesystem access routes through it
- `Wiki` — `New`, `Exists`, `Read`, `Tree` (dirs first, dotfiles skipped)

## internal/index — search and sync

- `Open(path)` — WAL mode + schema migration
- `Upsert` / `Delete` / `DeletePrefix` — with FTS5 triggers keeping the index in sync
- `Search(q, limit)` — bm25 ranking (title 8×) and body snippets, HTML-escaped
- `Sync(root, log)` — reconciles the index with the tree in one transaction (unchanged files skipped, missing files deleted); malformed notes skipped
- `Watch(ctx, root, ix, log)` — fsnotify with 200 ms debounce and new-directory rescan

Full mechanics: [Indexing & search](indexing.md).

## internal/api — the server

`Deps` carries every dependency; `New(d Deps) *echo.Echo` wires all routes. `Hub` owns the WebSocket lifecycle: one active turn per conversation, supersede-on-send, cancel, and a 500-message replay buffer for reconnects. Protocol details: [API](api.md).

## internal/store

Conversations and messages in the same `thoth.db` (separate `*sql.DB`, WAL makes sharing safe). The whole schema lives in embedded `.sql` migrations in `migrations/`, applied in filename order and gated on `PRAGMA user_version`; a single-row `app_metadata` table (enforced by `CHECK (id = 1)`) holds install facts (`installation_id`, `created_at`, seeded by `EnsureMetadata` on boot) and git sync state (`last_synced_at`, `sync_error`, written by `SetSyncResult`). IDs are valid RFC 4122 v4 UUIDs (`google/uuid`) because the Claude CLI requires UUIDs for `--session-id`. Timestamps are stored UTC so ordering is chronological.

## internal/cli

`Execute()` builds the root Cobra command. `serve` is a thin orchestration function whose helpers (`loadConfig`, `ensureWiki`, `openStores`, `resolveClaudeBin`, `onSettingsSaved`, `serveUntilShutdown`) keep each step readable. Details: [CLI](cli.md).

## internal/doctor

`Run(ctx, dir, addr, log)` runs the six install checks (wiki, claude, database, index, api, websocket) and returns `[]Check`, each carrying `Name`/`OK`/`Message`. The dashboard's Settings → Doctor tab runs the same suite via `GET /api/doctor` (details: [CLI](cli.md)).

## internal/github

`Client` talks to api.github.com (identity + repos; the token is never sent anywhere else), `Repo` stores the PAT and identity in the `github_auth` table, and `Service` drives the git sync (init, remote, commit, push — errors sanitized). Details: [Schema](schema.md).

## internal/settings

`Repo` (`OpenRepo(path)`) owns the `settings` KV table — `wiki_path`, `github_sync_*` keys — with sync-state conveniences. It deliberately runs no migrations and no WAL pragma: the doctor must never mutate a database it only reads. Details: [Schema](schema.md).
