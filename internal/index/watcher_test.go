package index

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchPicksUpChanges(t *testing.T) {
	root := t.TempDir()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Watch(ctx, root, ix, log) }()
	t.Cleanup(func() { cancel(); <-done })

	// give the watcher time to register the root
	time.Sleep(100 * time.Millisecond)

	p := filepath.Join(root, "daily", "note.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: Watched\ntype: daily\n---\nhello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := ix.Search("hello world", 10)
		if err == nil && len(got) == 1 {
			return // success
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("watcher did not index the new note within 5s")
}

func TestWatchPicksUpNewDirectory(t *testing.T) {
	root := t.TempDir()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Watch(ctx, root, ix, log) }()
	t.Cleanup(func() { cancel(); <-done })

	// give the watcher time to register the root
	time.Sleep(100 * time.Millisecond)

	p := filepath.Join(root, "projects", "newname", "note.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: New Project\ntype: project\n---\nbrand new folder\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := ix.Search("brand new folder", 10)
		if err == nil && len(got) == 1 {
			return // success
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("watcher did not index the note in the new directory within 5s")
}

// TestWatchRemovedDirectoryClearsIndex covers a removal that produces no
// per-file events: the whole subtree must be dropped from the index.
func TestWatchRemovedDirectoryClearsIndex(t *testing.T) {
	root := t.TempDir()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Watch(ctx, root, ix, log) }()
	t.Cleanup(func() { cancel(); <-done })

	time.Sleep(100 * time.Millisecond) // let the watcher register the root

	p := filepath.Join(root, "projects", "doomed", "note.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: Doomed\ntype: project\n---\ngone soon\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := ix.Search("gone soon", 10)
		if err == nil && len(got) == 1 {
			break // indexed
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got, err := ix.Search("gone soon", 10); err != nil || len(got) != 1 {
		t.Fatalf("note not indexed before removal: %v %+v", err, got)
	}

	if err := os.RemoveAll(filepath.Dir(p)); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := ix.Search("gone soon", 10)
		if err == nil && len(got) == 0 {
			return // success
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("removed directory's note still searchable after 5s")
}
