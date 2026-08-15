package index

import (
	"database/sql"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/shiv-source/thoth/internal/store"
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

// dbLike is the Exec/Query surface both *sql.DB and *sql.Tx provide, so the
// single-row operations run on a plain handle or inside Sync's transaction.
type dbLike interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

// Open opens the index database. The schema (notes, notes_fts, triggers)
// lives in the store's SQL migrations, so a store open bootstraps it first;
// Open itself issues no DDL.
func Open(path string) (*Index, error) {
	st, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	if err := st.Close(); err != nil {
		return nil, err
	}
	db, err := store.OpenDB(path)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	return &Index{db: db}, nil
}

func (ix *Index) Upsert(n Note) error {
	if err := upsert(ix.db, n); err != nil {
		return fmt.Errorf("index upsert %s: %w", n.Path, err)
	}
	return nil
}

func upsert(db dbLike, n Note) error {
	_, err := db.Exec(`
		INSERT INTO notes(path, title, kind, tags, body, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title = excluded.title,
			kind = excluded.kind,
			tags = excluded.tags,
			body = excluded.body,
			updated_at = excluded.updated_at`,
		n.Path, n.Title, n.Kind, strings.Join(n.Tags, ","), n.Body, n.UpdatedAt.Format(time.RFC3339))
	return err
}

func (ix *Index) Delete(path string) error {
	if err := del(ix.db, path); err != nil {
		return fmt.Errorf("index delete %s: %w", path, err)
	}
	return nil
}

func del(db dbLike, path string) error {
	_, err := db.Exec(`DELETE FROM notes WHERE path = ?`, path)
	return err
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
	defer func() { _ = rows.Close() }()
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
