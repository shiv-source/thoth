package wiki

import (
	"path"
	"regexp"
	"strings"
)

// Problem is a single save-protocol violation found by Validate: which file,
// which rule, and why it failed. Rules mirror the rulebook (see
// templates/CLAUDE.md): frontmatter correctness, filename shape, and the
// type/folder pairing.
type Problem struct {
	Path string
	Rule string
	Msg  string
}

// legacyTypes are frontmatter `type:` values from before the type list was
// derived from the folder layout. They are tolerated without a warning so
// existing notes keep validating cleanly; new notes should use the derived
// type for their folder.
var legacyTypes = map[string]bool{"note": true}

// datePrefixFolders are the folders whose rulebook line requires a date
// prefix on the filename (YYYY-MM-DD). Violations are warning-only — a note
// without its date prefix still indexes, it just may not sort chronologically.
var datePrefixFolders = map[string]bool{"meetings": true, "daily": true}

var (
	kebabCaseRe  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	datePrefixRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}`)
)

// Validate checks a markdown note against the save protocol derived from the
// wiki's own rules. It returns every problem found; a nil/empty result means
// the note follows the protocol. Validation is advisory — callers decide
// whether a problem skips the note or just warns — so a missing frontmatter
// `type` or a non-kebab filename never hard-fails a read that already works.
func Validate(rel string, content []byte) []Problem {
	if !IsMarkdownPath(rel) {
		return nil // attachments are indexed by filename, never validated as notes
	}
	problems := []Problem{}
	add := func(rule, msg string) {
		problems = append(problems, Problem{Path: rel, Rule: rule, Msg: msg})
	}

	meta, _, err := ParseNote(content)
	folder := topFolder(rel)
	if err != nil {
		add("frontmatter", err.Error())
	} else if folder != "" {
		want := noteType(folder)
		switch {
		case meta.Kind == "":
			add("type", "frontmatter type is missing; expected "+want)
		case meta.Kind != want && !legacyTypes[meta.Kind]:
			add("type", "frontmatter type "+meta.Kind+" does not match folder "+folder+"/; expected "+want)
		}
	}

	// path.Base/Ext not filepath: rel is slash-separated by contract (see
	// SafePath and the index's ToSlash), and filepath uses the platform
	// separator, so it would mis-split on Windows.
	base := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	if base != "TODO" && !kebabCaseRe.MatchString(base) {
		add("filename", "filename "+base+" is not kebab-case (lowercase letters, digits, hyphens)")
	}
	if datePrefixFolders[folder] && !datePrefixRe.MatchString(base) {
		add("date-prefix", "notes in "+folder+"/ start with YYYY-MM-DD, got "+base)
	}
	return problems
}

// topFolder returns the first path segment of a wiki-relative path, or "" for
// a file at the wiki root.
func topFolder(rel string) string {
	idx := strings.Index(rel, "/")
	if idx < 0 {
		return ""
	}
	return rel[:idx]
}
