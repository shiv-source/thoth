package wiki

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type NoteMeta struct {
	Title string
	Date  string
	Kind  string
	Tags  []string
}

// ParseNote splits YAML frontmatter from the body and validates required
// fields. Notes without frontmatter are invalid in a Thoth wiki.
func ParseNote(content []byte) (NoteMeta, []byte, error) {
	var meta NoteMeta
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return meta, nil, fmt.Errorf("frontmatter: note must start with ---")
	}
	end := bytes.Index(content[4:], []byte("\n---"))
	if end < 0 {
		return meta, nil, fmt.Errorf("frontmatter: missing closing ---")
	}
	raw := content[4 : 4+end]
	var fm struct {
		Title string   `yaml:"title"`
		Date  string   `yaml:"date"`
		Type  string   `yaml:"type"`
		Tags  []string `yaml:"tags"`
	}
	if err := yaml.Unmarshal(raw, &fm); err != nil {
		return meta, nil, fmt.Errorf("frontmatter: %w", err)
	}
	meta = NoteMeta{Title: fm.Title, Date: fm.Date, Kind: fm.Type, Tags: fm.Tags}
	if meta.Title == "" {
		return meta, nil, fmt.Errorf("frontmatter: title is required")
	}
	body := bytes.TrimPrefix(content[4+end+4:], []byte("\n"))
	return meta, body, nil
}

// FormatNote renders a note as YAML frontmatter followed by body, in the
// canonical shape the rulebook prescribes (title, date, tags, type). It is the
// write-side half of the note contract: FormatNote output always parses back
// through ParseNote, so any note writer (dashboard, agent, terminal) produces
// a note the whole wiki accepts. Empty optional fields and empty tag lists are
// omitted; a trailing newline is guaranteed.
func FormatNote(meta NoteMeta, body string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: ")
	b.WriteString(yamlScalar(meta.Title))
	b.WriteString("\n")
	if meta.Date != "" {
		b.WriteString("date: ")
		b.WriteString(yamlScalar(meta.Date))
		b.WriteString("\n")
	}
	if len(meta.Tags) > 0 {
		b.WriteString("tags: [")
		for i, tag := range meta.Tags {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(yamlScalar(tag))
		}
		b.WriteString("]\n")
	}
	if meta.Kind != "" {
		b.WriteString("type: ")
		b.WriteString(yamlScalar(meta.Kind))
		b.WriteString("\n")
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
