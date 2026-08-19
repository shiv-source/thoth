# Database schema

Everything lives in one SQLite file, `~/.thoth/thoth.db` (WAL mode). The schema is defined entirely by SQL migrations in `internal/store/migrations/` — one file per table, applied in filename order, gated on `PRAGMA user_version` (currently 7). Go code issues no DDL of its own.

## Tables

### `conversations` (migration `0001_conversations.sql`)

One row per chat shown in the UI history.

| Column | Type | Meaning |
|---|---|---|
| `id` | TEXT PK | Conversation id (v4 UUID) — also the `/chat/<id>` URL id |
| `title` | TEXT NOT NULL | First user message, truncated (display only) |
| `created_at` | TEXT NOT NULL | UTC RFC3339 — the list orders by it lexically DESC |
| `claude_session_id` | TEXT NOT NULL DEFAULT '' | The Claude CLI session backing the chat. Seeded with the conversation id; rotated to a fresh id via `--resume/--fork-session` when the CLI reports the stored session as locked |

### `messages` (migration `0002_messages.sql`)

The chat transcript, persisted for the UI history view.

| Column | Type | Meaning |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | Replayed in id order |
| `conversation_id` | TEXT NOT NULL | → `conversations.id` |
| `role` | TEXT NOT NULL | `user` or `assistant` |
| `content` | TEXT NOT NULL | Plain markdown text |
| `created_at` | TEXT NOT NULL | UTC RFC3339 |

### `notes` (migration `0003_notes.sql`)

The search index's source rows. The wiki markdown files are the source of truth — this table is derived data, rebuilt from the tree.

| Column | Type | Meaning |
|---|---|---|
| `path` | TEXT PK | Wiki-relative file path |
| `title` | TEXT NOT NULL | From frontmatter (required by the rulebook); the filename for attachments |
| `kind` | TEXT NOT NULL DEFAULT 'note' | `note`, `meeting`, `todo`, `file` (attachment), … |
| `tags` | TEXT NOT NULL DEFAULT '' | Comma-joined frontmatter tags |
| `body` | TEXT NOT NULL | Note text below the frontmatter; empty for attachments |
| `updated_at` | TEXT NOT NULL | UTC RFC3339Nano (sub-second), from file mtime |

### `notes_fts` (migration `0004_notes_fts.sql`)

FTS5 full-text index over notes, kept in sync by triggers. External-content index (`content='notes'`): no data duplication; searches always JOIN back to `notes`. `path` is UNINDEXED — matches run over title+body.

### `app_metadata` (migration `0005_app_metadata.sql`)

One-time install facts, single row (enforced by `CHECK (id = 1)`).

| Column | Type | Meaning |
|---|---|---|
| `id` | INTEGER PK DEFAULT 1 | Always 1 — the single-row constraint |
| `installation_id` | TEXT NOT NULL | v4 UUID identifying this installation |
| `created_at` | TEXT NOT NULL | First boot, UTC RFC3339 |

### `github_auth` (migration `0006_github_auth.sql`)

The connected GitHub account, single row. Identity only — the sync repo URL lives in `settings`.

| Column | Meaning |
|---|---|
| `token` | The PAT, plaintext (gh-CLI trust model). Never serialized by the API |
| `username` / `display_name` / `email` / `avatar_url` / `profile_url` | Identity from `GET /user` (+ `GET /user/emails` for the primary verified email) |
| `scopes` | `X-OAuth-Scopes` header — kept to warn about a missing `user:email` scope |
| `expires_at` | Reserved — `/user` returns no expiry |
| `account_created_at` / `account_updated_at` | The GitHub account's own timestamps |
| `created_at` / `updated_at` | When the connection was first made / last refreshed |

### `settings` (migration `0007_settings.sql`)

The app's user-facing settings, key/value. `config.toml` is deprecated — this table is the single source. New keys need no schema change.

| Key | Seed | Meaning |
|---|---|---|
| `wiki_path` | `~/.thoth/wiki` | Where the wiki lives (seed mirrors `settings.DefaultWikiPath`) |
| `wiki_folders` | — (absent) | Comma-separated scaffold folder set; absent/empty means the default 9 (`inbox, meetings, projects, links, setup, knowledge, todos, daily, attachments`). Applied when a wiki is scaffolded |
| `model` | — (absent) | The `--model` value enforced on every Claude CLI spawn; absent/empty keeps the CLI's default. Read at boot, applied on next start |
| `api_key` | `''` | The API key passed to spawned Claude CLI processes as `ANTHROPIC_API_KEY`; set from the web Settings (General tab). Empty (`''`) = not configured, inherit the server's environment. Never returned by the API — GET reports `has_api_key` only |
| `github_sync_repo` | `''` | The wiki's sync repo URL |
| `github_sync_enabled` | `'false'` | The auto-sync switch (`'true'`/`'false'`) |
| `github_last_synced_at` | `''` | UTC RFC3339 of the last successful git sync |
| `github_sync_error` | `''` | Sanitized error of the last failed sync |

### `llm_models` (migrations `0008_llm_models.sql` + `0009_llm_models_tag.sql`)

The user-editable model registry. Every startup seeds it from `internal/assets/models.json` (the single source for the built-in list) when the table is empty; afterwards the table is authoritative — rows are added/edited/deleted from the Settings → LLM Models tab.

| Column | Meaning |
|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` |
| `value` | The `--model` argument; `UNIQUE` — the `model` setting points at it |
| `name` | Display name (e.g. `Claude Opus 4.8`) |
| `tag` | Preset-friendly label rendered as a colored chip (e.g. `strongest`) |
| `provider` | Grouping label for the picker (e.g. `Anthropic`) |

## Reading and writing

- `internal/settings` owns the `settings` table (KV access, `SyncEnabled`/`SyncState`/`SetSyncResult` conveniences). Its `OpenRepo` deliberately runs no migrations and no WAL pragma — the doctor must never mutate a database it only reads.
- `internal/github` owns `github_auth`; `internal/store` owns conversations/messages/app_metadata and `llm_models`; `internal/index` owns notes/notes_fts.

## Upgrade note

The schema baseline changed with the per-table restructure (config.toml deprecation) — upgrading an existing install requires deleting `thoth.db` (it is derived data; the wiki files are the source of truth).
