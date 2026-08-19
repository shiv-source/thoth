package wiki

import (
	_ "embed"
	"strings"
)

//go:embed templates/CLAUDE.md
var rulebookTemplate string

// folderMapToken marks where the folder-map bullets are inserted. The template
// is otherwise static text.
const folderMapToken = "{{folder-map}}"

// Rulebook returns the CLAUDE.md content for the default folder set. It is the
// single source for the rulebook template.
func Rulebook() string { return RulebookFor(defaultFolders) }

// RulebookFor renders the CLAUDE.md content for a custom folder set, replacing
// the folder-map placeholder with bullets generated from folders. Scaffold
// writes this exact string so the rulebook always matches the layout.
func RulebookFor(folders []string) string {
	return strings.Replace(rulebookTemplate, folderMapToken, folderMap(folders), 1)
}
