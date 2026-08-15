package cli

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// glyphs is the block-letter font for the wordmark: five rows per letter.
var glyphs = map[rune][5]string{
	't': {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
	'h': {"█   █", "█   █", "█████", "█   █", "█   █"},
	'o': {"█████", "█   █", "█   █", "█   █", "█████"},
}

// isTerminal reports whether f is a character device — a real terminal —
// so the banner only uses ANSI styling when it will actually be seen.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// startupBanner renders the startup panel: a big block-letter "thoth"
// wordmark over a boxed panel with the version, UI URL, and wiki path.
// With color=true the wordmark gets an emerald gradient and the URL is
// bold emerald (callers pass true only when stderr is a TTY).
func startupBanner(version, host string, port int, wikiPath string, color bool) string {
	// lipgloss picks its color profile from the environment; force ANSI so
	// the styled banner renders regardless of TERM/NO_COLOR, and let the
	// plain path (no color styles) stay escape-free.
	if color {
		prev := lipgloss.ColorProfile()
		lipgloss.SetColorProfile(termenv.ANSI)
		defer lipgloss.SetColorProfile(prev)
	}

	u := (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, strconv.Itoa(port))}).String() + "/"

	shades := []string{"#059669", "#0a9e6e", "#10b981", "#34d399", "#6ee7b7"}
	var rows [5]string
	for i := range rows {
		var b strings.Builder
		for j, r := range "thoth" {
			if j > 0 {
				b.WriteString("  ")
			}
			g := glyphs[r][i]
			if color {
				g = lipgloss.NewStyle().Foreground(lipgloss.Color(shades[j])).Render(g)
			}
			b.WriteString(g)
		}
		rows[i] = b.String()
	}
	wordmark := strings.Join(rows[:], "\n")
	if color {
		wordmark = lipgloss.NewStyle().Bold(true).Render(wordmark)
	}

	label := func(s string) string {
		if color {
			return lipgloss.NewStyle().Faint(true).Render(s)
		}
		return s
	}
	ui := u
	if color {
		ui = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34d399")).Render(u)
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		"Thoth — ready",
		"",
		label("UI:      ")+ui,
		label("Wiki:    ")+wikiPath,
		label("Version: ")+version,
	)

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	if color {
		box = box.BorderForeground(lipgloss.Color("240"))
	}

	// Blank lines on both sides keep the panel clear of surrounding output.
	return "\n\n" + lipgloss.JoinVertical(lipgloss.Center, wordmark, box.Render(content)) + "\n\n"
}
