# Components

The Go backend is organized as small packages with one purpose each, communicating through narrow interfaces. Everything lives under `internal/` except the binary entrypoint.

| Package | Responsibility | Key exports |
|---|---|---|
| `cmd/thoth` | Thin `main` — calls the CLI and exits | `main` |
| `internal/cli` | Cobra commands: serve, init, version, doctor | `Execute()` |
| `agent` | The reusable native-agent library (repo-root module): provider-agnostic tool-use loop, the `Provider` wire seam, the tool registry, normalized events/model — the engine behind **Thoth Agent** | `Agent`, `New`, `Options`, `Event`, `Provider`, `Registry` |
| `agent/git` | The pure-Go git backend the agent's git tools run on — a go-git wrapper (no shell, no cgo) that stays wiki-agnostic and dependency-free of `internal/*` | `Init`, `Open`, `Repo`, `Identity`, `Auth`, `SetRemote` |
| `agent/tools` | The common, wiki-agnostic tool library: the `Tool`/`Registry` extension points, the `FS`/`OSFS` seam, shared arg/path/truncation helpers, and the generic file/search/get_time/git tools plus the host-seam health and conversation-memory tools (`HealthFunc`, `ConversationStore`, `system_health`, `list_conversations`) | `Tool`, `Registry`, `FS`, `OSFS`, `WalkFiles`, `ReadFile`, `WriteFile`, `List`, `Grep`, `Search`, `GitOptions`, `HealthFunc`, `ConversationStore` |
| `internal/agent` | The Thoth host on the `agent` library — the only Thoth-aware code that talks to it; the drop-in chat `Client` seam the Hub depends on | `Client`, `New`, `SystemPrompt`, `History`, `RegistryOptions` |
| `internal/agent/tools` | The wiki-specific agent tools: note authoring, todos, inbox, memory, tags, tree, recents — built on the `agent/tools` FS seam and the `internal/wiki` note contract | `WriteNote`, `ReadNote`, `ListTree`, `GetTodos`, `Remember` |
| `internal/wiki` | The file contract: scaffolding, parsing, path safety, tree | `Scaffold`, `ParseNote`, `SafePath`, `Wiki`, `Rulebook` |
| `internal/index` | SQLite + FTS5 + watcher | `Index`, `Sync`, `Watch`, `Search` |
| `internal/assets` | Static data files served by the API (embedded) | `models.json` → `ModelOptions` (the llm_models seed) |
| `internal/store` | Conversations, messages, and the llm_models registry (same db file) | `Store` |
| `internal/api` | Echo server: routes, WS hub, handlers | `Deps`, `New`, `Hub` |
| `internal/config` | Localhost bind constants (`127.0.0.1:8333`) + `ExpandHome` path helper | `DefaultHost`, `DefaultPort`, `ExpandHome` |
| `internal/doctor` | Shared install checks (the `thoth doctor` CLI and the Settings → Doctor tab run the same suite) | `Check`, `Run` |
| `internal/github` | GitHub identity (PAT storage) — the sync itself lives in `internal/api` | `Client`, `Auth`, `Repo`, `Service` |
| `internal/settings` | The settings KV table (thoth.db) — the single source for user-facing settings | `Repo`, `OpenRepo` |
| `internal/webui` | Embedded frontend | `Register` |

## agent/ — the reusable native agent library

The repo-root `agent` module is the provider-agnostic core of Thoth's native agent, organized so hosts and concrete providers import only what they need:

- **Root package (`agent.go`)** — `Agent`, the tool-use loop, plus re-exports of the public API. `New(Options)` fails when `Provider` or `Model` is missing. `Options` carries `Provider`, `Model`, `System`, `History` (prior messages of a conversation, capped to `HistoryCap` turns), `Tools` (the `*tools.Registry` the loop resolves `tool_use` calls through), `MaxIterations` (bounds provider turns; default 25 — a loop whose model keeps requesting tools past the cap terminates with an explicit error event instead of hanging), `MaxOutputTokens`, and `Logger`. An `Agent` is single-turn and not safe for concurrent `Start` calls — hosts build a fresh one per turn
- `events` — the normalized turn events a host forwards to its UI: `assistant_delta`, `tool_activity`, `thinking`, `turn_done`, `error`. The type names and fields are stable public API — the WS protocol and the frontend depend on them
- `model` — the normalized message/block/delta data model and the streaming `Builder` that accumulates deltas into the message the loop acts on
- `provider` — the wire-layer seam every model provider implements: `Provider.Stream(ctx, Request) (Stream, error)`. The loop knows nothing about a vendor's wire format — it sends one normalized `Request` (system, messages, tools, max tokens) and consumes normalized deltas from the returned `Stream`; `Accumulate` consumes a stream to a completed `Response` and always closes it. Concrete providers live as subpackages: `agent/provider/anthropic` and `agent/provider/openai` (OpenAI-compatible), both reading SSE through `agent/transport`
- `tools` — the reusable `Tool` extension point (stable `Name`/`Description`/`Schema`, executed by `Run`), the `Registry` (register once at startup, then read), and the **common** root-bounded file tools over an `FS` seam: `read_file`, `write_file`, `list`, `edit_file`, `append_file`, `rename_file`, `delete_file`, `grep`, `get_time`, plus `search` over a host-injected `SearchFunc`. `OSFS` binds every relative path to a root and rejects escapes; hosts inject their own `FS` to add further constraints. Wiki-specific tools (note authoring, todos, inbox, memory) live in `internal/agent/tools` — this library stays wiki-agnostic
- `git` — the pure-Go wrapper over go-git the git tools run on: `Init(path)` (idempotent — opens first, initializes a `main` repo only when absent), `Open`, `Status` (git-status-short), `Log(n)`, `Diff` (working-tree unified diff), `CommitAll(msg, identity)`, `SetRemote(url)` (adds or replaces the `origin` remote), `Push(auth)` (origin, over `file://` or https), and `Head`. Identity and auth are always parameters — hosts inject the committer and a lazy auth, so a token is never stored or logged
- `tools/git.go` — the git tools on top of that backend, configured by injected funcs (`GitOptions{RepoPath, Guard, Auth, Identity}`): `git_init` (idempotent local init), `git_status`, `git_log`, `git_diff` (read-only; clean "not a git repository" message when the workspace is unversioned), `git_commit` and `git_push` (guarded by the host's `Guard` — commit/push only after sync is enabled; both auto-init when absent). `RepoPath` is evaluated per call so the tools follow a live root; `Auth` and `Identity` are lazy — the push token is held only for the call
- `tools/health.go` — the `system_health` tool over a host-injected `HealthFunc func(ctx) ([]HealthReport, error)`; `HealthReport{Name, OK, Message}` is the host-agnostic shape of one check. Thoth backs it with `doctor.Run`, so the agent runs the same install checks as `thoth doctor`. Read-only self-diagnosis
- `tools/conversations.go` — the conversation-memory tools over the narrow `ConversationStore{ListConversations, Messages}` seam (with `Conversation`/`ConversationMessage` types defined here so the library stays `internal/*`-free): `list_conversations` (optionally filtered by title `q`), `get_conversation` (one conversation's transcript, most recent messages capped), and `search_conversations` (titles + message content, with a snippet of the matching message). Read-only recall; Thoth adapts `internal/store`

### The tool-use loop

```mermaid
flowchart TB
    START[Start turn] --> REQ[provider request: system + history + prompt + tools]
    REQ -->|streams deltas| EV[normalized events to the host]
    EV --> TOOL{model requests a tool?}
    TOOL -->|no| DONE[turn ends]
    TOOL -->|yes| RUN[Registry resolves name → runnable Tool]
    RUN --> RESULT[feed tool_result back]
    RESULT --> REQ
```

`FakeClient` in `internal/agent` replays scripted events and records calls — every chat consumer's tests use it, so no test ever touches a real provider.

The library replaced the spawned Claude Code CLI (epic #121): `internal/claude` — the flag lists, stream parsing, `persistent.go`'s process pool, and the build-tagged `proc_unix.go`/`proc_windows.go` kill mechanics — was deleted wholesale, so a CLI upgrade can no longer break the chat path.

## internal/agent — the Thoth Agent host

The only Thoth-aware code that talks to the reusable `agent` library (imported as `agentlib`). `Client` implements the chat seam the Hub depends on — `Start(ctx, sessionID, prompt, w)` — by driving `agent.Agent` instead of a CLI subprocess; `sessionID` is the conversation id for history lookup.

- `New(model, apiKey, wiki, store, index, opts…)` wires everything: the provider is picked from the model's provider name (`WithProviderConfig(name, baseURL)` — "Anthropic" → Anthropic, every other name → OpenAI-compatible pointed at its base URL; an empty name falls back to the model id prefixes `claude-*`/`gpt-*`; `WithProvider` overrides for tests), and the tools are built from the wiki.
- `Start` builds a fresh `agent.Agent` per turn: the system prompt is re-read from `wiki/CLAUDE.md` (the user-editable rulebook) so edits apply without restart, and the loop is single-turn while the Hub serves many conversations at once.
- `tools.go` — `wikiFS` implements the agent `tools.FS` seam (read/write/mkdir/list/stat/remove/rename), resolving every path through `wiki.SafePath` so nothing can escape the wiki; `search` runs over the FTS `index.Index`; `registry(RegistryOptions{Wiki, Index, TodosPath, InboxDir, MemoryPath, Git, Health, Conversations, CustomTools})` registers the common `agent/tools` catalog plus the wiki-specific tools below, all over the live wiki root. `Git` wires the git tools to the wiki when it carries a `RepoPath`; `Health` adds `system_health` when non-nil; `Conversations` adds the conversation-memory tools when non-nil; `CustomTools` lets a host extend the catalog with its own tools.
- `tools/` — the Thoth-wiki-specific tools: `write_note`, `read_note`, `list_tree`, `list_recent`, `search_by_tag`, `get_todos`, `update_todos`, `get_inbox`, `file_inbox`, `remember`. They share the note contract with the wiki via `wiki.ParseNote`/`wiki.FormatNote` (single source) and the folder-to-type rule via `wiki.NoteType`; configurable todo/inbox/memory paths default to the scaffolded layout. `Client.WithTools(...)` registers host custom tools.
- `system.go` — `SystemPrompt(w, folders)` reads the rulebook per call, falling back to `RulebookFor(folders)` when absent.
- `history.go` — `History(store)` maps `store.Messages` (roles user/assistant) to agent messages; the trailing user message (the in-flight prompt the loop appends) is dropped, and the loop caps the rest (`HistoryCap`).
- `host.go` — the injected-seam adapters: `DoctorHealth(dir)` wraps `doctor.Run` as the `tools.HealthFunc` behind `system_health`, and `conversationStore` adapts `store.Store` to the `tools.ConversationStore` seam behind the conversation-memory tools. `Client.New` wires the conversation tools from the store automatically; `WithHealthFunc` (serve passes `DoctorHealth(dir)`) opts the health tool in.

Wired into `serve` at boot — the Hub's `Client` is this host, so the whole chat path is in-process.

## internal/wiki — the file contract

- `Scaffold(dir)` — creates the 9 folders + rulebook; never overwrites an existing `CLAUDE.md`
- `EnsureGitRepo(root)` — pure-Go `agent/git.Init` (plus the same `.gitignore` the scaffold writes) so a pre-existing-but-unversioned wiki becomes versioned on startup; a no-op when already a repository
- `Rulebook()` — the single source of the rulebook text (embedded template); the frontmatter `type:` list is derived from the folder set (`NoteTypesFor`), so it can never drift
- `ParseNote(content)` — splits frontmatter, requires `title`, returns `NoteMeta` + body
- `Validate(rel, content)` — save-protocol checks (frontmatter, `type` matches the folder, kebab-case filename, date-prefix in `meetings/`/`daily/`); advisory problems, never fatal — the index logs them and the doctor's "malformed" check surfaces parse failures by name
- `SafePath(root, rel)` — rejects absolute paths and `..` escapes, then resolves symlinks (deepest existing ancestor) and rejects any real target outside root; every filesystem access routes through it
- `Wiki` — `New`, `Exists`, `Read`, `Tree` (dirs first, dotfiles and the root rulebook skipped via the shared `Visible` predicate); the root is guarded (`Root()`/`SetRoot`) so the settings wiki-path change can swap it while turns read; a directory that cannot be read keeps its node with an `Error` and no children instead of failing the whole walk (only an unreadable root errors); `Change`/`Changed` + op constants are the watcher's event-bus payload

## internal/index — search and sync

- `Open(path)` — WAL mode + schema migration
- `Upsert` / `Delete` / `DeletePrefix` — with FTS5 triggers keeping the index in sync
- `Search(q, limit)` — bm25 ranking (title 8×) and body snippets, HTML-escaped
- `Sync(root, log)` — reconciles the index with the tree in one transaction (unchanged files skipped, missing files deleted); malformed notes skipped
- `Watch(ctx, root, ix, log, opts ...WatchOption)` — fsnotify with 200 ms debounce and new-directory rescan; `WithPublisher` hooks one `wiki.Changed` batch per flush (startup publishes an empty one) for event-bus consumers; errors on the watcher's `Errors` channel are logged as structured warns (a silent drop would hide a directory that is no longer being watched)

Full mechanics: [Indexing & search](indexing.md).

## internal/api — the server

`Deps` carries every dependency; `New(d Deps) *echo.Echo` wires all routes. `Hub` owns the WebSocket lifecycle: one active turn per conversation, supersede-on-send, cancel, and a 500-message replay buffer for reconnects. It also keeps a client registry and `Broadcast`s server-push frames (non-blocking; slow clients are skipped). Clients still send `presence` frames for their tab's visibility, but Thoth Agent keeps no idle processes to flush, so they are accepted for protocol compatibility and ignored server-side. When `Deps.Events` carries the event bus (`go-warehouse/events`, built in `internal/cli/serve.go`), `newServer` subscribes and forwards every `wiki.Changed` batch to all clients as a `wiki_changed` frame. Protocol details: [API](api.md).

## internal/store

Conversations, messages, and the llm_models registry in the same `thoth.db` (separate `*sql.DB`, WAL makes sharing safe). The whole schema lives in embedded `.sql` migrations in `migrations/`, applied in filename order and gated on `PRAGMA user_version`; a single-row `app_metadata` table (enforced by `CHECK (id = 1)`) holds install facts (`installation_id`, `created_at`, seeded by `EnsureMetadata` on boot) and git sync state (`last_synced_at`, `sync_error`, written by `SetSyncResult`). IDs are valid RFC 4122 v4 UUIDs (`google/uuid`); the conversation id is the chat history key passed to the agent as `sessionID`. Timestamps are stored UTC so ordering is chronological.

`models.go` owns the `llm_models` CRUD (`ListModels`, `Model`, `CreateModel`, `UpdateModel`, `DeleteModel`); duplicate `value`s surface as the `ErrModelExists` sentinel (typed SQLite constraint code, not string matching), missing ids as `ErrModelNotFound`. Seeding is not a store method — `ensureModels` in `internal/cli` seeds from `assets.ModelOptions()` whenever the table is empty, so every startup self-heals an empty registry.

## internal/cli

`Execute()` builds the root Cobra command. `serve` is a thin orchestration function whose helpers (`thothDir`, `settleWikiPath`, `ensureWiki`, `openIndex`, `defaultModel`, `modelProvider`, `ensureModels`, `gitToolOptions`, `onSettingsSaved`, `serveUntilShutdown`) keep each step readable. `gitToolOptions` wires the agent's git tools to the live wiki root, the sync switch (the `Guard`), and the stored GitHub connection (lazy `Auth`/`Identity`); `ensureWiki` now also git-inits a pre-existing-but-unversioned wiki via `wiki.EnsureGitRepo`. Details: [CLI](cli.md).

## internal/doctor

`Run(ctx, Options{Dir, Addr, Log, HTTP, BaseURL})` runs the nine install checks (wiki, provider, api key, model, database, index, malformed, api, websocket) and returns `[]Check`, each carrying `Name`/`OK`/`Message`. The provider check probes the selected model's provider models endpoint with the resolved credential — the model's `llm_models` row names the provider, whose per-provider api key/base URL win over the shared key and provider default, the same resolution `serve` uses at boot (`modelProvider`/`providerConfig`/`providerProbeFor`). The HTTP client and base URL are injectable for tests. The dashboard's Settings → Doctor tab runs the same suite via `GET /api/doctor` (details: [CLI](cli.md)).

## internal/github

`Client` talks to api.github.com (identity + repos; the token is never sent anywhere else) and `Repo` stores the PAT and identity in the `github_auth` table; `Service` is a thin bundle of the two for API wiring. The git sync itself (init, remote, commit, push — errors sanitized) lives in `internal/api/git.go`, run on the same pure-Go `agent/git` backend the agent's git tools use, so no git binary is needed anywhere. Details: [Schema](schema.md).

## internal/settings

`Repo` (`OpenRepo(path)`) owns the `settings` KV table — `wiki_path`, `wiki_folders`, `github_sync_*` keys and the per-provider `provider_<slug>_api_key`/`provider_<slug>_base_url` keys (helpers `ProviderAPIKeyKey`/`ProviderBaseURLKey`) — with sync-state conveniences and `ProviderConfig(provider)`, the model→provider→credential resolution `serve` uses at boot. It deliberately runs no migrations and no WAL pragma: the doctor must never mutate a database it only reads. Details: [Schema](schema.md).

## internal/api — git sync

`internal/api/git.go` (`POST /api/git/setup`) is the host's wiki sync, run entirely on `agent/git`: it inits a repo in the wiki root when needed (`Init`), points `origin` at the requested URL (`SetRemote`), commits the tree (`CommitAll`; a clean tree is "nothing to commit", not an error), and pushes (`Push`) with the stored GitHub token as BasicAuth. The committer identity also comes from the `github_auth` row (display name falling back to username), so a connected account is required — there is no git-binary path, no credential-helper or SSH-agent fallback. Every step reports a fixed sanitized error that never echoes the URL or token. Details: [API](api.md), [Security](security.md).
