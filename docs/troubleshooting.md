# Troubleshooting & FAQ

This page collects common problems and their fixes. If your issue isn't here, the doctor checks and the server log are the fastest way to diagnose.

## First steps for any problem

1. **`thoth doctor`** — runs the shared health checks and reports each one with a message. `thoth doctor --fix` repairs the safe failures (missing config, missing wiki, out-of-sync index). Exit code `0` = all healthy.
2. **`thoth serve --dev`** — serves on port 8334 with data isolated under `~/.thoth/dev/`, so you can debug against a clean slate without touching your real wiki or database.
3. **Server logs** — run the server in a terminal and read the structured logs. Errors are logged with their context (`path`, `err`); 500 responses are deliberately generic, but the detail is in the log.

## "thoth" is not recognized

The binary isn't on your `PATH`. Either install it:

```sh
make install-bin PREFIX=/usr/local/bin
```

…or run it in place from the repo: `./bin/thoth serve`.

## "claude" is not found

Thoth drives the Claude Code CLI, which must be installed, logged in, and on your `PATH`. From the terminal:

```sh
claude --version      # should print a version
claude login          # sign in if prompted
```

The **claude** doctor check verifies the binary is found, `claude --version` works, and login is confirmed. If the check passes, `thoth doctor` will tell you exactly which part is missing.

## The wiki isn't being indexed

The index is derived data — it always syncs with the tree at startup and on changes. If search is missing notes:

1. Run `thoth doctor` and look at the **index** and **malformed** checks.
2. A note is skipped when its frontmatter doesn't parse or has no `title`. The **malformed** check names those files.
3. Attachments are indexed **by filename only**, and live in `attachments/` (hidden from the tree) — their content is not searchable.
4. Deleting `~/.thoth/thoth.db` costs nothing: it rebuilds from the tree on the next `serve`.

## Search returns nothing for a phrase

`*`, `OR`, `NOT`, quotes and similar are searched **literally** — the query is wrapped as a phrase. If the exact wording isn't in a note's title or body, nothing matches. Try a shorter query, or check that the note has a `title` in its frontmatter (titles rank 8× higher).

## The chat says the assistant is unavailable or a turn fails

- Check the **claude** doctor check: binary present, version works, logged in.
- If you set an API key in Settings → General, it is passed to spawned assistant processes as `ANTHROPIC_API_KEY`. A wrong/expired key fails turns — check it, or clear it to inherit the server's environment.
- The conversation's assistant process is killed on cancel; the next send respawns it and resumes the session from disk. If a process was killed and the CLI reports the stored session as "already in use", the session is forked to a fresh id automatically.

## Port 8333 is already in use

`thoth serve` binds `127.0.0.1:8333`; `serve --dev` binds 8334. If 8333 is taken, either stop the other process or run the dev server on 8334. The bind address is fixed by design — it is not configurable.

## The wiki path in Settings isn't taking effect

Changing the wiki path in Settings:

1. Scaffolds the new wiki if it doesn't exist
2. Syncs the index
3. Restarts the watcher

If the new path is unwritable or a file blocks it (e.g. a file named `attachments` where the directory belongs), `serve` fails fast with an actionable error. Check the path is an absolute directory you can write to.

## The wiki tree shows a folder with an error

A directory that exists but can't be read stays in the tree with an error and no children — one locked folder doesn't fail the whole tree. Fix the permissions on that directory; the tree refreshes on the next change or focus.

## A note I edited by hand isn't showing up

The index watches the wiki with fsnotify and re-indexes within ~200 ms. If you edited with an external tool and nothing changed:

- Confirm the file is valid markdown with parseable frontmatter (`title` required).
- Check the **index** doctor check for a count mismatch.
- As a last resort, restart `serve` — startup performs a full incremental sync (unchanged files are skipped).

## GitHub sync is failing

The **Git remote** tab in Settings surfaces the error from the last failed sync (sanitized — credentials are never echoed). Common causes:

- The token was revoked or lacks `repo` scope — reconnect the account.
- The remote repo requires review/permissions the token doesn't have.
- The wiki's git repo has diverged from the remote — reconcile locally with `git` and push again.

## I want to reset everything

```sh
rm -rf ~/.thoth/thoth.db        # derived data — rebuilds from the tree
rm -rf ~/.thoth/wiki            # your notes — back these up first!
thoth init && thoth serve
```

Deleting only the database is always safe. Deleting the wiki deletes your knowledge — export it first (it's just files).

## Where is my data stored?

| Path | Contents |
|---|---|
| `~/.thoth/wiki/` | Your notes — plain markdown (the source of truth) |
| `~/.thoth/thoth.db` | SQLite: search index + conversation history (derived, rebuildable) |
| `~/.thoth/stream-dump.json` | Raw assistant stream output (debugging, rotated past 10 MB) |

## FAQ

**Does Thoth work offline?** Yes — everything is local (server, index, wiki). The assistant needs network access to the model provider (Claude), but all your data stays on your machine.

**Is my data encrypted?** The data is stored in plain files on your disk and the server binds localhost only. There's no cloud copy unless you enable GitHub sync. For security details see [Security](security.md).

**Can I use Thoth without the dashboard?** Yes. The wiki is plain markdown in `~/.thoth/wiki` — open it in any editor, or run Claude Code in that directory in a terminal. Both interfaces obey the same rulebook.

**Can I change where the wiki lives?** Yes — Settings → General → wiki path (with a folder browser). The index and watcher move with it.

**Can I use a different model?** Yes — Settings → General → model, or manage the full registry in Settings → LLM Models.

**Do I need to keep the server running?** Only while you use the dashboard. Notes are just files; nothing needs the server to be read.
