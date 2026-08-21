# CLI

All commands live in `internal/cli` (Cobra). The binary entrypoint is `cmd/thoth/main.go`.

## Commands

### `thoth serve`

Starts the app on `127.0.0.1:8333` (default). Flags:

- `--dev` — bind the dev port (`127.0.0.1:8334`, `config.DevPort`) and isolate all data under `~/.thoth/dev/` (its own `thoth.db`, default wiki `~/.thoth/dev/wiki/`), so a running instance keeps 8333 and its data; `make dev` uses this. Vite's proxy follows via the `THOTH_PORT` env var. At boot the dev server rewrites the seeded prod wiki default to the dev wiki and reports `dev: true` plus the checkout's full commit id in `/api/health`; the UI shows a warning banner with that commit.

Startup sequence:

1. Open `thoth.db` (migrations) and read `wiki_path` (and the optional `model`) from the settings table
2. Scaffold the wiki if it doesn't exist
3. Open `thoth.db` (index + store), sync the search index with the tree
4. Start the fsnotify watcher
5. Resolve the turn's model and credential: the selected model's `llm_models` row names the provider, whose per-provider api key/base URL win over the shared key and the provider's default endpoint (`modelProvider` + `ProviderConfig`)
6. Build the Thoth Agent host — `agent.New(model, apiKey, wiki, store, index, …)` with the provider config and folder set. It runs in-process: there is no CLI subprocess to spawn or pool to pre-warm anywhere in the chat path
7. Serve; SIGINT/SIGTERM → graceful shutdown (cancels in-flight agent turns, then exits)

### `thoth init [path]`

Scaffolds a wiki directory — the configured folder set (or the default 9, now including `attachments/`) plus the `CLAUDE.md` rulebook, then initializes a local git repo with a `.gitignore` (`.DS_Store`, `*.db`) when git is installed. Defaults to `~/.thoth/wiki`. Never overwrites an existing rulebook.

**Optional** — `serve` scaffolds the default wiki automatically when it doesn't exist (see the startup sequence above). Run `init` only to choose a custom location.

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

| wiki | wiki exists with all 9 folders + `CLAUDE.md` |
| provider | the provider the selected model's `llm_models` row names answers its models endpoint, probed with the per-provider API key and base URL (falling back to the shared key and the provider default); 200 = reachable; 401 = bad/absent API key, 429 = rate limited, 5xx = provider error, timeout = unreachable |
| api key | a usable API key is configured for the selected provider — its own key, or the shared `api_key` fallback (unset = the agent inherits the key from the server environment) |
| model | a model is selected and exists in the `llm_models` registry (unset = the default model; a value not in the registry = "unknown model") |
| database | db opens in WAL with `notes` + `notes_fts` tables |
| index | indexed count matches the number of valid notes on disk |
| malformed | no markdown notes the index silently skips (unparseable frontmatter) |
| api | something speaks the Thoth protocol at the configured port (`GET /api/health` returns `ok`) |
| websocket | the chat WS upgrade succeeds (skipped when the api is unreachable) |

Flags:

| Flag | Purpose |
|---|---|
| `--fix` | Repair the safe failures: missing wiki (scaffolds), out-of-sync index (syncs). Never touches provider connectivity or API keys. |
| `--dir` | (hidden, test-only) Override `~/.thoth` |
| `--provider-base-url` | (hidden, test-only) Provider base URL the provider check probes instead of the provider's public endpoint |

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
