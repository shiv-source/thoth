# Knowledge base

The wiki is a directory of plain markdown (default `~/.thoth/wiki`, changeable in Settings). Its organization is defined in one place — the rulebook (`CLAUDE.md` in the wiki root), which Claude reads at the start of every session, whether driven by the app or used in a terminal.

## Layout

| Folder | Purpose |
|---|---|
| `inbox/` | Quick captures awaiting filing; cleanup moves them to their homes |
| `meetings/` | One file per meeting: `2026-08-14-standup.md` |
| `projects/<name>/` | One folder per project; `project.md` holds overview + status |
| `links/` | `bookmarks.md` master list (category → link + one-word reason); separate note files only when a link deserves one |
| `setup/` | One file per machine; `setup/servers/<name>.md` per server |
| `knowledge/` | One topic per file: software and tooling knowledge |
| `todos/` | `TODO.md` — the single task list (Now / Next / Someday) |
| `daily/` | `2026-08-14.md` quick-capture journal |
| `attachments/` | Reserved, app-managed: non-markdown assets (images, scripts, configs) — indexed by filename, hidden from the tree |

New top-level domains are added by convention: create the folder and extend the rulebook if it needs rules. The scaffold folder set is configurable — set `wiki_folders` in Settings (comma-separated) and every freshly scaffolded wiki uses your set instead of the default 9. `thoth init` picks the same configured set up when `thoth.db` exists.

## Conventions

- **Filenames** — kebab-case; time-based notes get date prefixes (`meetings/2026-08-14-standup.md`)
- **Frontmatter** — every note starts with:

```yaml
---
title: <Title>
date: <YYYY-MM-DD>
tags: [<tag>, <tag>]
type: <meeting|project|link|setup|knowledge|todo|daily|note>
---
```

- **One TODO list** — `todos/TODO.md`; tasks are never scattered across other files
- **No secrets** — never store credentials; write placeholders like `<db-password>`

The rulebook ships as a template at `internal/wiki/templates/CLAUDE.md` and is scaffolded by `thoth init`; the template and the in-repo validation are the same source (`Rulebook()`, see [Components](components.md)). The rulebook's folder map is generated from the configured folder set, so it always matches the layout. An existing `CLAUDE.md` is never overwritten — edit it freely to adapt the organization to how you think.

Every scaffold also initializes a local git repository (when git is installed) with a `.gitignore` covering `.DS_Store` and `*.db`, so the wiki is versioned from day one; the Settings → Git remote tab adds the remote and pushes.

## Notes in the index

Notes are indexed only when their frontmatter parses and has a `title` (`internal/wiki/note.go`). Malformed files are skipped and logged, never fatal — see [Indexing & search](indexing.md).

Non-markdown files (images, scripts, configs) are stored in the reserved `attachments/` directory and indexed by **filename only**, so search can find them even though the folder is hidden from the tree. Their content is not searchable — when a script or config is saved, write a companion note in the folder that uses it (e.g. `setup/servers/x.md` for `attachments/x.yaml`) so its purpose is discoverable by search. The rulebook encodes this protocol.

`attachments/` is app-managed and checked on every startup: if a *file* (not a directory) exists with that name where the directory belongs, `serve` fails fast with an actionable error (`reserved directory "attachments" is blocked by a file; move or remove <path>`) rather than serving without the directory or surfacing a bare "not a directory".
