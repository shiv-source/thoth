package index

import (
	"database/sql"
	"fmt"
	"html"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Note struct {
	Path      string
	Title     string
	Kind      string
	Body      string
	Tags      []string
	UpdatedAt time.Time
}

type Result struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Snippet string `json:"snippet"`
}

type Index struct {
	db *sql.DB
}

// Open opens (creating if needed) the index database and applies the schema.
func Open(path string) (*Index, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Index{db: db}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS notes (
			path TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'note',
			tags TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
			path UNINDEXED, title, body, content='notes', content_rowid='rowid'
		);`,
		`CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
			INSERT INTO notes_fts(rowid, path, title, body) VALUES (new.rowid, new.path, new.title, new.body);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
			INSERT INTO notes_fts(notes_fts, rowid, path, title, body) VALUES ('delete', old.rowid, old.path, old.title, old.body);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
			INSERT INTO notes_fts(notes_fts, rowid, path, title, body) VALUES ('delete', old.rowid, old.path, old.title, old.body);
			INSERT INTO notes_fts(rowid, path, title, body) VALUES (new.rowid, new.path, new.title, new.body);
		END;`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate index: %w", err)
		}
	}
	return nil
}

func (ix *Index) Upsert(n Note) error {
	_, err := ix.db.Exec(`
		INSERT INTO notes(path, title, kind, tags, body, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title = excluded.title,
			kind = excluded.kind,
			tags = excluded.tags,
			body = excluded.body,
			updated_at = excluded.updated_at`,
		n.Path, n.Title, n.Kind, strings.Join(n.Tags, ","), n.Body, n.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("index upsert %s: %w", n.Path, err)
	}
	return nil
}

func (ix *Index) Delete(path string) error {
	if _, err := ix.db.Exec(`DELETE FROM notes WHERE path = ?`, path); err != nil {
		return fmt.Errorf("index delete %s: %w", path, err)
	}
	return nil
}

// DeletePrefix removes the note at prefix and every note stored under it.
// Removing a directory delivers no per-file events, so the watcher uses this
// to clear a whole subtree at once. LIKE wildcards in the prefix (a directory
// named "50%" or "a_b") are escaped so only literal matches are removed.
func (ix *Index) DeletePrefix(prefix string) error {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(prefix)
	if _, err := ix.db.Exec(`DELETE FROM notes WHERE path = ? OR path LIKE ? ESCAPE '\'`, prefix, escaped+"/%"); err != nil {
		return fmt.Errorf("index delete prefix %s: %w", prefix, err)
	}
	return nil
}

func (ix *Index) Search(q string, limit int) ([]Result, error) {
	// char(1)/char(2) are the match markers: FTS5's snippet returns them
	// verbatim, so escaping can happen in Go instead of trusting the raw
	// note text to flow through into HTML.
	rows, err := ix.db.Query(`
		SELECT n.path, n.title, n.kind,
		       snippet(notes_fts, 2, char(1), char(2), '…', 12)
		FROM notes_fts
		JOIN notes n ON n.rowid = notes_fts.rowid
		WHERE notes_fts MATCH ?
		ORDER BY bm25(notes_fts, 0.0, 8.0, 1.0)
		LIMIT ?`, q, limit)
	if err != nil {
		return nil, fmt.Errorf("search %q: %w", q, err)
	}
	defer rows.Close()
	var out []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.Path, &r.Title, &r.Kind, &r.Snippet); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		// Escape the note text first, then turn the markers into real tags:
		// any <mark>/</mark> the note itself contains comes back escaped.
		r.Snippet = strings.ReplaceAll(html.EscapeString(r.Snippet), "\x01", "<mark>")
		r.Snippet = strings.ReplaceAll(r.Snippet, "\x02", "</mark>")
		out = append(out, r)
	}
	return out, rows.Err()
}

func (ix *Index) Close() error { return ix.db.Close() }
