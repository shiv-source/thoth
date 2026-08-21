# Migrating to Thoth Agent

Thoth's assistant was born as a **Claude Code CLI process** spawned headlessly per conversation. Epic #121 replaced it with **Thoth Agent** — a native, in-process assistant written in Go. This page explains why the migration happened, what changed, and what it means for you.

## What is Thoth Agent?

Thoth Agent is the built-in assistant: the thing that answers questions from your wiki and files new notes into it. It is a pure library call inside the `thoth` binary — the reusable `agent/` module (the tool-use loop, the `Provider` seam, the tool registry) hosted by `internal/agent`, which wires it to the wiki, the store, and the search index. It reports itself as `thoth-agent` in `GET /api/health`.

## What changed

| | Before (Claude Code CLI) | After (Thoth Agent) |
|---|---|---|
| **Process model** | A long-lived `claude` CLI process per conversation, lazily spawned and pooled | None — every turn is an in-process function call |
| **Provider** | Fixed to Anthropic via the CLI's own login and wire format | A `Provider` seam: Anthropic or any OpenAI-compatible endpoint (DeepSeek, Qwen, GLM, Grok, …) |
| **Cancellation** | Kill the process group (build-tagged `proc_unix.go`/`proc_windows.go`); the next turn respawns | Cancel the turn's context — the provider stream aborts, nothing lingers |
| **System prompt** | Read at process spawn | Re-read from `wiki/CLAUDE.md` on every turn, so rulebook edits apply without restart |
| **Setup** | Requires the `claude` binary installed, logged in, and on `PATH` | Only a model provider API key in Settings |
| **Write access** | The CLI ran with `--dangerously-skip-permissions` (fully unattended headless) | A full FS-backed tool catalog bounded to the wiki by `SafePath`: common file tools (`read_file`/`write_file`/`list`/`list_tree`/`grep`, `edit_file`/`append_file`/`rename_file`/`delete_file`, `get_time`, and `search` over the FTS index) plus wiki tools (`write_note`/`read_note`, `list_recent`/`search_by_tag`, `get_todos`/`update_todos`/`get_inbox`/`file_inbox`/`remember`). Common tools live in `agent/tools`; wiki-specific ones in `internal/agent/tools`, and hosts can register custom tools |

The pieces that made the CLI approach work were deleted wholesale: the flag lists, the stream-json parsing, the `persistent.go` process pool, and the process-kill mechanics.

## Why we did it

The CLI worked, but it made the chat path a foreign-body dependency:

- **An external runtime we don't control.** Every conversation paid for a subprocess we spawned, managed, and killed. A `claude` upgrade — new flags, changed stream output, a renamed session command — could break the app with no change on our side.
- **A heavyweight lifecycle to babysit.** Lazy-spawn one process per conversation, cap the pool, evict idle processes, flush on wiki-path change, pre-warm the most recent conversation at boot. All of that existed to work around a long-lived process; none of it belongs in a local-first app whose real asset is a markdown directory.
- **Anthropic-only.** The CLI spoke one wire format to one provider, so the model picker could only ever offer Claude models.
- **Slow and opaque to cancel.** A turn was only as interruptible as a process-group kill, and "already in use" session locks needed a fork dance. Rulebook edits required restarting the process.
- **Permission model by blunt instrument.** Headless mode cannot answer prompts, so note-writing ran with `--dangerously-skip-permissions` — everything or nothing. There was no narrower write path.

## The advantages

**For you (user-facing):**

- **Nothing to install or log in** — the `claude` binary, its login, and `PATH` setup are gone. Configure an API key in Settings and turn.
- **Any provider, not just Claude** — pick any model in Settings → LLM Models; each provider can carry its own API key and base URL.
- **Faster, cleaner turns** — no per-conversation process boot, no process to warm; a turn starts immediately.
- **A real, bounded write path** — the assistant writes only through `SafePath`-bounded tools. There is no "skip permissions" mode because the write path *is* the wiki.
- **Rulebook edits apply instantly** — the system prompt is read fresh each turn, no restart needed.

**For the codebase:**

- **One less moving part** — no subprocess lifecycle, no process-group kills, no session-id bookkeeping; the `claude_session_id` column is retained for schema stability but never written.
- **A clean seam to extend** — the `Provider` interface and the `tools.Registry` are the extension points; adding a provider is a new `agent/provider/*` subpackage, adding a capability is a new `Tool`.
- **Cancel is just context** — supersede, cancel, and shutdown all cancel a turn's context; the loop is bounded by `MaxIterations`, so nothing can hang.
- **Cross-compiles stay simple** — the platform-specific process-kill files are gone; the binary has no build-tagged kill mechanics left.

## What it means for you

- **First run** — `thoth serve`, then add a provider API key in Settings. `thoth doctor` verifies the provider, API key, model, and everything else.
- **Doctor checks changed** — the old `claude` binary/login check became `provider` + `api key` checks: the provider check probes the selected model's provider endpoint with the resolved credential.
- **Existing data is untouched** — your wiki, conversations, and settings carry over as-is. The `claude_session_id` column stays (unused); the shared `api_key` still works as the fallback credential, and per-provider keys win when set.
- **The wiki is still yours** — Thoth Agent and a terminal (Claude Code, any editor) read and write the same tree under the same rulebook. The two-interfaces contract did not change.

See [Architecture](architecture.md) for the system design, [Components](components.md) for the package layout, and [What's new](whats-new.md) for the milestone overview.
