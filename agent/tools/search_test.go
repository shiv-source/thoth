package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSearchToolEnforcesLimit(t *testing.T) {
	var gotLimit int
	results := make([]Result, 12)
	for i := range results {
		results[i] = Result{Path: "f.md", Snippet: "s"}
	}
	fn := func(_ context.Context, _ string, limit int) ([]Result, error) {
		gotLimit = limit
		return results, nil
	}
	tl := NewSearch(fn, 5)
	out, err := tl.Run(context.Background(), map[string]any{"query": "q"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if gotLimit != 5 {
		t.Fatalf("SearchFunc limit = %d, want 5", gotLimit)
	}
	if n := strings.Count(out, "\n") + 1; n != 5 {
		t.Fatalf("search output has %d entries, want 5", n)
	}
}

func TestSearchDefaultLimit(t *testing.T) {
	var gotLimit int
	fn := func(_ context.Context, _ string, limit int) ([]Result, error) {
		gotLimit = limit
		return nil, nil
	}
	tl := NewSearch(fn, 0)
	if _, err := tl.Run(context.Background(), map[string]any{"query": "q"}); err != nil {
		t.Fatal(err)
	}
	if gotLimit != searchDefaultLimit {
		t.Fatalf("default limit = %d, want %d", gotLimit, searchDefaultLimit)
	}
}

func TestSearchEmpty(t *testing.T) {
	fn := func(context.Context, string, int) ([]Result, error) { return nil, nil }
	tl := NewSearch(fn, 10)
	out, err := tl.Run(context.Background(), map[string]any{"query": "q"})
	if err != nil || out != "" {
		t.Fatalf("empty search = %q, %v", out, err)
	}
}

func TestSearchFormat(t *testing.T) {
	fn := func(context.Context, string, int) ([]Result, error) {
		return []Result{
			{Path: "a.md", Snippet: "first"},
			{Path: "b.md", Snippet: "second"},
		}, nil
	}
	tl := NewSearch(fn, 10)
	out, err := tl.Run(context.Background(), map[string]any{"query": "q"})
	if err != nil {
		t.Fatal(err)
	}
	want := "1. a.md: first\n2. b.md: second"
	if out != want {
		t.Fatalf("format = %q, want %q", out, want)
	}
}

func TestSearchErrorPropagates(t *testing.T) {
	want := errors.New("index down")
	fn := func(context.Context, string, int) ([]Result, error) { return nil, want }
	tl := NewSearch(fn, 10)
	_, err := tl.Run(context.Background(), map[string]any{"query": "q"})
	if !errors.Is(err, want) {
		t.Fatalf("search error = %v, want wrapped %v", err, want)
	}
	if !strings.Contains(err.Error(), "search:") {
		t.Fatalf("search error not prefixed: %v", err)
	}
}

func TestSearchArgValidation(t *testing.T) {
	fn := func(context.Context, string, int) ([]Result, error) { return nil, nil }
	tl := NewSearch(fn, 10)
	if _, err := tl.Run(context.Background(), map[string]any{}); err == nil {
		t.Fatal("missing query succeeded")
	}
	if _, err := tl.Run(context.Background(), map[string]any{"query": 42}); err == nil {
		t.Fatal("non-string query succeeded")
	}
}

func TestSearchCtxCancelled(t *testing.T) {
	fn := func(context.Context, string, int) ([]Result, error) { return nil, nil }
	tl := NewSearch(fn, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tl.Run(ctx, map[string]any{"query": "q"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("search on cancelled ctx = %v", err)
	}
}
