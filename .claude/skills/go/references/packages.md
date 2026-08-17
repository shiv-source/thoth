# Go packages (internal/* + cmd/thoth)

Each entry: path, purpose, api, canonical. Coverage: every package in
internal/ plus cmd/thoth — a missing package means this index is stale.

## cmd/thoth
- path: cmd/thoth/main.go
- purpose: Thin binary entrypoint — calls the CLI and exits
- api: main
- canonical: docs/components.md package table · cmd/thoth/main.go

## internal/cli
- path: internal/cli/
- purpose: Cobra commands — serve, init, version, doctor
- api: Execute(); serve helpers: ensureWiki, resolveClaudeBin, onSettingsSaved, serveUntilShutdown
- canonical: docs/components.md §internal/cli · internal/cli/root.go

## internal/claude — the blast wall
- path: internal/claude/
- purpose: The only package that knows CLI flags, stream parsing, and process kill
- api: Client (Start(ctx, sessionID, prompt, w EventWriter) error), CLIClient, PersistentClient, Event, ParseLine, FakeClient
- canonical: docs/components.md §internal/claude · internal/claude/client.go
- see: claude-blast-wall.md

## internal/wiki
- path: internal/wiki/
- purpose: The file contract — scaffolding, parsing, path safety, tree
- api: Scaffold, ParseNote, SafePath, Wiki (New/Exists/Read/Tree), Rulebook
- canonical: docs/components.md §internal/wiki · internal/wiki/wiki.go

## internal/index
- path: internal/index/
- purpose: SQLite + FTS5 search + fsnotify watcher
- api: Index, Sync, Watch, Search — Open, Upsert, Delete, DeletePrefix
- canonical: docs/components.md §internal/index · internal/index/index.go

## internal/assets
- path: internal/assets/
- purpose: Embedded static data — models.json → ModelOptions (Settings model picker)
- api: ModelOptions
- canonical: docs/components.md package table · internal/assets/assets.go

## internal/store
- path: internal/store/
- purpose: Conversations + messages (same db file); migrations/ = all DDL
- api: Store (Open, ListConversations, Messages, Close), EnsureMetadata
- canonical: docs/components.md §internal/store · internal/store/store.go

## internal/api
- path: internal/api/
- purpose: Echo server — routes, WS hub, handlers
- api: Deps, New(d Deps) *echo.Echo, Hub
- canonical: docs/components.md §internal/api · internal/api/server.go

## internal/config
- path: internal/config/
- purpose: Localhost bind constants + path helper
- api: DefaultHost, DefaultPort (127.0.0.1:8333), ExpandHome
- canonical: docs/components.md package table · internal/config/config.go

## internal/doctor
- path: internal/doctor/
- purpose: The six shared install checks (CLI + Settings → Doctor tab)
- api: Run(ctx, dir, addr, log) []Check; Check{Name, OK, Message}
- canonical: docs/components.md §internal/doctor · internal/doctor/doctor.go

## internal/github
- path: internal/github/
- purpose: GitHub identity (PAT storage) + git sync
- api: Client, Auth, Repo, Service
- canonical: docs/components.md §internal/github · internal/github/service.go

## internal/settings
- path: internal/settings/
- purpose: The settings KV table — single source for user-facing settings
- api: Repo, OpenRepo(path); SyncEnabled/SyncState/SetSyncResult conveniences
- canonical: docs/components.md §internal/settings · internal/settings/settings.go

## internal/webui
- path: internal/webui/
- purpose: Embedded frontend (generated dist; //go:embed all:dist)
- api: Register
- canonical: docs/components.md package table · internal/webui/embed.go

Stale if: a new package appears in internal/, an export listed above is
renamed, or docs/components.md's package table gains a row this index
lacks.
