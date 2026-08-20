package agent

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

// discardLog returns a logger that writes nowhere.
func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeStream replays a canned list of deltas, then reports usage.
type fakeStream struct {
	deltas []agentlib.Delta
	usage  agentlib.Usage
	closed bool
}

func (s *fakeStream) Next() (agentlib.Delta, error) {
	if len(s.deltas) == 0 {
		return agentlib.Delta{}, io.EOF
	}
	d := s.deltas[0]
	s.deltas = s.deltas[1:]
	return d, nil
}

func (s *fakeStream) Usage() agentlib.Usage { return s.usage }
func (s *fakeStream) Close() error          { s.closed = true; return nil }

// fakeProvider records the request it was given and streams back a canned
// turn. It implements the Provider seam using only the public agent API. A
// non-nil err is returned by Stream instead.
type fakeProvider struct {
	stream *fakeStream
	req    agentlib.Request
	err    error
}

func (p *fakeProvider) Stream(ctx context.Context, req agentlib.Request) (agentlib.Stream, error) {
	p.req = req
	if p.err != nil {
		return nil, p.err
	}
	return p.stream, nil
}

// openStore opens a store on a fresh temp db.
func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// openIndex opens an index on a fresh temp db whose schema was created by the
// store's migrations.
func openIndex(t *testing.T) *index.Index {
	t.Helper()
	db := filepath.Join(t.TempDir(), "index.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	ix, err := index.Open(db)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	return ix
}

// newWiki makes a wiki rooted at a fresh temp dir with a CLAUDE.md rulebook.
func newWiki(t *testing.T, rulebook string) *wiki.Wiki {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(rulebook), 0o644); err != nil {
		t.Fatalf("write rulebook: %v", err)
	}
	return wiki.New(dir)
}

// writeRulebook rewrites the wiki's CLAUDE.md in place.
func writeRulebook(t *testing.T, w *wiki.Wiki, content string) error {
	t.Helper()
	return os.WriteFile(filepath.Join(w.Root, "CLAUDE.md"), []byte(content), 0o644)
}
