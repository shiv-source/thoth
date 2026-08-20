package wiki

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		rel    string
		body   string
		wantOK bool
		want   string // substring expected in the first problem message
	}{
		{
			name:   "valid note",
			rel:    "knowledge/rust-notes.md",
			body:   "---\ntitle: Rust Notes\ntype: knowledge\n---\nbody\n",
			wantOK: true,
		},
		{
			name: "missing frontmatter",
			rel:  "knowledge/bad.md",
			body: "no frontmatter here\n",
			want: "frontmatter",
		},
		{
			name: "missing title",
			rel:  "knowledge/bad.md",
			body: "---\ntype: knowledge\n---\nbody\n",
			want: "title is required",
		},
		{
			name: "type missing",
			rel:  "knowledge/topic.md",
			body: "---\ntitle: Topic\n---\nbody\n",
			want: "type is missing",
		},
		{
			name: "type does not match folder",
			rel:  "meetings/2026-08-14-standup.md",
			body: "---\ntitle: Standup\ntype: knowledge\n---\nbody\n",
			want: "does not match folder meetings",
		},
		{
			name:   "legacy note type tolerated",
			rel:    "knowledge/topic.md",
			body:   "---\ntitle: Topic\ntype: note\n---\nbody\n",
			wantOK: true,
		},
		{
			name: "non-kebab filename",
			rel:  "knowledge/My Note.md",
			body: "---\ntitle: My Note\ntype: knowledge\n---\nbody\n",
			want: "not kebab-case",
		},
		{
			name:   "uppercase extension is a note",
			rel:    "meetings/2026-08-14-standup.MD",
			body:   "---\ntitle: Standup\ntype: meeting\n---\nbody\n",
			wantOK: true,
		},
		{
			name: "meeting note without date prefix",
			rel:  "meetings/standup.md",
			body: "---\ntitle: Standup\ntype: meeting\n---\nbody\n",
			want: "start with YYYY-MM-DD",
		},
		{
			name:   "meeting note with date prefix",
			rel:    "meetings/2026-08-14-standup.md",
			body:   "---\ntitle: Standup\ntype: meeting\n---\nbody\n",
			wantOK: true,
		},
		{
			name: "daily note without date prefix",
			rel:  "daily/journal.md",
			body: "---\ntitle: Journal\ntype: daily\n---\nbody\n",
			want: "start with YYYY-MM-DD",
		},
		{
			name:   "TODO master list is exempt from kebab-case",
			rel:    "todos/TODO.md",
			body:   "---\ntitle: TODO\ntype: todo\n---\nbody\n",
			wantOK: true,
		},
		{
			name:   "nested note type matches its top folder",
			rel:    "projects/thoth/status.md",
			body:   "---\ntitle: Status\ntype: project\n---\nbody\n",
			wantOK: true,
		},
		{
			name:   "root-level note skips the type rule",
			rel:    "note.md",
			body:   "---\ntitle: Note\n---\nbody\n",
			wantOK: true,
		},
		{
			name:   "attachments are never validated as notes",
			rel:    "attachments/install.sh",
			body:   "#!/bin/sh\n",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := Validate(tt.rel, []byte(tt.body))
			if tt.wantOK {
				if len(problems) != 0 {
					t.Fatalf("Validate(%q) = %+v, want no problems", tt.rel, problems)
				}
				return
			}
			if len(problems) == 0 {
				t.Fatalf("Validate(%q) = no problems, want one containing %q", tt.rel, tt.want)
			}
			if !strings.Contains(problems[0].Msg, tt.want) {
				t.Fatalf("Validate(%q) problem = %q, want it to contain %q", tt.rel, problems[0].Msg, tt.want)
			}
			if problems[0].Path != tt.rel {
				t.Fatalf("Validate problem path = %q, want %q", problems[0].Path, tt.rel)
			}
		})
	}
}

func TestValidateReportsMultipleProblems(t *testing.T) {
	// A non-kebab meeting note with a missing type trips three rules.
	problems := Validate("meetings/My Standup.md", []byte("---\ntitle: Standup\n---\nbody\n"))
	if len(problems) < 3 {
		t.Fatalf("expected type, filename, and date-prefix problems, got %+v", problems)
	}
	rules := map[string]bool{}
	for _, p := range problems {
		rules[p.Rule] = true
	}
	for _, rule := range []string{"type", "filename", "date-prefix"} {
		if !rules[rule] {
			t.Fatalf("missing %q problem in %+v", rule, problems)
		}
	}
}
