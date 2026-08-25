package wiki

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	agenttools "github.com/shiv-source/thoth/agent/tools"
)

const (
	// BookmarkFile is the wiki-relative path of the bookmarks master list
	// (the rulebook's "grouped by category, one line per link" file).
	BookmarkFile = "links/bookmarks.md"
	// ReadLaterFile is the wiki-relative path of the read-later queue.
	ReadLaterFile = "links/read-later.md"
	// DefaultCategory is the bookmarks group a bookmark lands in when the
	// caller supplies no category.
	DefaultCategory = "unfiled"
)

// ErrDuplicateBookmark is returned by Bookmark/AddReadLater when the link's
// URL is already saved, so callers can show the "already saved → open it"
// state instead of writing a second line into the master list.
var ErrDuplicateBookmark = errors.New("bookmark: URL already saved")

// Bookmark describes one link to append to the bookmarks master list or the
// read-later queue. URL must be an absolute http(s) URL; Category and Reason
// are optional (Category defaults to DefaultCategory) and must be single
// lines so neither can inject extra structure into the link file.
type Bookmark struct {
	Title    string
	URL      string
	Category string
	Reason   string
}

// LinkEntry is one parsed line of a link file: the markdown link plus the
// category section it sits under ("" for a flat list like the read-later
// queue).
type LinkEntry struct {
	Title    string
	URL      string
	Category string
	Reason   string
}

// linkLineRe matches a bookmarks line "- [title](url)…". The title may hold
// backslash-escaped brackets (write-side escapeLinkText); the URL is either a
// percent-encoded run (write-side encodeLinkURL) or the hand-edited <...>
// angle-bracket form, so a URL containing ")" (Wikipedia titles, …) still
// parses back verbatim. The capture after the URL is the optional " — reason"
// suffix.
var linkLineRe = regexp.MustCompile(`^-\s+\[((?:[^\]\\]|\\.)*)\]\((<[^>]*>|[^)]+)\)(.*)$`)

// linkURLEncoder percent-encodes the characters that would break markdown link
// parsing — "%" first so a pre-existing %XX sequence round-trips, then "(", ")"
// and spaces. The stored line stays mostly readable; decodeLinkURL reverses it.
var linkURLEncoder = strings.NewReplacer(`%`, `%25`, `(`, `%28`, `)`, `%29`, ` `, `%20`)

// encodeLinkURL escapes a URL for embedding in a markdown link line. Callers
// must decode before comparing (decodeLinkURL) so a ")" in the path does not
// truncate the round-trip.
func encodeLinkURL(u string) string {
	return linkURLEncoder.Replace(u)
}

// decodeLinkURL restores a URL embedded by encodeLinkURL. The hand-edited
// <...> angle-bracket form (raw, unencoded) is returned verbatim.
func decodeLinkURL(u string) string {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(u, "<") && strings.HasSuffix(u, ">") && len(u) >= 2 {
		return u[1 : len(u)-1]
	}
	if d, err := url.PathUnescape(u); err == nil {
		return d
	}
	return u
}

// escapeLinkText backslash-escapes the characters that would terminate or
// mis-parse a markdown link's text ([, ], \); unescapeLinkText reverses it.
func escapeLinkText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `[`, `\[`)
	s = strings.ReplaceAll(s, `]`, `\]`)
	return s
}

// unescapeLinkText reverses escapeLinkText: every backslash-prefixed character
// is taken literally.
func unescapeLinkText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// NormalizeURL canonicalizes a URL for dedup comparisons: lowercased scheme and
// host, default ports stripped, and the fragment dropped. The path and query
// are kept verbatim — http vs https, www vs bare, and "?utm=…" remain distinct
// resources — so normalization only collapses the accidental duplicates (case,
// "https://EXAMPLE.com:443/a#x" vs "https://example.com/a"). An unparseable
// input is returned unchanged.
func NormalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" {
		return u
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if port := parsed.Port(); port != "" {
		if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
			parsed.Host = host
		} else {
			parsed.Host = net.JoinHostPort(host, port)
		}
	} else {
		parsed.Host = host
	}
	parsed.Fragment = ""
	return parsed.String()
}

// Bookmark appends one line to the bookmarks master list (links/bookmarks.md),
// grouped under its category heading in the rulebook format:
//
//	## <category>
//	- [<title>](<url>) — <reason>
//
// The file (with note frontmatter, so bookmarks are valid notes) is created
// when missing, and the write is atomic (temp file + rename). A URL that is
// already saved returns ErrDuplicateBookmark without writing. Returns the
// wiki-relative path of the master list.
func (w *Wiki) Bookmark(b Bookmark) (string, error) {
	if err := validateLink(b); err != nil {
		return "", err
	}
	if err := w.appendLink(BookmarkFile, "Bookmarks", b, true); err != nil {
		return "", err
	}
	return BookmarkFile, nil
}

// AddReadLater appends one line to the read-later queue (links/read-later.md),
// the same line format as the bookmarks list but flat (no category grouping).
// Returns the wiki-relative path of the queue.
func (w *Wiki) AddReadLater(b Bookmark) (string, error) {
	if err := validateLink(b); err != nil {
		return "", err
	}
	if err := w.appendLink(ReadLaterFile, "Read Later", b, false); err != nil {
		return "", err
	}
	return ReadLaterFile, nil
}

// Bookmarks parses the bookmarks master list into its entries, used for the
// cheap "is this URL already saved?" dedup check and for listing. A missing
// file is an empty list, never an error.
func (w *Wiki) Bookmarks() ([]LinkEntry, error) {
	return w.readLinks(BookmarkFile)
}

// ReadLater parses the read-later queue into its entries. A missing file is
// an empty list, never an error.
func (w *Wiki) ReadLater() ([]LinkEntry, error) {
	return w.readLinks(ReadLaterFile)
}

// RemoveReadLater removes every read-later entry whose URL matches url,
// rewriting the queue atomically. It is idempotent: a queue that never
// existed (or no longer holds the URL) is a no-op, so the dashboard triage
// can remove an item without racing its own refresh.
func (w *Wiki) RemoveReadLater(url string) error {
	w.linksMu.Lock()
	defer w.linksMu.Unlock()
	content, err := w.Read(ReadLaterFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := splitLines(content)
	out := lines[:0]
	want := NormalizeURL(url)
	for _, raw := range lines {
		if m := linkLineRe.FindStringSubmatch(strings.TrimSpace(raw)); m != nil && NormalizeURL(decodeLinkURL(m[2])) == want {
			continue
		}
		out = append(out, raw)
	}
	if len(out) == len(lines) {
		return nil // nothing removed — nothing to rewrite
	}
	full, err := SafePath(w.Root(), ReadLaterFile)
	if err != nil {
		return err
	}
	if err := agenttools.AtomicWrite(full, []byte(joinLines(out)), 0o644); err != nil {
		return fmt.Errorf("bookmark: remove from %s: %w", ReadLaterFile, err)
	}
	return nil
}

// ValidSourceURL reports whether u is an acceptable capture source URL: an
// absolute http or https URL, or empty (no source). Source provenance rides
// in note frontmatter so a capture keeps a link back to where it came from.
func ValidSourceURL(u string) error {
	trimmed := strings.TrimSpace(u)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("source URL %q is not a valid URL", u)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("source URL scheme %q must be http or https", parsed.Scheme)
	}
	return nil
}

// validateLink validates the fields of a link entry: a non-empty single-line
// title, an absolute http(s) URL, and single-line category/reason.
func validateLink(b Bookmark) error {
	if strings.TrimSpace(b.Title) == "" {
		return errors.New("bookmark: title is required")
	}
	if strings.ContainsAny(b.Title, "\r\n") || strings.ContainsAny(b.Category, "\r\n") || strings.ContainsAny(b.Reason, "\r\n") {
		return errors.New("bookmark: title, category and reason must be single lines")
	}
	if strings.TrimSpace(b.URL) == "" {
		return errors.New("bookmark: url is required")
	}
	return ValidSourceURL(b.URL)
}

// appendLink appends b to the link file at rel. When grouped is true the
// entry lands under its category heading (the bookmarks list); otherwise it
// is appended flat (the read-later queue). The write is atomic and
// SafePath-bounded; the file is recreated when missing or empty.
func (w *Wiki) appendLink(rel, title string, b Bookmark, grouped bool) error {
	w.linksMu.Lock()
	defer w.linksMu.Unlock()
	full, err := SafePath(w.Root(), rel)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(full)
	missing := errors.Is(err, os.ErrNotExist)
	if err != nil && !missing {
		return fmt.Errorf("bookmark: read %s: %w", rel, err)
	}
	lines := splitLines(content)
	if missing || len(strings.TrimSpace(string(content))) == 0 {
		lines = freshLinkFile(title)
	}
	if urlSaved(lines, b.URL) {
		return ErrDuplicateBookmark
	}
	line := bookmarkLine(b)
	if grouped {
		lines = insertGrouped(lines, b.Category, line)
	} else {
		lines = append(lines, line)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("bookmark: create dir: %w", err)
	}
	if err := agenttools.AtomicWrite(full, []byte(joinLines(lines)), 0o644); err != nil {
		return fmt.Errorf("bookmark: write %s: %w", rel, err)
	}
	return nil
}

// freshLinkFile returns the starting lines of a link file: note frontmatter
// (so the master list is a valid, searchable note) plus the H1.
func freshLinkFile(title string) []string {
	return []string{
		"---",
		"title: " + title,
		"type: link",
		"---",
		"",
		"# " + title,
		"",
	}
}

// bookmarkLine renders one link line in the rulebook format:
// "- [title](url) — reason". The title and URL are escaped so any valid
// input round-trips through linkLineRe/readLinks.
func bookmarkLine(b Bookmark) string {
	line := "- [" + escapeLinkText(strings.TrimSpace(b.Title)) + "](" + encodeLinkURL(strings.TrimSpace(b.URL)) + ")"
	if reason := strings.TrimSpace(b.Reason); reason != "" {
		line += " — " + reason
	}
	return line
}

// urlSaved reports whether a link line in lines already carries url, compared
// after URL normalization so accidental duplicates (case, default port,
// fragment) are caught too.
func urlSaved(lines []string, url string) bool {
	want := NormalizeURL(url)
	for _, raw := range lines {
		m := linkLineRe.FindStringSubmatch(strings.TrimSpace(raw))
		if m != nil && NormalizeURL(decodeLinkURL(m[2])) == want {
			return true
		}
	}
	return false
}

// insertGrouped inserts line under the section headed by category, creating
// the section when it does not exist yet. The bookmarks list groups entries
// under "## category" headings.
func insertGrouped(lines []string, category, line string) []string {
	cat := strings.TrimSpace(category)
	if cat == "" {
		cat = DefaultCategory
	}
	heading := "## " + cat
	for i, l := range lines {
		if l != heading {
			continue
		}
		end := i + 1
		for end < len(lines) && !strings.HasPrefix(lines[end], "## ") {
			end++
		}
		insertAt := end
		for insertAt > i+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
			insertAt--
		}
		out := make([]string, 0, len(lines)+1)
		out = append(out, lines[:insertAt]...)
		out = append(out, line)
		return append(out, lines[insertAt:]...)
	}
	// New section: a blank line (unless the file already ends on one), the
	// heading, then the entry.
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	return append(lines, heading, line)
}

// readLinks parses a link file into ordered entries, tracking the current
// "## category" section. Lines that match neither a heading nor a link line
// (hand edits, the H1, frontmatter) are ignored.
func (w *Wiki) readLinks(rel string) ([]LinkEntry, error) {
	content, err := w.Read(rel)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []LinkEntry
	category := ""
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			category = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		m := linkLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		reason := strings.TrimSpace(m[3])
		reason = strings.TrimSpace(strings.TrimPrefix(reason, "—"))
		out = append(out, LinkEntry{
			Title:    unescapeLinkText(strings.TrimSpace(m[1])),
			URL:      decodeLinkURL(m[2]),
			Category: category,
			Reason:   reason,
		})
	}
	return out, nil
}

// splitLines splits content into its lines, dropping the trailing empty
// element a final newline creates. Empty content yields nil.
func splitLines(content []byte) []string {
	s := strings.TrimRight(string(content), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// joinLines joins lines back into content with a guaranteed trailing newline.
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
