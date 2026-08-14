package wiki

import (
	_ "embed"
)

//go:embed templates/CLAUDE.md
var rulebook string

// Rulebook returns the CLAUDE.md content. Scaffold writes this exact string;
// it is the single source for the rulebook template.
func Rulebook() string { return rulebook }
