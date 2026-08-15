package cli

import (
	"strings"
	"testing"
)

func TestStartupBannerContainsTheFacts(t *testing.T) {
	b := startupBanner("1.2.3", "127.0.0.1", 8333, "~/.thoth/wiki", false)

	for _, want := range []string{
		"1.2.3", "ready", "Version:",
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

func TestStartupBannerShowsTheBigWordmark(t *testing.T) {
	b := startupBanner("dev", "127.0.0.1", 8333, "/tmp/wiki", false)
	// The banner is padded with blank lines so it sits clear of prior logs.
	lines := strings.Split(strings.Trim(b, "\n"), "\n")
	if len(lines) < 11 {
		t.Fatalf("expected wordmark + info box (>=11 lines), got %d:\n%s", len(lines), b)
	}
	for _, l := range lines[:5] {
		if !strings.Contains(l, "█") {
			t.Errorf("wordmark row should use block glyphs:\n%s", b)
		}
	}
}

func TestStartupBannerAddsColorOnlyWhenAsked(t *testing.T) {
	plain := startupBanner("dev", "127.0.0.1", 8333, "/tmp/wiki", false)
	colored := startupBanner("dev", "127.0.0.1", 8333, "/tmp/wiki", true)

	if strings.Contains(plain, "\x1b[") {
		t.Error("plain banner must not contain escape sequences")
	}
	if !strings.Contains(colored, "\x1b[") {
		t.Errorf("colored banner should use ANSI styling:\n%q", colored)
	}
}
