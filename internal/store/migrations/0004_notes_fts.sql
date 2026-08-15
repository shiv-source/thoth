-- notes_fts — FTS5 full-text index over notes, plus the triggers that keep
-- it in sync.
--
-- content='notes', content_rowid='rowid' makes it an external-content index:
-- the text lives in notes, FTS stores only the inverted index, so no data is
-- duplicated and the triggers are the single write path (direct FTS writes
-- would desync it — searches always JOIN back to notes).
--
-- The triggers mirror inserts/updates/deletes into the FTS row (path is
-- UNINDEXED: search matches title+body, results read path/title/kind from
-- the notes row via the JOIN).

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    path UNINDEXED, title, body, content='notes', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, path, title, body) VALUES (new.rowid, new.path, new.title, new.body);
END;

CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, path, title, body) VALUES ('delete', old.rowid, old.path, old.title, old.body);
END;

CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, path, title, body) VALUES ('delete', old.rowid, old.path, old.title, old.body);
    INSERT INTO notes_fts(rowid, path, title, body) VALUES (new.rowid, new.path, new.title, new.body);
END;
