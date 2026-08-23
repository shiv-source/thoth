package wiki

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ExportOptions tunes ExportTo.
type ExportOptions struct {
	// IncludeHidden stores dotfiles and dot-directories (the wiki's .git,
	// .gitignore, .DS_Store, …) in the export so git history can travel.
	// Default: only .gitkeep is kept (it preserves empty scaffold folders);
	// every other dotfile is tooling, not wiki content.
	IncludeHidden bool
}

// exported reports whether a wiki-relative slash-separated path belongs in an
// export. Dotfiles are skipped by default except .gitkeep, which keeps empty
// scaffold folders alive through an export/import round-trip.
func exported(rel string, includeHidden bool) bool {
	if includeHidden {
		return true
	}
	if filepath.Base(rel) == ".gitkeep" {
		return true
	}
	return !Hidden(rel)
}

// ExportTo writes a zip of the wiki tree to dst, streamed: directories are
// stored as entries so empty folders survive, and each regular file is
// compressed as it is read, so memory stays bounded regardless of wiki size.
// The archive uses wiki-relative slash-separated entry names, so ImportFrom
// round-trips it onto another machine or root.
func (w *Wiki) ExportTo(dst io.Writer, opts ExportOptions) error {
	root := w.Root()
	zw := zip.NewWriter(dst)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if !exported(rel, opts.IncludeHidden) {
				return filepath.SkipDir
			}
			_, err := zw.Create(rel + "/")
			return err
		}
		if !exported(rel, opts.IncludeHidden) {
			return nil
		}
		return zipFile(zw, rel, p)
	})
	if cerr := zw.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	return nil
}

// zipFile stores the file at abs into the archive under the wiki-relative
// name rel, compressing its contents as they stream through.
func zipFile(zw *zip.Writer, rel, abs string) error {
	f, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = rel
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}
