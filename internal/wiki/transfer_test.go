package wiki

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// discardLog silences the advisory diagnostics ImportFrom may emit.
var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

// sampleWiki scaffolds a wiki with a note, an attachment, an empty folder
// (kept alive by .gitkeep) and the root rulebook, and returns its root.
func sampleWiki(t *testing.T, root string) string {
	t.Helper()
	if err := Scaffold(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inbox", "hello.md"), []byte("---\ntitle: Hello\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "projects", "thoth"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "thoth", "project.md"), []byte("---\ntitle: Thoth\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The scaffold wrote attachments/.gitkeep; add a real attachment.
	if err := os.WriteFile(filepath.Join(root, "attachments", "logo.png"), []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// exportedZip builds an in-memory zip from a map of path → content.
func exportedZip(t *testing.T, w *Wiki, opts ExportOptions) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	if err := w.ExportTo(&buf, opts); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func zipNames(t *testing.T, r io.ReaderAt, size int64) []string {
	t.Helper()
	zr, err := zip.NewReader(r, size)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func TestExportImportRoundTrip(t *testing.T) {
	src := sampleWiki(t, filepath.Join(t.TempDir(), "wiki"))
	zipR := exportedZip(t, New(src), ExportOptions{})

	// The export carries the rulebook, the notes, the attachment, and the
	// .gitkeep that keeps empty folders alive.
	for _, want := range []string{"CLAUDE.md", "inbox/hello.md", "projects/thoth/project.md", "attachments/logo.png", "knowledge/.gitkeep", "inbox/.gitkeep"} {
		found := false
		for _, n := range zipNames(t, zipR, int64(zipR.Len())) {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("export missing %s; entries: %v", want, zipNames(t, zipR, int64(zipR.Len())))
		}
	}

	// Import into a fresh root: the tree must match the source byte for byte.
	dst := filepath.Join(t.TempDir(), "wiki")
	dest := New(dst)
	result, err := dest.ImportFrom(zipR, int64(zipR.Len()), discardLog)
	if err != nil {
		t.Fatal(err)
	}
	if result.Backup != "" {
		t.Fatalf("backup created for a fresh root: %q", result.Backup)
	}
	for _, rel := range []string{"inbox/hello.md", "projects/thoth/project.md", "attachments/logo.png", "CLAUDE.md"} {
		got, err := dest.Read(rel)
		if err != nil {
			t.Fatalf("imported %s missing: %v", rel, err)
		}
		want, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s content differs after round-trip", rel)
		}
	}
}

func TestExportExcludesDotfilesByDefault(t *testing.T) {
	root := sampleWiki(t, filepath.Join(t.TempDir(), "wiki"))
	if err := os.WriteFile(filepath.Join(root, ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden", "secret.md"), []byte("---\ntitle: Secret\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipR := exportedZip(t, New(root), ExportOptions{})
	names := zipNames(t, zipR, int64(zipR.Len()))
	for _, absent := range []string{".DS_Store", ".hidden/secret.md", ".gitignore", ".git/"} {
		for _, n := range names {
			if n == absent {
				t.Errorf("default export must not contain %s (got %v)", absent, names)
			}
		}
	}

	// history=1 includes the dotfiles (git history can travel).
	zipR = exportedZip(t, New(root), ExportOptions{IncludeHidden: true})
	names = zipNames(t, zipR, int64(zipR.Len()))
	joined := strings.Join(names, ",")
	for _, want := range []string{".DS_Store", ".hidden/secret.md"} {
		if !strings.Contains(joined, want) {
			t.Errorf("history export missing %s: %v", want, names)
		}
	}
}

func TestImportRejectsTraversal(t *testing.T) {
	for _, tt := range []struct {
		name  string
		entry string
	}{
		{"parent", "../evil.txt"},
		{"absolute", "/etc/evil.txt"},
		{"dotdot-inside", "a/../evil.txt"},
		{"dot-segment", "./evil.txt"},
		{"nested-dotdot", "a/b/../../evil.txt"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			if _, err := zw.Create(tt.entry); err != nil {
				t.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(t.TempDir(), "wiki")
			if _, err := New(root).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog); err == nil {
				t.Fatalf("traversal entry %q accepted", tt.entry)
			}
			if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("root must stay untouched after a rejected import: %v", err)
			}
		})
	}
}

func TestImportRejectsMalformedZip(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wiki")
	if _, err := New(root).ImportFrom(bytes.NewReader([]byte("not a zip")), 9, discardLog); err == nil {
		t.Fatal("malformed zip accepted")
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("root must stay untouched: %v", err)
	}
}

func TestImportRejectsNotWiki(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("plain text, no rulebook, no frontmatter")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "wiki")).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog); !errors.Is(err, ErrNotWiki) {
		t.Fatalf("want ErrNotWiki, got %v", err)
	}
}

func TestImportRejectsFrontmatterlessNote(t *testing.T) {
	// A markdown file without frontmatter does not make the archive
	// wiki-shaped (the looksLikeWiki ParseNote fallback must fail).
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("inbox/bad.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("# No frontmatter here")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "wiki")).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog); !errors.Is(err, ErrNotWiki) {
		t.Fatalf("want ErrNotWiki, got %v", err)
	}
}

func TestImportMergeConflictSurfaces(t *testing.T) {
	// The archive is wiki-shaped (has CLAUDE.md) but names a regular FILE
	// "inbox" while the destination root has inbox/ as a directory — the
	// rename onto the existing directory fails and the merge error surfaces.
	root := filepath.Join(t.TempDir(), "wiki")
	if err := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"CLAUDE.md", "inbox"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog); err == nil {
		t.Fatal("merge conflict with an existing directory should fail")
	}
}

func TestImportAcceptsFrontmatterWithoutRulebook(t *testing.T) {
	// A wiki-shaped archive may carry no CLAUDE.md as long as a markdown note
	// parses with frontmatter (the looksLikeWiki fallback).
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("inbox/hello.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("---\ntitle: Hello\n---\nBody")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "wiki")
	result, err := New(root).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog)
	if err != nil {
		t.Fatal(err)
	}
	if result.Files != 1 {
		t.Fatalf("merged %d files, want 1", result.Files)
	}
	if b, err := os.ReadFile(filepath.Join(root, "inbox", "hello.md")); err != nil || !bytes.Contains(b, []byte("title: Hello")) {
		t.Fatalf("imported note = %q, %v", b, err)
	}
}

func TestImportRejectsTooLarge(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wiki")
	if _, err := New(root).ImportFrom(bytes.NewReader([]byte("x")), MaxImportZipBytes+1, discardLog); !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("want ErrArchiveTooLarge, got %v", err)
	}
}

func TestImportRejectsOversizedEntry(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("inbox/big.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(bytes.Repeat([]byte("x"), 1024)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	im := importer{log: discardLog, maxEntry: 100, maxTotal: 1000}
	// The header-declared size trips the cap before anything is written.
	if err := im.extract(zr.File[0], t.TempDir()); err == nil {
		t.Fatal("oversized entry accepted")
	}
}

func TestImportBackupFirstMerge(t *testing.T) {
	// The existing wiki root already holds inbox/conflict.md (old content)
	// and inbox/local.md (only present locally).
	root := sampleWiki(t, filepath.Join(t.TempDir(), "wiki"))
	if err := os.WriteFile(filepath.Join(root, "inbox", "conflict.md"), []byte("---\ntitle: Conflict\n---\nOLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inbox", "local.md"), []byte("---\ntitle: Local\n---\nKeep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A second wiki to import, with a newer conflict.md and a brand-new note.
	src := filepath.Join(t.TempDir(), "other")
	if err := Scaffold(src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "inbox", "conflict.md"), []byte("---\ntitle: Conflict\n---\nNEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "inbox", "new.md"), []byte("---\ntitle: New\n---\nImported"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipR := exportedZip(t, New(src), ExportOptions{})

	dest := New(root)
	result, err := dest.ImportFrom(zipR, int64(zipR.Len()), discardLog)
	if err != nil {
		t.Fatal(err)
	}
	if result.Backup == "" {
		t.Fatal("expected a backup for an existing root")
	}
	// Archive wins on conflicts, local-only files are kept, new files land.
	for rel, want := range map[string]string{
		"inbox/conflict.md": "NEW",
		"inbox/local.md":    "Keep me",
		"inbox/new.md":      "Imported",
	} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s after merge: %v", rel, err)
		}
		if !strings.Contains(string(b), want) {
			t.Errorf("%s = %q, want it to contain %q", rel, b, want)
		}
	}
	// The backup preserves the pre-import tree.
	b, err := os.ReadFile(filepath.Join(result.Backup, "inbox", "conflict.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "OLD") {
		t.Errorf("backup conflict.md = %q, want the original OLD content", b)
	}
}

func TestImportRejectsEmptyEntryName(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create(""); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "wiki")).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog); err == nil {
		t.Fatal("empty entry name accepted")
	}
}

func TestImportRejectsTotalSizeCap(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i < 3; i++ {
		f, err := zw.Create(fmt.Sprintf("inbox/f%d.md", i))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(bytes.Repeat([]byte("x"), 100)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	im := importer{log: discardLog, maxEntry: 1000, maxTotal: 250}
	// Two 100-byte entries fit; the third trips the uncompressed total cap.
	for _, f := range zr.File {
		if err := im.extract(f, t.TempDir()); err != nil {
			if !strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
	}
	t.Fatal("expected the total uncompressed cap to trip")
}

func TestExportFailsOnUnreadableDir(t *testing.T) {
	root := sampleWiki(t, filepath.Join(t.TempDir(), "wiki"))
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "secret.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var buf bytes.Buffer
	if err := New(root).ExportTo(&buf, ExportOptions{}); err == nil {
		t.Fatal("expected export to fail on an unreadable directory")
	}
}

func TestImportFailsWhenBackupUnreadable(t *testing.T) {
	// The existing root holds an unreadable directory, so the pre-import
	// backup copy fails and the whole import aborts before the root is
	// touched.
	root := sampleWiki(t, filepath.Join(t.TempDir(), "wiki"))
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "secret.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	src := sampleWiki(t, filepath.Join(t.TempDir(), "src"))
	zipR := exportedZip(t, New(src), ExportOptions{})
	if _, err := New(root).ImportFrom(zipR, int64(zipR.Len()), discardLog); err == nil {
		t.Fatal("expected the backup copy to fail")
	}
}

func TestExportFailsOnUnreadableFile(t *testing.T) {
	// A regular file that cannot be opened fails the export at the zip stage.
	root := sampleWiki(t, filepath.Join(t.TempDir(), "wiki"))
	secret := filepath.Join(root, "inbox", "secret.md")
	if err := os.WriteFile(secret, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(secret, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secret, 0o644) })
	var buf bytes.Buffer
	if err := New(root).ExportTo(&buf, ExportOptions{}); err == nil {
		t.Fatal("expected export to fail on an unreadable file")
	}
}

func TestImportFailsWhenWikiParentIsFile(t *testing.T) {
	// A root whose parent path is a regular file makes the parent-creation
	// step fail before any extraction happens.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := sampleWiki(t, filepath.Join(t.TempDir(), "src"))
	zipR := exportedZip(t, New(src), ExportOptions{})
	if _, err := New(filepath.Join(blocker, "wiki")).ImportFrom(zipR, int64(zipR.Len()), discardLog); err == nil {
		t.Fatal("expected import to fail when the wiki parent is a file")
	}
}

func TestImportFailsWhenWikiRootIsFile(t *testing.T) {
	// A wiki root that is itself a regular file fails the root-creation step
	// (the backup succeeds, then mkdir on the file path fails).
	root := filepath.Join(t.TempDir(), "wiki")
	if err := os.WriteFile(root, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := sampleWiki(t, filepath.Join(t.TempDir(), "src"))
	zipR := exportedZip(t, New(src), ExportOptions{})
	if _, err := New(root).ImportFrom(zipR, int64(zipR.Len()), discardLog); err == nil {
		t.Fatal("expected import to fail when the wiki root is a file")
	}
}

func TestImportFailsWhenEntryCollidesWithFile(t *testing.T) {
	// A zip naming a plain file "inbox" and then "inbox/a.md": the second
	// entry's parent is a file, so extraction fails instead of escaping.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"inbox", "inbox/a.md"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "wiki")).ImportFrom(bytes.NewReader(buf.Bytes()), int64(buf.Len()), discardLog); err == nil {
		t.Fatal("expected extraction to fail when a parent is a file")
	}
}

func TestExportEmptyWiki(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wiki")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := New(root).ExportTo(&buf, ExportOptions{}); err != nil {
		t.Fatal(err)
	}
	// An empty wiki still exports a valid (empty) zip.
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) != 0 {
		t.Fatalf("empty wiki exported %d entries", len(zr.File))
	}
}
