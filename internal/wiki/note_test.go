package wiki

import (
	"reflect"
	"testing"
)

func TestParseNote(t *testing.T) {
	in := []byte(`---
title: Standup 2026-08-14
date: 2026-08-14
tags: [standup, team]
type: meeting
---
Discussed the release.
`)
	meta, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	want := NoteMeta{Title: "Standup 2026-08-14", Date: "2026-08-14", Kind: "meeting", Tags: []string{"standup", "team"}}
	if !reflect.DeepEqual(meta, want) {
		t.Fatalf("meta mismatch: want %+v got %+v", want, meta)
	}
	if string(body) != "Discussed the release.\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestParseNoteRejectsMissingTitle(t *testing.T) {
	in := []byte("---\ndate: 2026-08-14\n---\nbody\n")
	if _, _, err := ParseNote(in); err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseNoteRejectsUnclosedFrontmatter(t *testing.T) {
	in := []byte("---\ntitle: x\n")
	if _, _, err := ParseNote(in); err == nil {
		t.Fatal("expected error for unclosed frontmatter")
	}
}

func TestParseNoteRejectsMissingFrontmatter(t *testing.T) {
	in := []byte("no frontmatter here\n")
	if _, _, err := ParseNote(in); err == nil {
		t.Fatal("expected error for note without frontmatter")
	}
}

func TestParseNoteRejectsBadYAML(t *testing.T) {
	in := []byte("---\n{{{\ntitle: x\n---\nbody\n")
	if _, _, err := ParseNote(in); err == nil {
		t.Fatal("expected error for malformed YAML frontmatter")
	}
}

func TestParseNoteClosedAtEOF(t *testing.T) {
	// A note whose closing --- lands exactly at EOF (no trailing newline)
	// must parse, with an empty body.
	in := []byte("---\ntitle: Terse\n---")
	meta, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Title != "Terse" {
		t.Fatalf("meta mismatch: %+v", meta)
	}
	if len(body) != 0 {
		t.Fatalf("expected empty body, got %q", body)
	}
}

func TestFormatNoteRoundTrip(t *testing.T) {
	meta := NoteMeta{Title: "Standup sync", Date: "2026-08-21", Kind: "meeting", Tags: []string{"work", "sync"}}
	content := FormatNote(meta, "line one\nline two\n")
	got, body, err := ParseNote(content)
	if err != nil {
		t.Fatalf("ParseNote of FormatNote output: %v\n%s", err, content)
	}
	if !reflect.DeepEqual(got, meta) {
		t.Fatalf("round-trip meta = %+v, want %+v", got, meta)
	}
	if string(body) != "line one\nline two\n" {
		t.Fatalf("round-trip body = %q", body)
	}

	// The canonical shape: title, date, tags, type, closing ---, body.
	want := "---\ntitle: Standup sync\ndate: 2026-08-21\ntags: [work, sync]\ntype: meeting\n---\nline one\nline two\n"
	if string(content) != want {
		t.Fatalf("FormatNote mismatch:\ngot  %q\nwant %q", content, want)
	}
}

func TestFormatNoteOmitsEmptyFieldsAndGuaranteesNewline(t *testing.T) {
	content := FormatNote(NoteMeta{Title: "Bare"}, "body")
	want := "---\ntitle: Bare\n---\nbody\n"
	if string(content) != want {
		t.Fatalf("FormatNote = %q, want %q", content, want)
	}

	// Values that would break YAML unquoted round-trip through ParseNote.
	meta := NoteMeta{Title: "A: note", Tags: []string{"tag one"}}
	got, _, err := ParseNote(FormatNote(meta, ""))
	if err != nil {
		t.Fatalf("ParseNote of quoted output: %v", err)
	}
	if got.Title != "A: note" || !reflect.DeepEqual(got.Tags, []string{"tag one"}) {
		t.Fatalf("round-trip = %+v", got)
	}
}
