# Thoth 🦉

**Your personal knowledge base with a built-in assistant.**

Thoth keeps everything you know as plain markdown in one wiki directory and gives you a chat dashboard to ask questions and save new knowledge. The built-in assistant follows the wiki's rulebook, so everything lands in the right place — and the same directory works directly from Claude Code in the terminal, anytime.

Local-first. One binary. No cloud, no account, no data leaving your machine.

## Why Thoth

- **You own your knowledge** — plain markdown files, readable and diffable, in a directory you control
- **Ask, don't search** — chat with your wiki in natural language, grounded in your notes
- **Organized by default** — meeting notes, projects, links, setups, TODOs each have a home; the rulebook teaches the assistant how to file them
- **Two interfaces, one contract** — the dashboard and the terminal behave identically
- **Fast, private, portable** — local SQLite full-text search, localhost-only, one static binary

## Requirements

- [Claude Code](https://claude.com/claude-code) installed and logged in (on your `PATH`)
- Go 1.26+ and Node 24 + pnpm — only if building from source

## Quick start

```sh
thoth serve       # starts on http://127.0.0.1:8333 (scaffolds the wiki for you)
thoth doctor      # verify everything is healthy
```

`thoth serve` scaffolds the default wiki at `~/.thoth/wiki` automatically on first run — `thoth init` is optional (use it to scaffold at a custom location).

Open the dashboard and ask *"what did we decide in Tuesday's standup?"* — or say *"save this: <anything>"* and watch it get filed, searchable within seconds.

## Install

```sh
git clone https://github.com/<you>/thoth
cd thoth
make install-bin PREFIX=/usr/local/bin   # build + embed + copy to PREFIX
# or build in place:
make build                    # → bin/thoth
```

Release binaries: `make release VERSION=vX.Y.Z` cross-compiles all five targets into `dist/` (darwin/linux × amd64/arm64, windows/amd64) — a real `VERSION` is required, the target refuses to ship `dev`-stamped binaries.

## Documentation

Full documentation — a Docusaurus site built from the docs in this repo:

- **Getting started** — install, first run, your first conversation
- **Using Thoth** — dashboard tour, chat, search, settings, GitHub sync, best practices
- **Architecture, API, CLI, Schema, Indexing, Security, Development** — reference pages with diagrams

Run it locally:

```sh
make docs-dev      # Docusaurus dev server with hot reload
make docs-build    # production build
```

Or read the markdown directly: **[docs/index.md](docs/index.md)**.

## Development

```sh
make help     # self-documenting target list
make dev      # Vite HMR + Go server together
make check    # everything CI enforces: fmt, lint, race, coverage (≥80%), build
```

- Backend: Go 1.26 · Echo · Cobra · SQLite (FTS5) · fsnotify — `internal/`
- Frontend: React 19 · TypeScript (strict) · Vite · Tailwind CSS v4 — `web/`
- Docs: Docusaurus 3 — `docs-site/` (renders `docs/`)
- Repo rules, conventions, and invariants: [`CLAUDE.md`](CLAUDE.md)

## Security

Local-only by design: the server binds `127.0.0.1`, the WebSocket accepts only localhost origins, all filesystem access is path-traversal-safe, and search snippets are XSS-escaped. Secrets never belong in the wiki — the rulebook enforces placeholders. Details in the [Security](docs/security.md) guide.

## License

[MIT](LICENSE)
