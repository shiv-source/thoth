package wiki

import "strings"

// defaultFolders is the scaffold folder set used when no custom set is
// configured. It mirrors the layout documented in docs/knowledge-base.md.
var defaultFolders = []string{
	"inbox", "meetings", "projects", "links", "setup", "knowledge", "todos", "daily",
}

// Folders returns a copy of the default scaffold folder names.
func Folders() []string { return append([]string(nil), defaultFolders...) }

// folderLines maps each default folder to its rulebook bullet text (after the
// leading "- "). This map is the single source for the rulebook's folder map;
// a custom folder not listed here falls back to a generic line built from its
// name.
var folderLines = map[string]string{
	"inbox":     "inbox/ — unfiled quick captures. Clean it up: file items into their proper homes, then delete them.",
	"meetings":  "meetings/ — one file per meeting: YYYY-MM-DD-<topic>.md",
	"projects":  "projects/<name>/ — one folder per project. project.md holds overview + current status.",
	"links":     "links/ — bookmarks.md is the master list, grouped by category, one line per link with a one-word reason. Give a link its own note file only when it deserves one.",
	"setup":     "setup/ — one file per machine; setup/servers/<name>.md per server.",
	"knowledge": "knowledge/ — one topic per file.",
	"todos":     "todos/ — TODO.md is the ONLY task list: sections Now / Next / Someday. Never scatter TODOs in other files.",
	"daily":     "daily/ — YYYY-MM-DD.md quick capture journal.",
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
