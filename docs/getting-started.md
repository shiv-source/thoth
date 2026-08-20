# Getting started

Thoth is a local-first personal knowledge base. Everything you know lives as plain markdown in one wiki directory; a built-in assistant answers questions from that wiki and saves new knowledge into it, through a polished web dashboard.

One binary. One directory (`~/.thoth`). No cloud, no account, no data leaving your machine.

## What you need

- [Claude Code](https://claude.com/claude-code) installed and logged in (on your `PATH`) — the assistant that reads and writes your notes
- The Thoth binary — install from source (below), or download a release

## Install Thoth

```sh
git clone https://github.com/shiv-source/thoth
cd thoth
make install-bin PREFIX=/usr/local/bin   # build + embed + copy to PREFIX
# or build in place:
make build                    # → bin/thoth
```

If you'd rather not install it system-wide, `make build` produces `bin/thoth` in the repo.

## First run

```sh
thoth serve       # starts on http://127.0.0.1:8333 (scaffolds the wiki for you)
thoth doctor      # verify everything is healthy
```

1. **`thoth serve`** — starts the dashboard at `http://127.0.0.1:8333`. On first run it **scaffolds the wiki automatically**: it creates `~/.thoth/wiki` with the standard folder layout (`inbox/`, `meetings/`, `projects/`, `links/`, `setup/`, `knowledge/`, `todos/`, `daily/`, `attachments/`) plus the `CLAUDE.md` rulebook that tells the assistant how to file and find things, and initializes a local git repository so the wiki is versioned from day one. Open the dashboard in your browser.
2. **`thoth doctor`** — runs ten health checks (wiki, Claude binary + login, API key, model, database, index, malformed notes, API, WebSocket) and reports each. `thoth doctor --fix` repairs the safe failures. Exit code `0` means healthy.

> **`thoth init` is optional.** `serve` scaffolds the default wiki at `~/.thoth/wiki` for you. Run `thoth init` only when you want the wiki at a custom location: `thoth init ~/notes`.

## Your first conversation

In the dashboard, ask:

- *"what did we decide in Tuesday's standup?"* — the assistant searches your wiki and answers from it
- *"save this: the release is planned for Friday, owner is Priya"* — the assistant files a note into the right folder, searchable within seconds

Because notes are plain markdown, you can also open the wiki directly in any editor — or point Claude Code at `~/.thoth/wiki` in a terminal. Both interfaces follow the same rulebook, so notes land the same way.

## Next steps

- [Using Thoth](using-thoth.md) — the dashboard tour, chat, search, settings, and best practices
- [Knowledge base](knowledge-base.md) — the wiki layout and conventions
- [Architecture](architecture.md) — how it works under the hood
- [Troubleshooting](troubleshooting.md) — common issues and fixes
