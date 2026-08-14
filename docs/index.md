# Thoth Documentation

Thoth is a local-first personal knowledge base. You keep everything you know as plain markdown in one wiki directory; a Go server drives the Claude Code CLI to answer questions from that wiki and save new knowledge into it, through a polished web dashboard.

One binary. One directory (`~/.thoth`). No cloud, no auth, localhost only.

## Getting started

1. Install [Claude Code](https://claude.com/claude-code) and log in
2. `thoth init` — scaffold the default wiki at `~/.thoth/wiki`
3. `thoth serve` — open http://127.0.0.1:8333
4. `thoth doctor` — verify your setup anytime

## Documentation map

| Page | What it covers |
|---|---|
| [Architecture](architecture.md) | System design, the two layers, data contract, diagrams |
| [Knowledge base](knowledge-base.md) | The wiki directory: layout, conventions, the rulebook |
| [Components](components.md) | Deep dive into every Go package |
| [CLI](cli.md) | `serve`, `init`, `version`, `doctor` — flags and behavior |
| [API](api.md) | REST endpoints and the WebSocket chat protocol |
| [Indexing & search](indexing.md) | SQLite schema, FTS5, the file watcher |
| [Frontend](frontend.md) | React structure, design system, hooks, state flow |
| [Security](security.md) | Threat model and the mechanisms that implement it |
| [Development](development.md) | Toolchain, commands, testing gates, CI, contribution rules |

## Where things live

| Path | Role |
|---|---|
| `~/.thoth/config.toml` | Settings (wiki path, host/port, claude binary, model) |
| `~/.thoth/thoth.db` | SQLite: search index + conversation history (derived data) |
| `~/.thoth/wiki/` | The knowledge base — plain markdown you own |
| `CLAUDE.md` (repo root) | Rules for working on this codebase |
| `CLAUDE.md` (wiki root) | Rules Claude follows when reading/writing your notes |
