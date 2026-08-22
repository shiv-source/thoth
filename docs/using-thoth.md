# Using Thoth

Thoth gives you two ways to work with the same wiki: a **web dashboard** at `http://127.0.0.1:8333`, and the **wiki directory itself** (`~/.thoth/wiki`) which you can open in any editor — or drive directly with Claude Code in a terminal. Both follow the same rulebook, so notes land identically no matter who wrote them.

## The dashboard

The dashboard is organized into five views, reachable from the sidebar:

| View | Purpose |
|---|---|
| **Dashboard** | A landing page with KPIs, quick actions, and an overview of recent notes, chats, and tags |
| **Chat** | Natural-language conversation with your wiki — ask questions, save notes |
| **Notes** | Browse and read the wiki tree inline |
| **Search** | Full-text search over note titles and bodies |
| **Settings** | Wiki path, model, API key, LLM model registry, doctor checks, GitHub sync |

### Chat

The chat view is the heart of Thoth. Every conversation runs against the built-in **Thoth Agent**, which reads your wiki and writes into it under the rulebook's rules.

- **Ask** — type a question and get an answer grounded in your notes. The assistant searches the index and cites what it reads.
- **Save** — say *"save this: …"* and the assistant files the note into the right folder (see [Knowledge base](knowledge-base.md)) with correct frontmatter. Watch the wiki tree refresh within about 200 ms as the note is indexed.
- **Stop** — cancel an in-flight turn with the Stop button. The turn's request is aborted and nothing from that turn is saved.
- **New chat** — start a fresh conversation with the New chat button. Conversations are listed and day-grouped in the sidebar; delete them there too.

### Notes

The Notes view renders the wiki tree on the left and the open note on the right. Click any folder to expand it; click a note to read it. Markdown notes render with code highlighting; image attachments render inline; other attachments offer a download action.

The open note is reflected in the URL (`/notes/<path>`), so a note stays open across reloads and is shareable.

### Search

Full-text search over the wiki's notes — title matches rank 8× higher than body matches, and snippets show the matched text with `<mark>` highlights. Recent searches are kept locally (most-recent first, cleared with one click).

To get good search results, keep frontmatter titles accurate and write notes with descriptive titles — the title is what shows up and what ranks.

## Settings

Open Settings (gear icon in the header) to configure Thoth.

| Tab | What it does |
|---|---|
| **General** | The wiki path (change it here, with a folder browser), the provider and model used for new conversations, and the scaffold folders |
| **Providers** | One collapsible panel per provider: its credential form (an API key and base URL) plus the models registered under it (add/edit/delete). Keys are stored locally in thoth.db, never read from the environment, and never returned by the API |
| **Doctor** | The same health checks as `thoth doctor`, in the UI — run them any time, read each row's status |
| **Git remote** | Connect a GitHub account and push your wiki to a remote repo for sync and backup |

### GitHub sync

With the **Git remote** tab you can:

1. Connect a GitHub account with a token
2. Pick a repository from the connected account (or type any URL)
3. Turn on **auto-sync** to record the sync preference
4. **Initialize & Push** — the server initializes the repo if needed, points `origin` at the URL, commits the current tree, and pushes the branch

The wiki keeps its own local git history regardless — every scaffold initializes a repository (in-process, via the pure-Go `agent/git` backend — no git binary needed).

## The wiki itself

The real source of truth is the `~/.thoth/wiki` directory. You can:

- Open it in any editor and write/edit notes by hand
- Run `git` in it directly (every scaffold — whether from `thoth serve` or `thoth init` — initializes a repository)
- Point Claude Code at it in a terminal — `cd ~/.thoth/wiki && claude` — and use an assistant without the dashboard (Claude Code is a separate, optional tool; Thoth itself never spawns it)

Because files are the source of truth, `thoth.db` (the index + conversation history) is disposable — delete it and the index rebuilds from your files. Nothing is ever lost.

## Best practices

### Get the most out of the assistant

- **Be specific when saving** — *"save this: standup 2026-08-14 — decided to ship the report endpoint first, owner Priya"* gives the assistant everything it needs to file a well-formed meeting note.
- **Ask follow-ups in one conversation** — the assistant keeps the session context, so *"and who owns the docs?"* continues the same thread.
- **Reference what you know is filed** — if it's not in the wiki yet, the assistant will say so rather than invent it.

### Keep the wiki healthy

- **Frontmatter matters** — every note starts with `---` frontmatter including a `title`. Notes without a parseable title are skipped by the index (and shown by `thoth doctor`'s *malformed* check). See [Knowledge base](knowledge-base.md) for the exact shape.
- **Let the rulebook file things** — folders like `meetings/` and `todos/` have conventions (date prefixes, single TODO list). The assistant follows them; when you hand-write notes, follow them too so nothing drifts.
- **Attachments belong in `attachments/`** — non-markdown files (images, scripts, configs) are stored there and indexed by filename. When you save a script or config, add a companion note in the folder that uses it so its purpose is discoverable.
- **No secrets** — never store credentials in notes. Write placeholders like `<db-password>`. The rulebook enforces this.
- **Version everything** — the wiki is a git repo from day one; push it to a remote for backup, and you can always roll back.

## Troubleshooting

Something not working? See [Troubleshooting](troubleshooting.md) — it covers missing binaries, the doctor checks, index syncing, and more.
