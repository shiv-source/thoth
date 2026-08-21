package wiki

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// NoteMeta is a note's frontmatter as the wiki knows it. The note type has a
// single canonical YAML key — `type` (what the rulebook and FormatNote write)
// — with `kind` accepted as an alias; both map onto this field.
type NoteMeta struct {
	Title string
	Date  string
	Kind  string
	Tags  []string
}

// ParseNote splits YAML frontmatter from the body and validates the fields.
// Notes without frontmatter are invalid in a Thoth wiki. Parsing tolerates a
// UTF-8 BOM and CRLF line endings, finds the closing fence (--- or ...) by
// line, and accepts `kind:` as an alias for the canonical `type:` key. Bad
// `date`/`type`/`tags` values fail loudly instead of being indexed silently.
func ParseNote(content []byte) (NoteMeta, []byte, error) {
	var meta NoteMeta
	raw, body, err := splitFrontmatter(content)
	if err != nil {
		return meta, nil, err
	}
	var fm struct {
		Title yaml.Node `yaml:"title"`
		Date  yaml.Node `yaml:"date"`
		Type  yaml.Node `yaml:"type"`
		Kind  yaml.Node `yaml:"kind"`
		Tags  yaml.Node `yaml:"tags"`
	}
	if err := yaml.Unmarshal(raw, &fm); err != nil {
		return meta, nil, fmt.Errorf("frontmatter: %w", err)
	}
	if meta.Title, err = scalarString(fm.Title); err != nil {
		return meta, nil, fmt.Errorf("frontmatter: title: %w", err)
	}
	if meta.Title == "" {
		return meta, nil, errors.New("frontmatter: title is required")
	}
	if meta.Date, err = noteDate(fm.Date); err != nil {
		return meta, nil, fmt.Errorf("frontmatter: %w", err)
	}
	if meta.Kind, err = noteTypeValue(fm.Type, fm.Kind); err != nil {
		return meta, nil, fmt.Errorf("frontmatter: %w", err)
	}
	if meta.Tags, err = noteTags(fm.Tags); err != nil {
		return meta, nil, fmt.Errorf("frontmatter: %w", err)
	}
	return meta, body, nil
}

// splitFrontmatter splits content into the raw YAML between the opening and
// closing fences and the body after the closing fence. A UTF-8 BOM is
// stripped first and both \n and \r\n line endings are accepted. The closing
// fence is the first later line that is exactly `---` or `...` (optional
// trailing whitespace allowed) — lines inside multiline YAML values are
// indented, so they never match.
func splitFrontmatter(content []byte) (raw, body []byte, err error) {
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	line, rest := splitLine(content)
	if !isFence(line) {
		return nil, nil, errors.New("frontmatter: note must start with ---")
	}
	for cur := rest; ; {
		l, r := splitLine(cur)
		if isFence(l) {
			return rest[:len(rest)-len(cur)], r, nil
		}
		if len(r) == 0 {
			return nil, nil, errors.New("frontmatter: missing closing fence (--- or ...)")
		}
		cur = r
	}
}

// splitLine returns the first line of b without its terminator (or trailing
// \r) and the remainder after the newline. The final line carries no
// newline, so it is returned with a nil remainder.
func splitLine(b []byte) (line, rest []byte) {
	i := bytes.IndexByte(b, '\n')
	if i < 0 {
		return b, nil
	}
	line = b[:i]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, b[i+1:]
}

// isFence reports whether line is a frontmatter fence at column 0: `---` or
// `...`, optionally followed by whitespace.
func isFence(line []byte) bool {
	trimmed := bytes.TrimRight(line, " \t")
	return bytes.Equal(trimmed, []byte("---")) || bytes.Equal(trimmed, []byte("..."))
}

// scalarString returns a scalar node's value as a string, coercing non-string
// scalars exactly like decoding into a string field would. An absent or null
// node yields "".
func scalarString(n yaml.Node) (string, error) {
	if n.Kind == 0 || n.Tag == "!!null" {
		return "", nil
	}
	if n.Kind != yaml.ScalarNode {
		return "", errors.New("must be a string")
	}
	return n.Value, nil
}

// noteDate validates the frontmatter date: absent/null is fine, present must
// be a YYYY-MM-DD calendar date.
func noteDate(n yaml.Node) (string, error) {
	if n.Kind == 0 || n.Tag == "!!null" {
		return "", nil
	}
	if n.Kind != yaml.ScalarNode {
		return "", errors.New("date must be a YYYY-MM-DD string")
	}
	if _, err := time.Parse("2006-01-02", n.Value); err != nil {
		return "", fmt.Errorf("date %q is not YYYY-MM-DD", n.Value)
	}
	return n.Value, nil
}

// noteTypeValue resolves the note type from the canonical `type:` key and the
// `kind:` alias. When both are present they must agree; the winner must be a
// plain string scalar (never a list, number, or bool) with no whitespace.
func noteTypeValue(typeN, kindN yaml.Node) (string, error) {
	typ, err := typeToken(typeN)
	if err != nil {
		return "", err
	}
	kind, err := typeToken(kindN)
	if err != nil {
		return "", err
	}
	if typ != "" && kind != "" && typ != kind {
		return "", fmt.Errorf("type %q and kind %q disagree", typ, kind)
	}
	if typ != "" {
		return typ, nil
	}
	return kind, nil
}

// typeToken validates one type/kind value: absent/null is "", otherwise a
// string scalar with no whitespace.
func typeToken(n yaml.Node) (string, error) {
	if n.Kind == 0 || n.Tag == "!!null" {
		return "", nil
	}
	if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
		return "", errors.New("type must be a string")
	}
	if strings.ContainsAny(n.Value, " \t\r\n") {
		return "", fmt.Errorf("type %q must be a single token", n.Value)
	}
	return n.Value, nil
}

// noteTags validates the frontmatter tags: absent/null is nil, otherwise a
// sequence of non-empty scalar values (numbers and dates are coerced to
// strings, matching how the index joins them).
func noteTags(n yaml.Node) ([]string, error) {
	if n.Kind == 0 || n.Tag == "!!null" {
		return nil, nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, errors.New("tags must be a list")
	}
	tags := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode || item.Tag == "!!null" {
			return nil, fmt.Errorf("tag %q must be a string", item.Value)
		}
		if strings.TrimSpace(item.Value) == "" {
			return nil, errors.New("tags must not be empty")
		}
		tags = append(tags, item.Value)
	}
	return tags, nil
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
