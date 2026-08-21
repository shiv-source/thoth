package tools

import (
	"bytes"
	"testing"
)

func TestParseNote(t *testing.T) {
	cases := []struct {
		name    string
		content string
		note    Note
		body    string
		wantErr bool
	}{
		{
			name:    "full frontmatter",
			content: "---\ntitle: Standup\ndate: 2026-08-21\ntype: meeting\ntags: [work, sync]\n---\nbody text",
			note:    Note{Title: "Standup", Date: "2026-08-21", Type: "meeting", Tags: []string{"work", "sync"}},
			body:    "body text",
		},
		{
			name:    "minimal frontmatter",
			content: "---\ntitle: A Note\n---\nhello",
			note:    Note{Title: "A Note"},
			body:    "hello",
		},
		{
			name:    "body after blank line",
			content: "---\ntitle: X\n---\n\nbody",
			note:    Note{Title: "X"},
			body:    "\nbody",
		},
		{
			name:    "missing frontmatter",
			content: "no frontmatter",
			wantErr: true,
		},
		{
			name:    "missing closing marker",
			content: "---\ntitle: X",
			wantErr: true,
		},
		{
			name:    "empty title",
			content: "---\n---\nbody",
			wantErr: true,
		},
		{
			name:    "unparsable yaml",
			content: "---\ntitle: [\n---",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			note, body, err := ParseNote([]byte(tc.content))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseNote(%q) succeeded, want error", tc.content)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseNote: %v", err)
			}
			if !notesEqual(note, tc.note) {
				t.Fatalf("note = %+v, want %+v", note, tc.note)
			}
			if string(body) != tc.body {
				t.Fatalf("body = %q, want %q", body, tc.body)
			}
		})
	}
}

func TestFormatNoteRoundTrip(t *testing.T) {
	note := Note{
		Title: "Standup sync",
		Date:  "2026-08-21",
		Type:  "meeting",
		Tags:  []string{"work", "sync"},
	}
	content := FormatNote(note, "line one\nline two\n")
	got, body, err := ParseNote(content)
	if err != nil {
		t.Fatalf("ParseNote of FormatNote output: %v\n%s", err, content)
	}
	if !notesEqual(got, note) {
		t.Fatalf("round-trip note = %+v, want %+v", got, note)
	}
	if string(body) != "line one\nline two\n" {
		t.Fatalf("round-trip body = %q", body)
	}

	// The generated frontmatter must match the canonical wiki shape: title,
	// date, type, tags in flow style, closing --- on its own line.
	want := "---\ntitle: Standup sync\ndate: 2026-08-21\ntype: meeting\ntags: [work, sync]\n---\nline one\nline two\n"
	if !bytes.Equal(content, []byte(want)) {
		t.Fatalf("FormatNote output mismatch:\ngot:  %q\nwant: %q", content, want)
	}
}

func TestFormatNoteQuoting(t *testing.T) {
	// Values that would break YAML unquoted must round-trip through the wiki
	// parser even though they are written quoted.
	note := Note{Title: "A: note", Date: "2026-08-21", Tags: []string{"tag one"}}
	content := FormatNote(note, "")
	got, _, err := ParseNote(content)
	if err != nil {
		t.Fatalf("ParseNote: %v\n%s", err, content)
	}
	if got.Title != "A: note" || got.Tags[0] != "tag one" {
		t.Fatalf("round-trip = %+v", got)
	}
}

func TestHasMarkdownExt(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"note.md", true},
		{"dir/note.markdown", true},
		{"NOTE.MD", true},
		{"note.txt", false},
		{"note", false},
		{"note.mdd", false},
	}
	for _, tc := range cases {
		if got := HasMarkdownExt(tc.path); got != tc.want {
			t.Fatalf("HasMarkdownExt(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// notesEqual compares two Notes field-by-field ([]string is not comparable).
func notesEqual(a, b Note) bool {
	if a.Title != b.Title || a.Date != b.Date || a.Type != b.Type || len(a.Tags) != len(b.Tags) {
		return false
	}
	for i := range a.Tags {
		if a.Tags[i] != b.Tags[i] {
			return false
		}
	}
	return true
}
