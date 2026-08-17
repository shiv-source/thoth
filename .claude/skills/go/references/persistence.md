# Persistence — thoth.db, migrations, index

Everything lives in one SQLite file, ~/.thoth/thoth.db (WAL mode). The
schema is defined entirely by SQL migrations in internal/store/migrations/,
applied in filename order and gated on PRAGMA user_version (currently 7).
Go code issues no DDL of its own.

## Migrations (internal/store/migrations/)
- 0001_conversations.sql — conversations: id (v4 UUID, also the /chat/<id> URL id), title, created_at (UTC RFC3339), claude_session_id (seeded as id; rotated via --resume/--fork-session on stale locks)
- 0002_messages.sql — messages: id, conversation_id, role (user|assistant), content, created_at; replayed in id order
- 0003_notes.sql — notes: path PK, title (frontmatter, required), kind (from frontmatter type), tags, body, updated_at — derived from the wiki tree
- 0004_notes_fts.sql — notes_fts: FTS5 external-content index over notes, kept in sync by triggers; path UNINDEXED
- 0005_app_metadata.sql — app_metadata: single row (CHECK id = 1) — installation_id, created_at; seeded by EnsureMetadata on boot
- 0006_github_auth.sql — github_auth: PAT (plaintext, gh-CLI trust model, never serialized by the API) + identity + scopes
- 0007_settings.sql — settings: key/value table; new keys need no schema change

## Settings keys (0007)
- wiki_path — seeded to ~/.thoth/wiki (mirrors settings.DefaultWikiPath)
- model — the --model value on every CLI spawn; absent/empty keeps the CLI default
- github_sync_repo / github_sync_enabled / github_last_synced_at / github_sync_error — the sync switch + state

## Ownership (who owns which table)
- internal/settings → settings (runs no migrations, no WAL pragma — the doctor must never mutate a database it only reads)
- internal/github → github_auth
- internal/store → conversations, messages, app_metadata
- internal/index → notes, notes_fts

## The index (internal/index)
- WAL + schema migration on Open; Upsert/Delete/DeletePrefix keep FTS5 in sync via triggers
- Search(q, limit): bm25 ranking (title 8×), HTML-escaped snippets with safe <mark> highlights
- Sync(root, log): reconciles the index with the tree in one transaction; malformed notes skipped
- Watch(ctx, root, ix, log): fsnotify with 200 ms debounce, new-directory rescan

## Rules that matter here
- thoth.db is derived data — files are the source of truth; deleting thoth.db is a supported upgrade path
- IDs are RFC 4122 v4 UUIDs (google/uuid) because the Claude CLI requires UUIDs for --session-id
- Timestamps are stored UTC so ordering is chronological

Canonical: docs/schema.md · docs/indexing.md

Stale if: migration count ≠ 7, a new settings key is missing above, or a
table gains a column without a docs/schema.md update.
