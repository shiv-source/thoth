# API

The server exposes REST for everything except the live chat, which is a WebSocket. All routes are registered in `internal/api/server.go`.

## REST endpoints

| Method + Path | Request | Response |
|---|---|---|
| `GET /api/health` | — | `{status, claude:{found,path}, wiki:{path,exists}}` |
| `GET /api/search?q=&limit=` | q required; limit default 20, clamped 1–100 | `{results:[{path,title,kind,snippet}]}` — snippet is HTML-escaped with safe `<mark>` highlights |
| `GET /api/notes?path=` | wiki-relative path | `{path, content}` |
| `GET /api/wiki/tree` | — | `{nodes:[{name,path,is_dir,children}]}` |
| `GET /api/doctor` | — | `{checks:[{name, ok, message}]}` — the shared `internal/doctor` suite (same checks as `thoth doctor`) |
| `GET /api/settings` | — | config fields plus `repo_url` (stored in the DB's `github_auth` row, not config.toml) |
| `PUT /api/settings` | config fields plus `repo_url` | saved config; `repo_url` persists to the DB (400 "connect GitHub first" when non-empty and no account is connected); validates `wiki_path`/`host` non-empty, port 1–65535 |
| `POST /api/github/auth` | `{token}` | identity `{username, display_name, email, avatar_url, scopes}` — the token itself is never returned; 400 "token is required" / "github rejected the token" |
| `GET /api/github/auth` | — | identity (all fields empty when not connected) |
| `DELETE /api/github/auth` | — | `{ok:true}` (idempotent) |
| `GET /api/github/repos` | — | `{repos:[{full_name, clone_url}]}` — the connected account's repos (fetched with the stored token; empty when not connected; 400 "github rejected the token" when revoked) |
| `GET /api/conversations` | — | `{conversations:[{id,title,created_at}]}` |
| `POST /api/conversations` | `{title}` | `{id,title}` |
| `GET /api/conversations/:id` | — | `{conversation, messages:[…]}` |
| `POST /api/git/setup` | `{url}` | `{ok:true}` or `{ok:false, error}` — inits a repo in the wiki dir if needed, points `origin` at `url`, commits the tree, and pushes `HEAD`; each git command has a 15 s timeout, errors are sanitized (never echo credentials/URLs) |

**Errors:** JSON `{"error":"<msg>"}` — 400 for client errors, 404 not found, 500 always the generic `{"error":"internal error"}` (details go to the server log only).

**SPA deep links:** `/chat/<conversation-id>` serves the app shell (index.html fallback in `internal/webui`), which loads and pins that conversation; unknown `/api/*` paths stay JSON 404s.

## WebSocket chat (`/ws`)

One socket per browser tab. The protocol is small and typed on both sides (`internal/api/chat.go` ↔ `web/src/ws/chat.ts`):

| Direction | Frames |
|---|---|
| client → server | `{"type":"send","text":…}` · `{"type":"cancel"}` · `{"type":"resume","conversation_id":…}` · `{"type":"open","conversation_id":…}` |
| server → client | `assistant_start` · `assistant_delta {text}` · `tool_activity {tool, detail}` · `turn_done {conversation_id}` · `error {message}` |

```mermaid
sequenceDiagram
    participant UI as Browser
    participant Go as Go server
    participant CC as Claude CLI
    participant W as Wiki dir

    UI->>Go: send {text}
    Go->>Go: create/find conversation, persist user msg
    Go->>UI: assistant_start
    Go->>CC: spawn claude -p --verbose (cwd=wiki, session-id)
    CC->>W: read CLAUDE.md + notes
    CC-->>Go: stream-json lines
    Go-->>UI: assistant_delta ×N (+ tool_activity)
    CC->>W: write notes (when saving)
    CC-->>Go: result
    Go->>Go: persist assistant msg
    Go->>UI: turn_done {conversation_id}
    Note over W,Go: fsnotify reindexes within 200ms
```

**Semantics:**

- **Supersede** — a new `send` while a turn runs cancels the in-flight turn and starts the next
- **Cancel** — kills the CLI process group; the UI receives `error {message:"cancelled"}` and nothing is persisted for that turn
- **Resume** — after a reconnect, the client sends `resume`; the server replays the last turn's frames (≤ 500-message ring), then continues live
- **Open** — pins the connection to an existing conversation so the next `send` continues it (conversation-history load). No replay, no other effect; an unknown id gets `error {message:"unknown conversation"}` and the connection stays unpinned
- **Titles** — derived from the first message, truncated at 60 runes
- **Origins** — only localhost origins are accepted on the upgrade (see [Security](security.md))
