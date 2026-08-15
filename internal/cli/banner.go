package cli

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// isTerminal reports whether f is a character device — a real terminal —
// so the banner only uses ANSI colors when they will actually be seen.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// startupBanner renders the terminal panel printed when the server comes up:
// a boxed panel with the version, UI URL, and wiki path. With color=true the
// URL is bold green (callers pass true only when stderr is a TTY).
func startupBanner(version, host string, port int, wikiPath string, color bool) string {
	u := (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, strconv.Itoa(port))}).String() + "/"
	title := fmt.Sprintf("Thoth %s — ready", version)
	plainUI := "UI:   " + u
	plainWiki := "Wiki: " + wikiPath

	// Geometry is computed in runes from the plain text only — ANSI codes
	// and multi-byte characters must never change the box shape.
	runes := utf8.RuneCountInString
	inner := runes(title)
	if n := runes(plainUI); n > inner {
		inner = n
	}
	if n := runes(plainWiki); n > inner {
		inner = n
	}
	total := inner + 6 // │ + two spaces of padding on each side

	ui := plainUI
	if color {
		ui = "UI:   " + "\x1b[1;32m" + u + "\x1b[0m"
	}

	// row pads display to the full box width using plain's rune length.
	row := func(plain, display string) string {
		return "│  " + display + strings.Repeat(" ", inner-runes(plain)) + "  │"
	}

	bar := strings.Repeat("─", total-2)
	return fmt.Sprintf("┌%s┐\n%s\n%s\n%s\n%s\n└%s┘\n",
		bar,
		row(title, title),
		row("", ""),
		row(plainUI, ui),
		row(plainWiki, plainWiki),
		bar,
	)
}
