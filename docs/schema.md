# Database schema

Everything lives in one SQLite file, `~/.thoth/thoth.db` (WAL mode). The schema is defined entirely by SQL migrations in `internal/store/migrations/` — one file per table, applied in filename order, gated on `PRAGMA user_version` (currently 9). Go code issues no DDL of its own.

```mermaid
erDiagram
    conversations ||--o{ messages : "has"
    conversations {
        text id PK
        text title
        text created_at
        text claude_session_id
    }
    messages {
        integer id PK
        text conversation_id FK
        text role
        text content
        text created_at
    }
    notes ||--|| notes_fts : "FTS5 external content"
    notes {
        text path PK
        text title
        text kind
        text tags
        text body
        text updated_at
    }
    settings {
        text key PK
        text value
    }
    app_metadata {
        integer id PK "always 1"
        text installation_id
        text created_at
    }
    github_auth {
        text token
        text username
        text email
        text created_at
    }
    llm_models {
        integer id PK
        text value "UNIQUE"
        text name
        text tag
        text provider
    }
```

## Tables

### `conversations` (migration `0001_conversations.sql`)

One row per chat shown in the UI history.

| Column | Type | Meaning |
|---|---|---|
| `id` | TEXT PK | Conversation id (v4 UUID) — also the `/chat/<id>` URL id |
| `title` | TEXT NOT NULL | First user message, truncated (display only) |
| `created_at` | TEXT NOT NULL | UTC RFC3339 — the list orders by it lexically DESC, with `rowid DESC` breaking same-second ties so "most recent" is deterministic |
| `claude_session_id` | TEXT NOT NULL DEFAULT '' | Legacy Claude CLI session id. Thoth Agent no longer spawns the CLI and nothing reads or writes the column; it is **retained for schema stability** — the decision was to leave it rather than migrate it out (see below) |

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
| `model` | — (absent) | The model value selected for every turn; absent/empty falls back to the first seeded claude model. Read at boot, applied on next start |
| `api_key` | `''` | The shared API key — the fallback credential for any provider without its own key (see the `provider_*` keys). Empty (`''`) = not configured, inherit the server's environment. Never returned by the API — GET reports `has_api_key` only |
| `provider_<slug>_api_key` | — (absent) | A provider's own API key; `slug` is the lowercased provider label with non-alphanumerics stripped (`DeepSeek` → `deepseek`, `Z.AI` → `zai`). Wins over the shared `api_key` for that provider's models. Write-only like the shared key — GET reports `has_api_key` only |
| `provider_<slug>_base_url` | — (absent) | A provider's API base URL override (`DeepSeek` → `provider_deepseek_base_url`). Empty/absent = the provider's default endpoint. Applied on next boot |
| `github_sync_repo` | `''` | The wiki's sync repo URL |
| `github_sync_enabled` | `'false'` | The auto-sync switch (`'true'`/`'false'`) |
| `github_last_synced_at` | `''` | UTC RFC3339 of the last successful git sync |
| `github_sync_error` | `''` | Sanitized error of the last failed sync |

### `llm_models` (migrations `0008_llm_models.sql` + `0009_llm_models_tag.sql`)

The user-editable model registry. Every startup seeds it from `internal/assets/models.json` (the single source for the built-in list) when the table is empty; afterwards the table is authoritative — rows are added/edited/deleted from the Settings → LLM Models tab.

| Column | Meaning |
|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` |
| `value` | The model id sent to the provider on every turn; `UNIQUE` — the `model` setting points at it |
| `name` | Display name (e.g. `Claude Opus 4.8`) |
| `tag` | Preset-friendly label rendered as a colored chip (e.g. `strongest`) |
| `provider` | Grouping label for the picker (e.g. `Anthropic`) |

## Reading and writing

- `internal/settings` owns the `settings` table (KV access, `SyncEnabled`/`SyncState`/`SetSyncResult` conveniences, and `ProviderConfig` for the model→provider→credential resolution). Its `OpenRepo` deliberately runs no migrations and no WAL pragma — the doctor must never mutate a database it only reads.
- `internal/github` owns `github_auth`; `internal/store` owns conversations/messages/app_metadata and `llm_models`; `internal/index` owns notes/notes_fts.
- **`claude_session_id` decision (T12):** the column is retained, no migration. Thoth Agent stopped writing it when it replaced the CLI; dropping it would rewrite `conversations` for a column nothing reads, and keeping it preserves the schema for any rollback tooling. The settings resolution at boot is: the selected model's `llm_models` row names the provider → `provider_<slug>_api_key`/`provider_<slug>_base_url` when set, else the shared `api_key` and the provider's default endpoint (`settings.ProviderConfig`).

## Upgrade note

The schema baseline changed with the per-table restructure (config.toml deprecation) — upgrading an existing install requires deleting `thoth.db` (it is derived data; the wiki files are the source of truth).
