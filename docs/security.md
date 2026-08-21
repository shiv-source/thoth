# Security

Thoth's security model is simple by design: **local-only, single-user, no authentication — and the boundary is enforced in code, not by assumption.** Everything below assumes an attacker can serve a webpage in the user's browser and try to talk to `127.0.0.1:8333`.

## Threat model

| Threat | Mitigation |
|---|---|
| Remote access to the server | Binds `127.0.0.1` on port 8333 — the address is a code constant, not a setting |
| Malicious webpage driving the WebSocket (WebSockets bypass CORS) | Origin check on upgrade: only localhost origins accepted |
| Path traversal reading arbitrary files | Every filesystem access routes through `SafePath` — absolute paths and `..` escapes rejected |
| Stored XSS via note content | Snippets are HTML-escaped server-side before `<mark>` highlighting; markdown rendering escapes HTML |
| Secrets leaking into the wiki | The rulebook forbids secrets (placeholders only) |
| Secrets leaking out of the settings API | The API key is write-only: `GET /api/settings` reports `has_api_key` only, never the key |
| Orphaned Claude processes | Cancel kills the process group; shutdown cancels all in-flight turns |
| Error details leaking internals | 500s return a generic body; details go to the server log only |

## Implementation pointers

- **Bind default** — `127.0.0.1:8333` constants in `internal/config/config.go` (the address is not user-configurable)
- **Origin check** — `allowLocalOrigin` in `internal/api/chat.go`: accepts a missing Origin header (curl, tests, non-browser clients) or hostnames `localhost` / `127.0.0.1` / `::1`
- **Path safety** — `internal/wiki/path.go` `SafePath`, used by `Wiki.Read` and the `/api/notes` handler; the directory picker (`GET /api/fs/dirs`) lists subdirectories only — it never returns file contents and is bound by the same localhost-only origin rules
- **Snippet escaping** — `html.EscapeString` then control-marker → `<mark>` replacement in `internal/index/index.go`, covered by `TestSearchSnippetEscapesHTML`
- **Process hygiene** — process-group kill on unix (`internal/claude/proc_unix.go`), direct-child kill on windows; in-flight turns cancelled on SIGTERM before shutdown completes
- **Secrets policy** — the wiki rulebook (`internal/wiki/templates/CLAUDE.md`) plus the repo-level rule in the root `CLAUDE.md`

## Deliberate trade-offs

- No authentication — correct for a localhost-bound single-user app; the origin check closes the realistic browser-based attack, not network access in general
- The spawned Claude CLI runs **fully unattended by default** (`--dangerously-skip-permissions` — bypasses all permission checks; headless mode cannot answer prompts and note-saving is the app's core feature). The CLI flags are fixed in `internal/claude/client.go` (the blast wall)
- Conversation history lives in the same local SQLite file as the index; nothing leaves the machine
- The GitHub PAT is stored plaintext in `thoth.db` (`github_auth`) — the same trust model as the `gh` CLI's own credentials file; the API never returns it and errors never echo it, and it is only ever sent to `api.github.com`
- The API keys are stored plaintext in the `settings` table — the shared `api_key` plus the per-provider `provider_<slug>_api_key` keys — the same local-file trust model as the PAT; the native agent sends them only to the model provider's endpoint on this machine, and the API never returns them (`has_api_key` only, and `PUT` with an empty key leaves the stored key unchanged)
