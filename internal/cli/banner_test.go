package cli

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStartupBannerContainsTheFacts(t *testing.T) {
	b := startupBanner("1.2.3", "127.0.0.1", 8333, "~/.thoth/wiki", false)

	for _, want := range []string{
		"Thoth", "1.2.3", "ready",
		"http://127.0.0.1:8333/",
		"~/.thoth/wiki",
	} {
		if !strings.Contains(b, want) {
			t.Errorf("banner missing %q:\n%s", want, b)
		}
	}
}

func TestStartupBannerFormatsIPv6Hosts(t *testing.T) {
	b := startupBanner("dev", "::1", 8333, "/tmp/wiki", false)
	if !strings.Contains(b, "http://[::1]:8333/") {
		t.Errorf("banner should bracket IPv6 hosts:\n%s", b)
	}
}

func TestStartupBannerDrawsAnEvenBox(t *testing.T) {
	b := startupBanner("dev", "127.0.0.1", 8333, "~/.thoth/wiki", false)
	lines := strings.Split(strings.TrimSuffix(b, "\n"), "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 box rows (top, title, blank, ui, wiki, bottom), got %d:\n%s", len(lines), b)
	}
	for i, l := range lines {
		if utf8.RuneCountInString(l) != utf8.RuneCountInString(lines[0]) {
			t.Errorf("row %d width %d != row 0 width %d:\n%s", i, utf8.RuneCountInString(l), utf8.RuneCountInString(lines[0]), b)
		}
	}
	if !strings.HasPrefix(lines[0], "┌") || !strings.HasPrefix(lines[5], "└") {
		t.Errorf("box should be closed with corner rows:\n%s", b)
	}
}

func TestStartupBannerAddsColorOnlyWhenAsked(t *testing.T) {
	plain := startupBanner("dev", "127.0.0.1", 8333, "/tmp/wiki", false)
	colored := startupBanner("dev", "127.0.0.1", 8333, "/tmp/wiki", true)

	if strings.Contains(plain, "\x1b[") {
		t.Error("plain banner must not contain escape sequences")
	}
	if !strings.Contains(colored, "\x1b[1;32m") {
		t.Errorf("colored banner should highlight the URL:\n%q", colored)
	}
}
