package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSyncIndexesTree(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("meetings/ok.md", "---\ntitle: Good\ntype: meeting\n---\nvalid body\n")
	write("knowledge/bad.md", "no frontmatter at all\n")
	write("meetings/ignored.txt", "not markdown\n")

	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	got, err := ix.Search("valid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "meetings/ok.md" {
		t.Fatalf("expected exactly the valid note indexed, got %+v", got)
	}
}

func TestSyncRemovesDeletedNotes(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "meetings", "ok.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: Good\ntype: meeting\n---\nvalid body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	got, err := ix.Search("valid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "meetings/ok.md" {
		t.Fatalf("expected the valid note indexed, got %+v", got)
	}

	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatalf("Sync after delete: %v", err)
	}
	got, err = ix.Search("valid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected zero results after note deleted, got %+v", got)
	}
}

// TestSyncSkipsUnchangedFiles is the point of Sync over a full rebuild: a
// file whose mtime still matches the stored updated_at is not re-read or
// re-indexed, so its stale body would survive if it were changed without a
// new mtime — and it is refreshed once the mtime moves.
func TestSyncSkipsUnchangedFiles(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "note.md")
	if err := os.WriteFile(p, []byte("---\ntitle: Note\n---\nalpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatal(err)
	}
	got, err := ix.Search("alpha", 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("expected alpha indexed: %v %+v", err, got)
	}

	// Same mtime, different content: the sync must skip the file, keeping
	// the old body in the index.
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: Note\n---\nbeta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatal(err)
	}
	got, err = ix.Search("alpha", 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("unchanged-mtime file was re-indexed: %v %+v", err, got)
	}
	if got, err = ix.Search("beta", 10); err != nil || len(got) != 0 {
		t.Fatalf("skipped file should keep old body: %v %+v", err, got)
	}

	// A genuinely new mtime re-indexes the file.
	time.Sleep(1100 * time.Millisecond) // exceed mtime's second granularity
	if err := os.WriteFile(p, []byte("---\ntitle: Note\n---\ngamma\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatal(err)
	}
	got, err = ix.Search("gamma", 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("expected gamma after new mtime: %v %+v", err, got)
	}
	if got, err = ix.Search("alpha", 10); err != nil || len(got) != 0 {
		t.Fatalf("stale body survived re-index: %v %+v", err, got)
	}
}

func TestSyncErrorOnMissingRoot(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	if err := ix.Sync(filepath.Join(t.TempDir(), "missing"), discardLog()); err == nil {
		t.Fatal("expected error walking a missing root")
	}
}

func TestSyncSkipsUnreadableFile(t *testing.T) {
	root := t.TempDir()
	// ReadFile on a directory errors, and the sync must log and skip it.
	if err := os.MkdirAll(filepath.Join(root, "meetings", "weird.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	if err := ix.Sync(root, discardLog()); err != nil {
		t.Fatalf("Sync must skip unreadable files, got %v", err)
	}
}
