# CLI

All commands live in `internal/cli` (Cobra). The binary entrypoint is `cmd/thoth/main.go`.

## Commands

### `thoth serve`

Starts the app on `host:port` from the config (default `127.0.0.1:8333`). No flags.

Startup sequence:

1. Load `~/.thoth/config.toml` (persist defaults on first run)
2. Scaffold the wiki if it doesn't exist
3. Open `thoth.db` (index + store), rebuild the search index from the tree
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

Runs six health checks and reports each:

| Check | What it verifies |
|---|---|
| config | `config.toml` parses |
| wiki | wiki exists with all 8 folders + `CLAUDE.md` |
| claude | binary found; `claude --version` works; login status confirmed |
| database | db opens in WAL with `notes` + `notes_fts` tables |
| index | indexed count matches the number of valid notes on disk |
| port | the configured port is free |

Flags:

| Flag | Purpose |
|---|---|
| `--fix` | Repair the safe failures: missing config (writes defaults), missing wiki (scaffolds), out-of-sync index (rebuilds). Never touches your Claude login. |
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
