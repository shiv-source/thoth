package wiki

import "strings"

// AttachmentsDir is the reserved directory for non-markdown assets (images,
// scripts, configs). It is app-managed: the server recreates it if missing
// (see EnsureReservedDir), and the rulebook tells Claude never to delete it.
const AttachmentsDir = "attachments"

// defaultFolders is the scaffold folder set used when no custom set is
// configured. It mirrors the layout documented in docs/knowledge-base.md.
var defaultFolders = []string{
	"inbox", "meetings", "projects", "links", "setup", "knowledge", "todos", "daily", AttachmentsDir,
}

// Folders returns a copy of the default scaffold folder names.
func Folders() []string { return append([]string(nil), defaultFolders...) }

// noteType maps a folder name to its frontmatter `type:` value. Default
// folders are plural directory names with a singular type (meetings ->
// meeting); custom folders fall back to the same rule (recipes -> recipe).
// The rule is deliberately mechanical so the type list can never drift from
// the folder list. The reserved attachments dir is not a note type.
func noteType(folder string) string {
	return strings.TrimSuffix(folder, "s")
}

// NoteTypesFor returns the frontmatter `type:` values for the given folders,
// derived from the folder names (see noteType). The attachments dir is not a
// note type. Order follows the folder order.
func NoteTypesFor(folders []string) []string {
	types := make([]string, 0, len(folders))
	for _, f := range folders {
		if f == AttachmentsDir {
			continue
		}
		types = append(types, noteType(f))
	}
	return types
}

// NoteTypes returns the default frontmatter `type:` values.
func NoteTypes() []string { return NoteTypesFor(defaultFolders) }

// folderLines maps each default folder to its rulebook bullet text (after the
// leading "- "). This map is the single source for the rulebook's folder map;
// a custom folder not listed here falls back to a generic line built from its
// name.
var folderLines = map[string]string{
	"inbox":       "inbox/ — unfiled quick captures. Clean it up: file items into their proper homes, then delete them.",
	"meetings":    "meetings/ — one file per meeting: YYYY-MM-DD-<topic>.md",
	"projects":    "projects/<name>/ — one folder per project. project.md holds overview + current status.",
	"links":       "links/ — bookmarks.md is the master list, grouped by category, one line per link with a one-word reason. Give a link its own note file only when it deserves one.",
	"setup":       "setup/ — one file per machine; setup/servers/<name>.md per server.",
	"knowledge":   "knowledge/ — one topic per file.",
	"todos":       "todos/ — TODO.md is the ONLY task list: sections Now / Next / Someday. Never scatter TODOs in other files.",
	"daily":       "daily/ — YYYY-MM-DD.md quick capture journal.",
	"attachments": "attachments/ — non-markdown assets (images, scripts, configs). Save the file here, and write a companion note in the folder that uses it (e.g. setup/servers/x.md for attachments/x.yaml) so its content stays searchable.",
}

// folderMap renders the rulebook's folder-map bullets for the given folders.
func folderMap(folders []string) string {
	var b strings.Builder
	for _, f := range folders {
		line, ok := folderLines[f]
		if !ok {
			line = f + "/ — custom workspace folder."
		}
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
