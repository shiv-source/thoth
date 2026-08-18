package index

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/wiki"
)

// waitForChange polls the captured event batches until one contains want.
func waitForChange(t *testing.T, mu *sync.Mutex, batches *[]wiki.Changed, want wiki.Change) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		for _, b := range *batches {
			for _, c := range b.Changes {
				if c == want {
					mu.Unlock()
					return
				}
			}
		}
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no published batch contained %+v within 5s", want)
}

// waitForStartup polls until the watcher's empty startup event arrives —
// it fires after every watch is registered, so tests may write after it.
func waitForStartup(t *testing.T, mu *sync.Mutex, batches *[]wiki.Changed) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(*batches)
		mu.Unlock()
		if n > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("no startup event published within 5s")
}

func TestWatchPicksUpChanges(t *testing.T) {
	root := t.TempDir()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
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
	t.Cleanup(func() { _ = ix.Close() })
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
	t.Cleanup(func() { _ = ix.Close() })
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

// newPublishingWatcher starts Watch with a publisher capturing every event
// batch into a mutex-guarded slice.
func newPublishingWatcher(t *testing.T, root string) (mu *sync.Mutex, batches *[]wiki.Changed) {
	t.Helper()
	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	mu = &sync.Mutex{}
	batches = &[]wiki.Changed{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Watch(ctx, root, ix, log, WithPublisher(func(_ context.Context, c wiki.Changed) {
			mu.Lock()
			defer mu.Unlock()
			*batches = append(*batches, c)
		}))
	}()
	t.Cleanup(func() { cancel(); <-done })
	return mu, batches
}

func TestWatchPublishesChangeBatch(t *testing.T) {
	root := t.TempDir()
	mu, batches := newPublishingWatcher(t, root)
	waitForStartup(t, mu, batches)

	p := filepath.Join(root, "daily", "note.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\ntitle: Watched\ntype: daily\n---\nhello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForChange(t, mu, batches, wiki.Change{Op: wiki.OpCreate, Path: "daily/note.md"})

	if err := os.WriteFile(p, []byte("---\ntitle: Watched\ntype: daily\n---\nupdated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForChange(t, mu, batches, wiki.Change{Op: wiki.OpWrite, Path: "daily/note.md"})

	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	waitForChange(t, mu, batches, wiki.Change{Op: wiki.OpRemove, Path: "daily/note.md"})
}

func TestWatchPublishesNoDotfileNoise(t *testing.T) {
	root := t.TempDir()
	mu, batches := newPublishingWatcher(t, root)

	// wait for the startup event, then count batches
	waitForStartup(t, mu, batches)
	mu.Lock()
	before := len(*batches)
	mu.Unlock()

	if err := os.WriteFile(filepath.Join(root, ".hidden.md"), []byte("noise"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "x.md"), []byte("noise"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("rulebook"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 700ms > startup settle + 200ms debounce: noise must not publish
	time.Sleep(700 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(*batches) != before {
		t.Fatalf("noise paths published events: %+v", (*batches)[before:])
	}
}

func TestWatchPublishesOnStart(t *testing.T) {
	root := t.TempDir()
	mu, batches := newPublishingWatcher(t, root)
	waitForStartup(t, mu, batches)

	mu.Lock()
	got := (*batches)[0]
	mu.Unlock()
	if len(got.Changes) != 0 {
		t.Fatalf("startup event must carry no changes, got %+v", got)
	}
}
