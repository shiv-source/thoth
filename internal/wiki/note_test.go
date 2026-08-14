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
