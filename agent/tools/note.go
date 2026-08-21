package tools

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Note is the frontmatter metadata of a wiki note: title, date (YYYY-MM-DD),
// type and tags. It mirrors the Thoth wiki contract (internal/wiki.NoteMeta)
// exactly so notes written by the agent parse in the wiki.
type Note struct {
	Title string
	Date  string
	Type  string
	Tags  []string
}

// HasMarkdownExt reports whether path names a markdown note (.md or
// .markdown, case-insensitive).
func HasMarkdownExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

// ParseNote splits YAML frontmatter from the body and validates required
// fields. Notes without frontmatter, or without a title, are invalid in a
// Thoth wiki — this mirrors internal/wiki.ParseNote so both agree on what a
// note is.
func ParseNote(content []byte) (Note, []byte, error) {
	var note Note
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return note, nil, fmt.Errorf("frontmatter: note must start with ---")
	}
	end := bytes.Index(content[4:], []byte("\n---"))
	if end < 0 {
		return note, nil, fmt.Errorf("frontmatter: missing closing ---")
	}
	raw := content[4 : 4+end]
	var fm struct {
		Title string   `yaml:"title"`
		Date  string   `yaml:"date"`
		Type  string   `yaml:"type"`
		Tags  []string `yaml:"tags"`
	}
	if err := yaml.Unmarshal(raw, &fm); err != nil {
		return note, nil, fmt.Errorf("frontmatter: %w", err)
	}
	note = Note{Title: fm.Title, Date: fm.Date, Type: fm.Type, Tags: fm.Tags}
	if note.Title == "" {
		return note, nil, fmt.Errorf("frontmatter: title is required")
	}
	body := bytes.TrimPrefix(content[4+end+4:], []byte("\n"))
	return note, body, nil
}

// FormatNote renders a Note as YAML frontmatter followed by body, matching the
// wiki's canonical note shape: title, date, type, tags on their own lines,
// with tags in flow style ([a, b]) and a single trailing newline. The result
// parses back through ParseNote and the wiki's internal ParseNote.
func FormatNote(n Note, body string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: ")
	b.WriteString(yamlScalar(n.Title))
	b.WriteString("\n")
	if n.Date != "" {
		b.WriteString("date: ")
		b.WriteString(yamlScalar(n.Date))
		b.WriteString("\n")
	}
	if n.Type != "" {
		b.WriteString("type: ")
		b.WriteString(yamlScalar(n.Type))
		b.WriteString("\n")
	}
	if len(n.Tags) > 0 {
		b.WriteString("tags: [")
		for i, tag := range n.Tags {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(yamlScalar(tag))
		}
		b.WriteString("]\n")
	}
	b.WriteString("---\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// yamlScalar renders s as a YAML scalar, quoting it only when the plain form
// would round-trip to a different value or fail to parse. This keeps dates and
// plain words unquoted while safely quoting values that carry YAML meaning.
func yamlScalar(s string) string {
	if plainYAMLScalar(s) {
		return s
	}
	out, err := yaml.Marshal(s)
	if err != nil {
		return s
	}
	return strings.TrimSuffix(string(out), "\n")
}

// plainYAMLScalar reports whether s can be written unquoted and still parse
// back to itself as a string.
func plainYAMLScalar(s string) bool {
	if s == "" || strings.TrimSpace(s) != s || strings.ContainsAny(s, "\n") {
		return false
	}
	var out string
	if err := yaml.Unmarshal([]byte(s), &out); err != nil {
		return false
	}
	return out == s
}
