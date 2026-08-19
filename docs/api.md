# API

The server exposes REST for everything except the live chat and server-push notifications (`wiki_changed`), which are WebSockets. All routes are registered in `internal/api/server.go`.

## REST endpoints

| Method + Path | Request | Response |
|---|---|---|
| `GET /api/health` | — | `{status, claude:{found,path}, wiki:{path,exists}, version, dev, commit, default_wiki_path}` — `dev` is true under `serve --dev` (the UI shows a warning banner); `commit` is the full git commit id the dev server runs from (empty otherwise); `default_wiki_path` is the mode's wiki default in tilde form (`~/.thoth/wiki`, or `~/.thoth/dev/wiki` in dev) — the settings hint reads it |
| `GET /api/search?q=&limit=` | q required; limit default 20, clamped 1–100 | `{results:[{path,title,kind,snippet}]}` — snippet is HTML-escaped with safe `<mark>` highlights |
| `GET /api/notes?path=` | wiki-relative path | `{path, content}` |
| `GET /api/wiki/tree` | — | `{nodes:[{name,path,is_dir,children}]}` |
| `GET /api/fs/dirs?path=` | absolute directory path | `{dirs:[…]}` — the immediate subdirectories in lexical order, powering the Settings directory picker; 400 when the path is missing or not a readable directory |
| `GET /api/doctor` | — | `{checks:[{name, ok, message}]}` — the shared `internal/doctor` suite (same checks as `thoth doctor`) |
| `GET /api/settings` | — | `{wiki_path, wiki_folders, model, has_api_key, repo_url, sync_enabled}` — every value lives in the `settings` key/value table in thoth.db (config.toml is deprecated); `wiki_folders` is the configured scaffold folder set (`[]` when unset → defaults); the api key itself is never returned, only whether one is stored |
| `PUT /api/settings` | `{wiki_path, wiki_folders, model, api_key?, repo_url, sync_enabled}` | saved values (KV upserts; the wiki-path-change callback runs first); 400 when `wiki_path` is empty; an empty or omitted `api_key` leaves the stored key unchanged |
| `GET /api/models` | — | `{models:[{id, value, name, description, provider}]}` — the llm_models table (seeded from `internal/assets/models.json` on first boot, then user-editable); the chosen `value` feeds the `model` setting |
| `POST /api/models` | `{value, name, description?, provider?}` | the created model with its `id`; 400 when `value`/`name` are empty, 409 when the value is taken |
| `PUT /api/models/:id` | `{value, name, description?, provider?}` | the updated model; 400/404/409 as above. Renaming the selected model's value moves the `model` setting to it |
| `DELETE /api/models/:id` | — | `{ok:true}`; 404 when the model is missing. Deleting the selected model clears the `model` setting (next turn uses the CLI default) |
| `POST /api/github/auth` | `{token}` | identity `{username, display_name, email, avatar_url, scopes}` — the token itself is never returned; 400 "token is required" / "github rejected the token" |
| `GET /api/github/auth` | — | identity (all fields empty when not connected) |
| `DELETE /api/github/auth` | — | `{ok:true}` (idempotent) |
| `GET /api/github/repos` | — | `{repos:[{full_name, clone_url}]}` — the connected account's repos (fetched with the stored token; empty when not connected; 400 "github rejected the token" when revoked) |
| `GET /api/conversations` | — | `{conversations:[{id,title,created_at}]}` |
| `POST /api/conversations` | `{title}` | `{id,title}` |
| `GET /api/conversations/:id` | — | `{conversation, messages:[…]}` |
| `DELETE /api/conversations/:id` | — | `{ok:true}` — removes the conversation and its messages (idempotent) |
| `POST /api/git/setup` | `{url}` | `{ok:true}` or `{ok:false, error}` — inits a repo in the wiki dir if needed, points `origin` at `url`, commits the tree, and pushes `HEAD`; each git command has a 15 s timeout, errors are sanitized (never echo credentials/URLs) |

**Errors:** JSON `{"error":"<msg>"}` — 400 for client errors, 404 not found, 500 always the generic `{"error":"internal error"}` (details go to the server log only).

**Logging:** every `/api/*` request is logged at Info level with method, path, status, and duration (`internal/api/logging.go`) — the source of truth for latency investigations. Errors carry the error text; SPA assets and `/ws` are not logged.

**SPA deep links:** `/chat/<conversation-id>` serves the app shell (index.html fallback in `internal/webui`), which loads and pins that conversation; unknown `/api/*` paths stay JSON 404s.

## WebSocket chat (`/ws`)

One socket per browser tab. The protocol is small and typed on both sides (`internal/api/chat.go` ↔ `web/src/ws/chat.ts`):

| Direction | Frames |
|---|---|
| client → server | `{"type":"send","text":…}` · `{"type":"cancel"}` · `{"type":"resume","conversation_id":…}` · `{"type":"open","conversation_id":…}` · `{"type":"new_chat"}` |
| server → client | `assistant_start` · `assistant_thinking {text}` · `assistant_delta {text}` · `tool_activity {tool, detail}` · `turn_done {conversation_id}` · `wiki_changed {changes:[{op,path}]}` · `error {message}` |

```mermaid
sequenceDiagram
    participant UI as Browser
    participant Go as Go server
    participant CC as Claude CLI
    participant W as Wiki dir

    UI->>Go: send {text}
    Go->>Go: create/find conversation, persist user msg
    Go->>UI: assistant_start
    Go->>CC: first turn: spawn claude -p --input-format stream-json (cwd=wiki, session-id)
    Go->>CC: later turns: control message over the process's stdin
    CC->>W: read CLAUDE.md + notes
    CC-->>Go: stream-json lines
    Go-->>UI: assistant_delta ×N (+ tool_activity)
    CC->>W: write notes (when saving)
    CC-->>Go: result
    Go->>Go: persist assistant msg
    Go->>UI: turn_done {conversation_id}
    Note over W,Go: fsnotify reindexes + publishes to the event bus within ~200ms
    Go->>UI: wiki_changed {changes} (broadcast; UI refetches the tree)
    Note over UI: GET /api/wiki/tree
```

**Semantics:**

- **Supersede** — a new `send` while a turn runs cancels the in-flight turn and starts the next
- **Cancel** — kills the conversation's pooled CLI process (there is no per-turn interrupt in the plain CLI); the UI receives `error {message:"cancelled"}` and nothing is persisted for that turn. The next send respawns the process, resuming the session from disk
- **Resume** — after a reconnect, the client sends `resume`; the server replays the last turn's frames (≤ 500-message ring), then continues live
- **Open** — pins the connection to an existing conversation so the next `send` continues it (conversation-history load). No replay, no other effect; an unknown id gets `error {message:"unknown conversation"}` and the connection stays unpinned
- **New chat** — unpins the connection (cancels an in-flight turn first, which emits `error {message:"cancelled"}`); the next `send` starts a fresh conversation
- **Sessions** — every conversation stores its Claude CLI session id (`conversations.claude_session_id`, seeded as the conversation id, migration 0003 backfills legacy rows) and owns one long-lived CLI process, lazily spawned and evicted after 10 idle minutes (the pool lives in `internal/claude/persistent.go`). A turn reusing a session id the CLI reports as "already in use" (stale lock from a killed process) forks once into a fresh id via `--resume <old> --fork-session` and persists the fork
- **Titles** — derived from the first message, truncated at 60 runes
- **Origins** — only localhost origins are accepted on the upgrade (see [Security](security.md))
- **Wiki changes** — the index watcher publishes each 200 ms debounce batch to the in-process event bus (`go-warehouse/events`); the server broadcasts `wiki_changed {changes:[{op,path}]}` (op: `create|write|remove|rename`, wiki-relative path; only paths the tree displays) to every connected client, which refetches `GET /api/wiki/tree`. A watcher (re)start publishes an empty batch so a wiki-path change in Settings also refreshes the tree. Broadcasts are non-blocking: a client with a full write buffer misses the frame and recovers on its next reconnect/focus refetch; `wiki_changed` frames are not replayed on resume
