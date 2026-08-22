# Security

Thoth's security model is simple by design: **local-only, single-user, no authentication — and the boundary is enforced in code, not by assumption.** Everything below assumes an attacker can serve a webpage in the user's browser and try to talk to `127.0.0.1:8333`.

## Threat model

| Threat | Mitigation |
|---|---|
| Remote access to the server | Binds `127.0.0.1` on port 8333 — the address is a code constant, not a setting |
| Malicious webpage driving the WebSocket (WebSockets bypass CORS) | Origin check on upgrade: only localhost origins accepted |
| Path traversal reading arbitrary files | Every filesystem access routes through `SafePath` — absolute paths, `..` escapes, and symlinks resolving outside the wiki root are rejected |
| Stored XSS via note content | Snippets are HTML-escaped server-side before `<mark>` highlighting; markdown rendering escapes HTML |
| Secrets leaking into the wiki | The rulebook forbids secrets (placeholders only) |
| Secrets leaking out of the settings API | The API key is write-only: `GET /api/v1/settings` reports `has_api_key` only, never the key |
| Runaway agent turns / orphaned provider streams | Cancel, supersede, and server shutdown cancel the turn's context, aborting the provider stream; the tool loop is bounded by `MaxIterations` so a model stuck requesting tools ends with an explicit error instead of hanging, and every turn is bounded by a timeout so a provider that stalls on the wire returns an error instead of hanging the socket |
| Turn content leaving the machine | The prompt, history, and tool results go only to the model provider configured in Settings (localhost single-user app) — never anywhere else; the never-store-secrets rulebook keeps credentials out of prompts |
| Error details leaking internals | 500s return a generic body; details go to the server log only |

## Implementation pointers

- **Bind default** — `127.0.0.1:8333` constants in `internal/config/config.go` (the address is not user-configurable)
- **Origin check** — `allowLocalOrigin` in `internal/api/v1/chat.go`: accepts a missing Origin header (curl, tests, non-browser clients) or hostnames `localhost` / `127.0.0.1` / `::1`
- **Path safety** — `internal/wiki/path.go` `SafePath`, used by `Wiki.Read`, the `/api/v1/notes` handler, and the agent's wiki file tools: it rejects absolute paths and `..` escapes syntactically, then resolves the deepest existing ancestor with `EvalSymlinks` and rejects any real target outside the wiki root — so a symlink inside the wiki cannot read or write through to the rest of the machine. Writes are atomic (temp file + rename, `agent/tools.AtomicWrite`); the directory picker (`GET /api/v1/fs/dirs`) lists subdirectories only — it never returns file contents and is bound by the same localhost-only origin rules
- **Snippet escaping** — `html.EscapeString` then control-marker → `<mark>` replacement in `internal/index/index.go`, covered by `TestSearchSnippetEscapesHTML`
- **Turn lifecycle** — every turn runs on a context derived from the Hub's (`internal/api/v1/chat.go`): cancel, supersede, and server shutdown cancel it, which aborts the provider stream; the tool loop is capped by `agent.MaxIterations`, and `internal/agent` bounds each turn with `WithTurnTimeout` (default 10 minutes), so no turn or subprocess can hang the server or the client socket
- **Secrets policy** — the wiki rulebook (`internal/wiki/templates/CLAUDE.md`) plus the repo-level rule in the root `CLAUDE.md`

## Deliberate trade-offs

- No authentication — correct for a localhost-bound single-user app; the origin check closes the realistic browser-based attack, not network access in general
- The assistant writes only through `wiki.SafePath`-bounded tools (`read_file`, `write_file`, `list`, `search` over the wiki root, built by `internal/agent`) — symlink-proof and atomic — there is no "skip permissions" mode because the write path *is* the wiki, period; a prompt can never reach the filesystem outside it
- Conversation history lives in the same local SQLite file as the index; nothing leaves the machine except the turn's prompt/history/tool results, which go to the configured model provider over its API
- The GitHub PAT is stored plaintext in `thoth.db` (`github_auth`) — the same trust model as the `gh` CLI's own credentials file; the API never returns it and errors never echo it, and it is only ever sent to `api.github.com`. The git sync (`POST /api/v1/git/setup`, `internal/api/v1/git.go`) runs on the pure-Go `agent/git` backend (no git binary) and authenticates pushes with this stored token as BasicAuth — the token goes only to the configured remote's host during a push. SSH remotes and credential-helper/SSH-agent flows are no longer honored: a sync requires the GitHub token. Errors are sanitized fixed messages that never include the remote URL or the token
- The API keys are stored plaintext in the `settings` table — the per-provider `provider_<slug>_api_key` keys — the same local-file trust model as the PAT; Thoth Agent sends them only to the model provider's endpoint on this machine, and the API never returns them (`has_api_key` only, and `PUT` with an empty key leaves the stored key unchanged)
