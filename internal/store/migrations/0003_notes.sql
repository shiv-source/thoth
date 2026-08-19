-- notes — the search index's source rows. The wiki markdown files are the
-- source of truth; this table is derived data rebuilt from the tree, so it
-- can always be regenerated (doctor --fix → index.Sync).
--
-- path       wiki-relative file path (primary key — one row per note file)
-- title      from the frontmatter (required by the wiki rulebook)
-- kind       note | meeting | todo | … (frontmatter "type", default 'note');
--             "file" marks non-markdown attachments indexed by filename only
-- tags       comma-joined frontmatter tags
-- body       the note text below the frontmatter
-- updated_at UTC RFC3339Nano (sub-second precision), from the file's
-- modification time — nanosecond granularity keeps same-second edits visible
-- to the index sync's mtime check

CREATE TABLE IF NOT EXISTS notes (
    path TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'note',
    tags TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
