package wiki

import (
	"bytes"
	"fmt"

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
