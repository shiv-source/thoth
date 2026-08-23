package wiki

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Import limits. The wiki is plain files, so an import is bounded by what it
// extracts to disk, not by memory — the caps stop a huge or hostile archive
// from exhausting disk. Documented in docs/development.md § API.
const (
	// MaxImportZipBytes caps the multipart request body (see the api
	// handler's http.MaxBytesReader).
	MaxImportZipBytes = 200 << 20 // 200 MiB
	// maxEntryBytes caps a single extracted entry, both as declared in the
	// zip header and as actually produced while copying.
	maxEntryBytes = 100 << 20 // 100 MiB
	// maxTotalUncompressed caps the whole archive's extracted size.
	maxTotalUncompressed = 500 << 20 // 500 MiB
)

// Sentinel errors the api handler maps onto status codes.
var (
	// ErrArchiveTooLarge means the archive exceeds MaxImportZipBytes.
	ErrArchiveTooLarge = errors.New("archive is too large")
	// ErrNotWiki means the archive is not wiki-shaped (no rulebook, no note
	// with frontmatter).
	ErrNotWiki = errors.New("archive is not a wiki: missing CLAUDE.md and no markdown note with frontmatter")
)

// ImportResult reports what ImportFrom did.
type ImportResult struct {
	// Files is the number of files merged into the wiki root.
	Files int `json:"files"`
	// Backup is the path of the pre-import backup directory, or "" when the
	// wiki root did not exist (nothing to back up).
	Backup string `json:"backup"`
}

// ImportFrom restores a wiki zip onto the wiki root as a backup-first merge.
// Every entry is validated (no absolute or `..` paths, no symlink escapes,
// per-entry and total size caps) and extracted into a staging directory; only
// then is the real root touched. The existing tree is copied to a sibling
// `<root>-backup-<timestamp>` first, then each staged file is renamed into
// place — the archive wins on conflicts, local-only files are kept. A wiki
// root that does not exist is created. Returns ErrNotWiki when the archive
// carries no rulebook and no markdown note with frontmatter, and
// ErrArchiveTooLarge when size exceeds MaxImportZipBytes; validation failures
// leave the real root untouched.
func (w *Wiki) ImportFrom(src io.ReaderAt, size int64, log *slog.Logger) (ImportResult, error) {
	if size > MaxImportZipBytes {
		return ImportResult{}, ErrArchiveTooLarge
	}
	zr, err := zip.NewReader(src, size)
	if err != nil {
		return ImportResult{}, fmt.Errorf("not a valid zip archive: %w", err)
	}
	root := w.Root()
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return ImportResult{}, fmt.Errorf("prepare wiki parent: %w", err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(root), ".thoth-import-")
	if err != nil {
		return ImportResult{}, fmt.Errorf("create staging dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	im := importer{log: log, maxEntry: maxEntryBytes, maxTotal: maxTotalUncompressed}
	for _, f := range zr.File {
		if err := im.extract(f, staging); err != nil {
			return ImportResult{}, err
		}
	}
	if !looksLikeWiki(staging) {
		return ImportResult{}, ErrNotWiki
	}

	backup, err := backupWiki(root)
	if err != nil {
		return ImportResult{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ImportResult{}, fmt.Errorf("create wiki root: %w", err)
	}
	files, err := mergeTree(staging, root)
	if err != nil {
		return ImportResult{}, fmt.Errorf("merge wiki: %w", err)
	}
	return ImportResult{Files: files, Backup: backup}, nil
}

// importer carries the per-import state. maxEntry and maxTotal mirror the
// package caps but are injectable so tests can exercise them with small
// archives.
type importer struct {
	log      *slog.Logger
	total    int64
	maxEntry int64
	maxTotal int64
}

// extract validates one archive entry and writes it under dest. The header's
// declared size is checked first (cheap), then the copy is capped so a lying
// header cannot blow past maxEntryBytes.
func (im *importer) extract(f *zip.File, dest string) error {
	if err := validEntryName(f.Name); err != nil {
		return err
	}
	if int64(f.UncompressedSize64) > im.maxEntry {
		return fmt.Errorf("archive entry %s is too large (%d bytes, max %d)", f.Name, f.UncompressedSize64, im.maxEntry)
	}
	if im.total+int64(f.UncompressedSize64) > im.maxTotal {
		return fmt.Errorf("archive exceeds %d bytes uncompressed", im.maxTotal)
	}
	target := filepath.Join(dest, filepath.FromSlash(f.Name))
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open archive entry %s: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	n, err := io.Copy(out, io.LimitReader(rc, im.maxEntry+1))
	if err != nil {
		return fmt.Errorf("extract archive entry %s: %w", f.Name, err)
	}
	if n > im.maxEntry {
		return fmt.Errorf("archive entry %s is too large (%d bytes, max %d)", f.Name, n, im.maxEntry)
	}
	im.total += n
	return nil
}

// validEntryName rejects zip entry names that would escape the extraction
// target: absolute paths, `..` segments, and anything that does not clean to
// itself ("./x", "a/../b", backslash tricks). Names are compared after
// platform-normalizing, matching how the file would actually be created.
func validEntryName(name string) error {
	if name == "" {
		return errors.New("empty archive entry name")
	}
	// Directory entries are stored with a trailing slash; strip it before the
	// path checks so "attachments/" validates like "attachments".
	norm := filepath.FromSlash(strings.TrimSuffix(name, "/"))
	clean := filepath.Clean(norm)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("archive entry %q escapes the wiki root", name)
	}
	if clean != norm {
		return fmt.Errorf("archive entry %q is not a clean relative path", name)
	}
	return nil
}

// looksLikeWiki reports whether the extracted tree is wiki-shaped: a
// root-level CLAUDE.md (the rulebook) or at least one markdown note that
// parses with frontmatter.
func looksLikeWiki(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		return true
	}
	ok := false
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || ok || d.IsDir() || !IsMarkdownPath(p) {
			return nil
		}
		if b, rerr := os.ReadFile(p); rerr == nil {
			if _, _, perr := ParseNote(b); perr == nil {
				ok = true
			}
		}
		return nil
	})
	return ok
}

// backupWiki copies the existing wiki root to a sibling <root>-backup-<ts>
// directory so an import can be recovered from. A missing root needs no
// backup ("" returned).
func backupWiki(root string) (string, error) {
	if _, err := os.Stat(root); err != nil {
		return "", nil
	}
	backup := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-backup-"+time.Now().Format("20060102-150405"))
	if err := copyTree(root, backup); err != nil {
		return "", fmt.Errorf("backup wiki: %w", err)
	}
	return backup, nil
}

// copyTree recursively copies src to dst (created if missing), preserving
// directory structure and file contents.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(p, target)
	})
}

// copyFile copies one regular file's contents.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// mergeTree renames every file from src (a staging dir on the same
// filesystem) onto dst, creating parent directories as needed and overwriting
// existing files. Returns the number of files merged.
func mergeTree(src, dst string) (int, error) {
	files := 0
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.Rename(p, target); err != nil {
			return err
		}
		files++
		return nil
	})
	return files, err
}
