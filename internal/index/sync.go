package index

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/shiv-source/thoth/internal/wiki"
)

// Sync reconciles the index with the tree rooted at root in a single
// transaction: files whose stored updated_at matches their mtime are kept as
// is, new or changed files are upserted, and indexed paths that no longer
// exist on disk are deleted. Malformed notes are skipped and logged — the
// index must never block on one bad file.
//
// An edit within the same second as the last index write is invisible to the
// mtime check; the next edit to that file re-syncs it.
func (ix *Index) Sync(root string, log *slog.Logger) error {
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("index: begin sync: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// existing maps path → stored updated_at. Paths the walk keeps or
	// upserts are consumed; whatever remains afterwards was deleted (or
	// became unreadable) since the last sync and must leave the index too.
	existing := map[string]string{}
	rows, err := tx.Query(`SELECT path, updated_at FROM notes`)
	if err != nil {
		return fmt.Errorf("index: sync load: %w", err)
	}
	for rows.Next() {
		var path, updatedAt string
		if err := rows.Scan(&path, &updatedAt); err != nil {
			_ = rows.Close()
			return fmt.Errorf("index: sync scan: %w", err)
		}
		existing[path] = updatedAt
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("index: sync load: %w", err)
	}
	_ = rows.Close()

	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".md" {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return fmt.Errorf("index: rel path %s: %w", p, rerr)
		}
		rel = filepath.ToSlash(rel)
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		stored, ok := existing[rel]
		if ok && stored == info.ModTime().Format(time.RFC3339) {
			delete(existing, rel)
			return nil // unchanged since the last index write
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			log.Warn("index: cannot read note", "path", p, "err", rerr)
			return nil // stays in existing → deleted below
		}
		meta, _, perr := wiki.ParseNote(b)
		if perr != nil {
			log.Warn("index: skipping malformed note", "path", p, "err", perr)
			return nil // stays in existing → deleted below
		}
		delete(existing, rel)
		return upsert(tx, Note{
			Path: rel, Title: meta.Title, Kind: meta.Kind,
			Tags: meta.Tags, Body: string(b), UpdatedAt: info.ModTime(),
		})
	})
	if err != nil {
		return fmt.Errorf("index: sync walk: %w", err)
	}

	for path := range existing {
		if err := del(tx, path); err != nil {
			return fmt.Errorf("index: sync delete %s: %w", path, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: sync commit: %w", err)
	}
	return nil
}
