# CLI

All commands live in `internal/cli` (Cobra). The binary entrypoint is `cmd/thoth/main.go`.

## Commands

### `thoth serve`

Starts the app on `127.0.0.1:8333` (default). Flags:

- `--dev` — bind the dev port (`127.0.0.1:8334`, `config.DevPort`) and isolate all data under `~/.thoth/dev/` (its own `thoth.db`, default wiki `~/.thoth/dev/wiki/`, and stream dump), so a running instance keeps 8333 and its data; `make dev` uses this. Vite's proxy follows via the `THOTH_PORT` env var. At boot the dev server rewrites the seeded prod wiki default to the dev wiki and reports `dev: true` plus the checkout's full commit id in `/api/health`; the UI shows a warning banner with that commit.

Startup sequence:

1. Open `thoth.db` (migrations) and read `wiki_path` (and the optional `model`) from the settings table
2. Scaffold the wiki if it doesn't exist
3. Open `thoth.db` (index + store), sync the search index with the tree
4. Start the fsnotify watcher
5. Resolve the claude binary (config → `PATH` → warn)
6. Serve; SIGINT/SIGTERM → graceful shutdown (cancels in-flight Claude turns, then exits)

### `thoth init [path]`

Scaffolds a wiki directory — the 8 folders plus the `CLAUDE.md` rulebook. Defaults to `~/.thoth/wiki`. Never overwrites an existing rulebook.

```sh
thoth init                    # ~/.thoth/wiki
thoth init ~/notes            # custom location
```

### `thoth version`

Prints `thoth <version>` (`dev` in development builds).

### `thoth doctor`

Runs nine health checks and reports each. The checks live in the shared `internal/doctor` package — the dashboard's Settings → Doctor tab runs the same suite over `GET /api/doctor`:

| Check | What it verifies |
|---|---|

| wiki | wiki exists with all 8 folders + `CLAUDE.md` |
| claude | binary found; `claude --version` works; login status confirmed |
| api key | an API key is stored in the settings table (unset = inherit the server's `ANTHROPIC_API_KEY`) |
| model | a model is selected in the settings table (unset = the CLI's own default) |
| database | db opens in WAL with `notes` + `notes_fts` tables |
| index | indexed count matches the number of valid notes on disk |
| api | something speaks the Thoth protocol at the configured port (`GET /api/health` returns `ok`) |
| websocket | the chat WS upgrade succeeds (skipped when the api is unreachable) |

Flags:

| Flag | Purpose |
|---|---|
| `--fix` | Repair the safe failures: missing config (writes defaults), missing wiki (scaffolds), out-of-sync index (syncs). Never touches your Claude login. |
| `--dir` | (hidden, test-only) Override `~/.thoth` |

**Exit codes:** `0` when all checks pass, `1` when any fails. Script-friendly:

```sh
thoth doctor --fix && echo healthy
```

## Examples

```sh
# Full first-run
thoth init
thoth doctor          # confirm everything
thoth serve           # http://127.0.0.1:8333
```
