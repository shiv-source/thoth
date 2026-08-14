package index

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/shiv-source/thoth/internal/wiki"
)

// Rebuild walks root and reindexes every .md file. Malformed notes are
// skipped and logged — the index must never block on one bad file.
func (ix *Index) Rebuild(root string, log *slog.Logger) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".md" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			log.Warn("index: cannot read note", "path", p, "err", err)
			return nil
		}
		meta, _, err := wiki.ParseNote(b)
		if err != nil {
			log.Warn("index: skipping malformed note", "path", p, "err", err)
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return fmt.Errorf("index: rel path %s: %w", p, err)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return ix.Upsert(Note{
			Path:      filepath.ToSlash(rel),
			Title:     meta.Title,
			Kind:      meta.Kind,
			Tags:      meta.Tags,
			Body:      string(b),
			UpdatedAt: info.ModTime(),
		})
	})
}
