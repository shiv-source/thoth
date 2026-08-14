# Thoth 🦉

Your personal knowledge base, powered by the Claude Code CLI.

Thoth keeps everything you know as plain markdown in one wiki directory and
gives you a chat interface to ask it questions and save new knowledge.
Claude follows the wiki's rulebook (`CLAUDE.md`) so everything lands in the
right place — and you can use Claude Code in the terminal on the same
directory any time you like.

## Requirements

- [Claude Code](https://claude.com/claude-code) installed, logged in, on your PATH
- Go 1.2x+ (to build from source) or a released binary

## Quick start

```sh
thoth init        # scaffold the default wiki at ~/.thoth/wiki
thoth serve       # starts on http://127.0.0.1:8333
```

Open the dashboard, ask "what did we decide in Tuesday's standup?", or say
"save this: <anything>". Everything the app creates lives under `~/.thoth/`
(config, SQLite index, default wiki). The wiki path is configurable in
Settings.

## Development

```sh
make web     # build the frontend (required before `go build`)
make test    # go test ./...
make race    # go test -race ./...
make lint    # golangci-lint + frontend lint + typecheck
```

- Backend: Go, Echo, SQLite (FTS5) — see `internal/`
- Frontend: React + TypeScript + Tailwind — see `web/`
- Design: `docs/superpowers/specs/2026-08-14-thoth-design.md`

## Security

Local-only, no authentication. Never store secrets in the wiki — the rulebook
enforces placeholders. By default the server binds to 127.0.0.1.

## License

MIT
