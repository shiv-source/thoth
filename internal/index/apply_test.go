package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestApply covers the branch table of apply() directly — deterministic and
// fast, without waiting on filesystem events.
func TestApply(t *testing.T) {
	root := t.TempDir()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	log := discardLog()

	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("daily/ok.md", "---\ntitle: Good\ntype: daily\n---\nindexed by apply\n")
	write("daily/bad.md", "no frontmatter\n")

	// non-.md files are ignored
	apply(ix, root, filepath.Join(root, "daily", "note.txt"), log)

	// a valid note gets upserted
	apply(ix, root, filepath.Join(root, "daily", "ok.md"), log)
	got, err := ix.Search("indexed by apply", 10)
	if err != nil || len(got) != 1 || got[0].Path != "daily/ok.md" {
		t.Fatalf("apply did not index the note: %v %+v", err, got)
	}

	// malformed notes are skipped
	apply(ix, root, filepath.Join(root, "daily", "bad.md"), log)
	got, err = ix.Search("no frontmatter", 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("apply indexed a malformed note: %v %+v", err, got)
	}

	// a deleted file is removed from the index
	gone := filepath.Join(root, "daily", "ok.md")
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	apply(ix, root, gone, log)
	got, err = ix.Search("indexed by apply", 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("apply did not delete the removed note: %v %+v", err, got)
	}
}

func TestApplyUnreadablePath(t *testing.T) {
	root := t.TempDir()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	// A directory named like a note: ReadFile fails, apply must not panic.
	dir := filepath.Join(root, "weird.md")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	apply(ix, root, dir, discardLog())
}

func TestApplyPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	p := filepath.Join(other, "x.md")
	if err := os.WriteFile(p, []byte("---\ntitle: X\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	// filepath.Rel(root, p) fails for unrelated roots; apply must return.
	apply(ix, root, p, discardLog())
}

func TestApplyClosedIndexLogsAndContinues(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "a.md")
	if err := os.WriteFile(p, []byte("---\ntitle: A\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}
	// Upsert fails against the closed DB; apply must log and return, not crash.
	apply(ix, root, p, discardLog())
	apply(ix, root, filepath.Join(root, "gone.md"), discardLog())
}

func TestWatchErrorOnMissingRoot(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := Watch(ctx, filepath.Join(t.TempDir(), "missing"), ix, discardLog()); err == nil {
		t.Fatal("expected error watching a missing root")
	}
}

func TestWatchReturnsOnCancel(t *testing.T) {
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Watch(ctx, root, ix, discardLog()) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected context error from Watch")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Watch did not return after cancel")
	}
}
