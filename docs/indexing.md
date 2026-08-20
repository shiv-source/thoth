# Indexing & search

The search index is derived data: it can always be rebuilt by walking the wiki. It lives in `~/.thoth/thoth.db` alongside conversation history.

## Schema

```sql
CREATE TABLE notes (
  path TEXT PRIMARY KEY, title TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'note', tags TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL, updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE notes_fts USING fts5(
  path UNINDEXED, title, body, content='notes', content_rowid='rowid'
);
```

Three triggers keep FTS in sync with `notes` on insert, delete, and update — the index and the table can never drift, no matter what code writes.

The database runs in **WAL mode**; only the Go process writes it.

## Search

`Index.Search` (`internal/index/index.go`):

- Match: FTS5 over title + body; the API wraps user queries as literal phrases (quotes stripped), so `*`, `OR`, `NOT` etc. are searched literally
- Ranking: `bm25` with the title weighted 8× the body
- Snippets: from the body, with `\x01`/`\x02` control markers → HTML-escaped → safe `<mark>`/`</mark>` tags

## Keeping the index current

```mermaid
flowchart LR
    A[Any writer<br/>app-served Claude / terminal Claude / user] -->|edits .md files| W[wiki dir]
    W -->|fsnotify events| EV{watcher}
    EV -->|debounce 200ms| P[pending map]
    P -->|file exists| U[parse → Upsert]
    P -->|file gone| D[Delete / DeletePrefix]
    P -->|malformed| L[log + skip]
    S[startup / path change] -->|walk, one transaction| R[Sync: upsert changed → delete missing]
    R --> DB[("thoth.db")]
    U --> DB
    D --> DB
```

- **Watcher** (`internal/index/watcher.go`) — watches every non-hidden directory (dot-directories like `.git` are skipped, so git activity never churns the watcher); Write/Create/Remove/Rename coalesce into a 200 ms debounce flush. New directories are added to the watch with an immediate rescan, so files written before registration are still indexed; watched directories are tracked so a removal is still recognized as a directory and clears the whole subtree. Each flush also publishes one `wiki.Changed` batch (op + wiki-relative path) to the event bus via `WithPublisher`; only paths the tree displays are published — markdown notes and directories, see `wiki.Visible` — so attachment changes publish nothing, and watcher start publishes an empty batch.
- **Sync** (`internal/index/sync.go`) — reconciles the index with the tree at startup and on wiki-path changes, in a single transaction: files whose stored `updated_at` matches their mtime are kept as is, new or changed files are upserted, and indexed paths no longer on disk are deleted. Notes with unparsable frontmatter are skipped and logged, never fatal. Every non-hidden file is indexed: markdown notes (`.md`/`.markdown`, case-insensitive) parsed for their frontmatter, and other files (attachments like images and scripts) by **filename only** so search can find them — their content is never indexed. `updated_at` is stored at nanosecond precision (RFC3339Nano), so edits within the same second as the last index write are still picked up.

## Guarantees

| Scenario | Outcome |
|---|---|
| Note saved by Claude in the app | Indexed within ~200 ms |
| Attachment (image, script, config) saved | Indexed by filename within ~200 ms (hidden from the tree) |
| Note edited in a terminal (Claude Code, vim) | Indexed within ~200 ms |
| Note deleted | Removed from index (file or whole directory) |
| Any tree-visible wiki change | A `wiki_changed` batch is published to the event bus within ~200 ms (the API layer pushes it to the UI, which refetches the tree) |
| Wiki path changed in Settings | New path scaffolded if needed, index synced, watcher restarted |
| App restarted | Incremental sync at startup — unchanged files are skipped, so the index still always reflects the tree |
| `thoth.db` deleted | No data loss — synced from the tree on next serve |
