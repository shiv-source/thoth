package wiki

import (
	"bytes"
	"reflect"
	"strings"
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

func TestParseNoteCRLF(t *testing.T) {
	// A note written on Windows (CRLF throughout) parses, and the body keeps
	// its original line endings.
	in := []byte("---\r\ntitle: Standup\r\ndate: 2026-08-14\r\ntags: [standup, team]\r\ntype: meeting\r\n---\r\nDiscussed the release.\r\n")
	meta, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	want := NoteMeta{Title: "Standup", Date: "2026-08-14", Kind: "meeting", Tags: []string{"standup", "team"}}
	if !reflect.DeepEqual(meta, want) {
		t.Fatalf("meta mismatch: want %+v got %+v", want, meta)
	}
	if string(body) != "Discussed the release.\r\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestParseNoteUTF8BOM(t *testing.T) {
	// A leading UTF-8 BOM (as some Windows editors prepend) is stripped
	// before parsing, alone and combined with CRLF.
	bom := []byte{0xEF, 0xBB, 0xBF}
	for _, in := range [][]byte{
		append(bom, []byte("---\ntitle: BOM note\n---\nbody\n")...),
		append(bom, []byte("---\r\ntitle: BOM note\r\n---\r\nbody\r\n")...),
	} {
		meta, body, err := ParseNote(in)
		if err != nil {
			t.Fatalf("ParseNote: %v", err)
		}
		if meta.Title != "BOM note" {
			t.Fatalf("meta mismatch: %+v", meta)
		}
		if !bytes.HasPrefix(body, []byte("body")) {
			t.Fatalf("body mismatch: %q", body)
		}
	}
}

func TestParseNoteDotsTerminator(t *testing.T) {
	// The YAML `...` document terminator closes frontmatter too.
	in := []byte("---\ntitle: Terse\n...\nbody\n")
	meta, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Title != "Terse" {
		t.Fatalf("meta mismatch: %+v", meta)
	}
	if string(body) != "body\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestParseNoteNestedFenceInMultilineValue(t *testing.T) {
	// A `---` line inside a multiline YAML value is not a fence: only a
	// column-0 line that is exactly `---` (or `...`) closes frontmatter.
	in := []byte(`---
title: Long
summary: |
  Part one
  ---
  Part three
type: knowledge
---
The body.
`)
	meta, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Title != "Long" || meta.Kind != "knowledge" {
		t.Fatalf("meta mismatch: %+v", meta)
	}
	if string(body) != "The body.\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestParseNoteKindAlias(t *testing.T) {
	// `kind:` is an accepted alias for the canonical `type:` key.
	in := []byte("---\ntitle: Alias\nkind: meeting\n---\nbody\n")
	meta, _, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Kind != "meeting" {
		t.Fatalf("meta mismatch: %+v", meta)
	}
}

func TestParseNoteTypeKindDisagree(t *testing.T) {
	in := []byte("---\ntitle: Conflicted\ntype: meeting\nkind: knowledge\n---\nbody\n")
	if _, _, err := ParseNote(in); err == nil {
		t.Fatal("expected error for disagreeing type and kind")
	}
}

func TestParseNoteFieldValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string // substring expected in the error
	}{
		{
			name: "date wrong format",
			body: "---\ntitle: X\ndate: August 2026\n---\nbody\n",
			want: "not YYYY-MM-DD",
		},
		{
			name: "date not a string",
			body: "---\ntitle: X\ndate: 2026\n---\nbody\n",
			want: "date",
		},
		{
			name: "type is a list",
			body: "---\ntitle: X\ntype: [meeting, daily]\n---\nbody\n",
			want: "type must be a string",
		},
		{
			name: "type is a number",
			body: "---\ntitle: X\ntype: 42\n---\nbody\n",
			want: "type must be a string",
		},
		{
			name: "type has whitespace",
			body: "---\ntitle: X\ntype: \"meeting notes\"\n---\nbody\n",
			want: "single token",
		},
		{
			name: "tags not a list",
			body: "---\ntitle: X\ntags: standalone\n---\nbody\n",
			want: "tags must be a list",
		},
		{
			name: "tag is a nested list",
			body: "---\ntitle: X\ntags: [[a]]\n---\nbody\n",
			want: "must be a string",
		},
		{
			name: "empty tag",
			body: "---\ntitle: X\ntags: [\"\", b]\n---\nbody\n",
			want: "must not be empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseNote([]byte(tt.body))
			if err == nil {
				t.Fatalf("ParseNote accepted %s: %q", tt.name, tt.body)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ParseNote error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestParseNoteEmptyDateIsValid(t *testing.T) {
	// A note with no date at all is fine — date is optional.
	in := []byte("---\ntitle: Dateless\n---\nbody\n")
	meta, _, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Date != "" {
		t.Fatalf("meta mismatch: %+v", meta)
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
