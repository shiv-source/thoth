package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	agenttools "github.com/shiv-source/thoth/agent/tools"
)

// ValidFolder reports whether folder is an acceptable top-level note folder
// for a save: a single, non-hidden directory name that is not the reserved
// attachments dir and cannot escape the wiki root. It is the shared gate for
// every save-style entry point (the agent's write_note and the promotion
// endpoint), so no caller can write into a path the wiki would not file.
func ValidFolder(folder string) error {
	if folder == "" {
		return errors.New("folder is required")
	}
	if strings.ContainsAny(folder, `/\`) {
		return fmt.Errorf("folder %q must be a single directory name", folder)
	}
	if folder == "." || folder == ".." {
		return fmt.Errorf("folder %q is not allowed", folder)
	}
	if Hidden(folder) {
		return fmt.Errorf("folder %q must not be hidden", folder)
	}
	if folder == AttachmentsDir {
		return fmt.Errorf("folder %q is reserved for attachments", folder)
	}
	return nil
}

// SaveOptions describes a note to promote into the wiki.
type SaveOptions struct {
	Folder    string           // top-level folder (see ValidFolder); type derives from it
	Title     string           // frontmatter title; empty derives from Body
	Body      string           // the note's markdown body
	Tags      []string         // optional frontmatter tags
	SourceURL string           // optional capture provenance; http(s) only, emitted as source: frontmatter
	Now       func() time.Time // clock for the frontmatter date; nil falls back to time.Now
}

// Save writes a new note into Folder/ and returns its wiki-relative path. It
// is the promotion/save path shared by the API: the type comes from the
// folder (NoteType), the filename is a kebab-case slug of the title (with a
// date prefix for the folders the rulebook date-prefixes), the content is
// rendered by FormatNote and checked by Validate, and the write is atomic and
// SafePath-bounded. A note saved here always parses and validates.
func (w *Wiki) Save(o SaveOptions) (string, error) {
	if err := ValidFolder(o.Folder); err != nil {
		return "", err
	}
	if err := ValidSourceURL(o.SourceURL); err != nil {
		return "", err
	}
	title := strings.TrimSpace(o.Title)
	if title == "" {
		title = DefaultTitle(o.Body)
	}
	if title == "" {
		return "", errors.New("note title is empty")
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	date := now().Format("2006-01-02")
	base := Kebab(title)
	if base == "" {
		base = "untitled"
	}
	if datePrefixFolders[o.Folder] {
		base = date + "-" + base
	}
	rel := filepath.ToSlash(filepath.Join(o.Folder, base+".md"))
	content := FormatNote(NoteMeta{
		Title:  title,
		Date:   date,
		Kind:   NoteType(o.Folder),
		Tags:   o.Tags,
		Source: strings.TrimSpace(o.SourceURL),
	}, o.Body)
	if problems := Validate(rel, content); len(problems) > 0 {
		return "", fmt.Errorf("save: %s: %s", problems[0].Rule, problems[0].Msg)
	}
	w.saveMu.Lock()
	defer w.saveMu.Unlock()
	// Two saves can derive the same slug (two quick captures, repeated
	// promotion). Never silently overwrite — pick the next free -N suffix.
	rel = w.nextFreeRel(rel)
	full, err := SafePath(w.Root(), rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("save: create dir: %w", err)
	}
	if err := agenttools.AtomicWrite(full, content, 0o644); err != nil {
		return "", fmt.Errorf("save: write: %w", err)
	}
	return rel, nil
}

// nextFreeRel returns rel, or rel with a -N suffix, for the first path that
// does not already exist under the wiki root. Callers hold saveMu so the
// check-then-write is serialized.
func (w *Wiki) nextFreeRel(rel string) string {
	full, err := SafePath(w.Root(), rel)
	if err != nil {
		return rel
	}
	if _, err := os.Stat(full); errors.Is(err, os.ErrNotExist) {
		return rel
	}
	ext := filepath.Ext(rel)
	stem := strings.TrimSuffix(rel, ext)
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s-%d%s", stem, i, ext)
		full, err := SafePath(w.Root(), cand)
		if err != nil {
			return rel
		}
		if _, err := os.Stat(full); errors.Is(err, os.ErrNotExist) {
			return cand
		}
	}
}

// DefaultTitle derives a note title from a markdown body: the text of the
// first ATX heading (# …), else the first non-empty line, with heading and
// blockquote markers stripped. It is the promotion default, so an answer that
// opens with a heading files under that heading, and a captured selection
// (whose body is a blockquote) files under its first quoted line.
func DefaultTitle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "#") {
			t = strings.TrimSpace(strings.TrimLeft(t, "#"))
			if t == "" {
				continue // bare "#" line — keep scanning for real text
			}
		}
		if strings.HasPrefix(t, ">") {
			t = strings.TrimSpace(strings.TrimPrefix(t, ">"))
			if t == "" {
				continue // bare ">" line — keep scanning for real text
			}
		}
		return t
	}
	return ""
}

// Kebab converts a title into a kebab-case filename stem: lowercase letters
// and digits joined by single hyphens, matching the rulebook's filename rule
// (validate.go kebabCaseRe). Runs of separators collapse to one hyphen; a
// stem with nothing but separators yields "".
func Kebab(title string) string {
	// Cap the stem like the store caps titles (60 runes) so a pathological
	// answer cannot produce an overlong filename.
	runes := []rune(strings.TrimSpace(title))
	if len(runes) > 60 {
		runes = runes[:60]
	}
	var b strings.Builder
	lastDash := false
	for _, r := range runes {
		r = lowerASCII(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if lastDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			lastDash = false
		} else {
			lastDash = true
		}
	}
	return b.String()
}

// lowerASCII maps an upper-ASCII letter to its lowercase form; everything
// else is returned unchanged (non-ASCII letters fall out of the kebab stem,
// matching the rulebook's ASCII-only filename rule).
func lowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
