package api

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/internal/wiki"
)

// captureRequest is the POST /api/v1/capture body — the single write surface
// for the browser extension and the dashboard quick capture. Kind selects the
// save path; the remaining fields are per-kind inputs (see capture).
type captureRequest struct {
	Kind     string   `json:"kind"`
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Reason   string   `json:"reason"`
	Tags     []string `json:"tags"`
	Folder   string   `json:"folder"`
	Category string   `json:"category"`
}

// captureResponse mirrors the promotion endpoint's reply: the created file's
// wiki-relative path plus the derived title/type (advisory — the file holds
// the real frontmatter).
type captureResponse struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// capture is the unified capture endpoint: bookmark, note/selection, and
// read-later all land through the same rulebook save paths (wiki.Bookmark and
// wiki.Save), so a capture can never drift from the wiki contract. 400 on an
// unknown kind or bad input; 409 when the URL is already saved.
func capture(c echo.Context, d Deps) error {
	var req captureRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	switch req.Kind {
	case "bookmark":
		return captureBookmark(c, d, req)
	case "note", "selection":
		return captureNote(c, d, req)
	case "readlater":
		return captureReadLater(c, d, req)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": `kind must be one of "bookmark", "note", "selection", "readlater"`,
		})
	}
}

// captureBookmark appends a link line to the bookmarks master list. Bookmark
// captures are metadata-only (title/URL/reason) — full-page text is never
// captured implicitly, so one click cannot dump secrets into the wiki.
func captureBookmark(c echo.Context, d Deps, req captureRequest) error {
	url := strings.TrimSpace(req.URL)
	if url == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required for bookmarks"})
	}
	if verr := wiki.ValidSourceURL(url); verr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
	}
	if strings.ContainsAny(req.Category, "\r\n") || strings.ContainsAny(req.Reason, "\r\n") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "category and reason must be single lines"})
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = url
	}
	rel, err := d.Wiki.Bookmark(wiki.Bookmark{
		Title:    title,
		URL:      url,
		Category: strings.TrimSpace(req.Category),
		Reason:   strings.TrimSpace(req.Reason),
	})
	if err != nil {
		if errors.Is(err, wiki.ErrDuplicateBookmark) {
			return c.JSON(http.StatusConflict, map[string]any{
				"error": "url already saved",
				"path":  wiki.BookmarkFile,
			})
		}
		return internalError(c, d, "capture bookmark", err)
	}
	return c.JSON(http.StatusCreated, captureResponse{Path: rel, Title: title, Type: "bookmark"})
}

// captureNote files a note or text selection via the same save path the
// assistant's own saves use, carrying the source URL as frontmatter so the
// capture keeps provenance and dedup is possible.
func captureNote(c echo.Context, d Deps, req captureRequest) error {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "text is required for notes"})
	}
	url := strings.TrimSpace(req.URL)
	if url != "" {
		if verr := wiki.ValidSourceURL(url); verr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
		}
	}
	folder := strings.TrimSpace(req.Folder)
	if folder == "" {
		folder = defaultSaveFolder(d)
	}
	rel, err := d.Wiki.Save(wiki.SaveOptions{
		Folder:    folder,
		Title:     strings.TrimSpace(req.Title),
		Body:      req.Text,
		Tags:      req.Tags,
		SourceURL: url,
	})
	if err != nil {
		// The save's own validations (empty folder, unsafe folder, bad source
		// URL) are client errors; filesystem failures are internal.
		if verr := wiki.ValidFolder(folder); verr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
		}
		return internalError(c, d, "capture note", err)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = wiki.DefaultTitle(req.Text)
		if title == "" {
			title = filepath.Base(rel)
		}
	}
	return c.JSON(http.StatusCreated, captureResponse{Path: rel, Title: title, Type: wiki.NoteType(folder)})
}

// captureReadLater appends a line to the read-later queue
// (links/read-later.md), the same line format as the bookmarks list.
func captureReadLater(c echo.Context, d Deps, req captureRequest) error {
	url := strings.TrimSpace(req.URL)
	if url == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required for read-later"})
	}
	if verr := wiki.ValidSourceURL(url); verr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = url
	}
	rel, err := d.Wiki.AddReadLater(wiki.Bookmark{Title: title, URL: url, Reason: strings.TrimSpace(req.Reason)})
	if err != nil {
		if errors.Is(err, wiki.ErrDuplicateBookmark) {
			return c.JSON(http.StatusConflict, map[string]any{
				"error": "url already saved",
				"path":  wiki.ReadLaterFile,
			})
		}
		return internalError(c, d, "capture read-later", err)
	}
	return c.JSON(http.StatusCreated, captureResponse{Path: rel, Title: title, Type: "read-later"})
}

// captureCheck is the cheap "is this URL already saved?" check the extension
// runs before bookmarking (#2): it answers from the bookmarks master list so
// one-click bookmarking never creates a duplicate. Returns the existing file
// path so the UI can offer "already saved → open it".
func captureCheck(c echo.Context, d Deps) error {
	u := strings.TrimSpace(c.QueryParam("url"))
	if u == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}
	if verr := wiki.ValidSourceURL(u); verr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
	}
	bookmarks, err := d.Wiki.Bookmarks()
	if err != nil {
		return internalError(c, d, "capture check", err)
	}
	for _, b := range bookmarks {
		if b.URL == u {
			return c.JSON(http.StatusOK, map[string]any{"exists": true, "path": wiki.BookmarkFile})
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"exists": false})
}

// captureReadLaterList returns the read-later queue — the dashboard's triage
// surface: every queued link with its title and reason.
func captureReadLaterList(c echo.Context, d Deps) error {
	entries, err := d.Wiki.ReadLater()
	if err != nil {
		return internalError(c, d, "read-later list", err)
	}
	type item struct {
		Title  string `json:"title"`
		URL    string `json:"url"`
		Reason string `json:"reason"`
	}
	out := make([]item, 0, len(entries))
	for _, e := range entries {
		out = append(out, item{Title: e.Title, URL: e.URL, Reason: e.Reason})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out})
}

// captureReadLaterDelete removes one URL from the read-later queue. Removal is
// idempotent, so the dashboard triage can clear an item without racing.
func captureReadLaterDelete(c echo.Context, d Deps) error {
	u := strings.TrimSpace(c.QueryParam("url"))
	if u == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}
	if verr := wiki.ValidSourceURL(u); verr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
	}
	if err := d.Wiki.RemoveReadLater(u); err != nil {
		return internalError(c, d, "read-later remove", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

// captureInboxCount is the toolbar-badge source: how many captures are sitting
// unfiled in the inbox folder. It reads the index (the derived, always-synced
// store) so the badge is one cheap query, never a filesystem walk.
func captureInboxCount(c echo.Context, d Deps) error {
	count, err := d.Index.CountByPrefix("inbox")
	if err != nil {
		return internalError(c, d, "inbox count", err)
	}
	return c.JSON(http.StatusOK, map[string]int{"count": count})
}

// summarizeRequest is the POST /api/v1/capture/summarize body: the page to
// condense. Text is the page content (or selection); the assistant writes the
// summary as a knowledge note carrying the source URL in frontmatter.
type summarizeRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// summarizeCapture runs one assistant turn over the captured page text and
// promotes the resulting summary to a knowledge/ note with source: in
// frontmatter — the extension's "Ask Thoth to summarize this page" (#5). It
// composes the same two primitives as every capture: the agent and the wiki
// save path.
func summarizeCapture(c echo.Context, d Deps) error {
	var req summarizeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	url := strings.TrimSpace(req.URL)
	if url != "" {
		if verr := wiki.ValidSourceURL(url); verr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
		}
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "text is required"})
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Captured page"
	}

	prompt := fmt.Sprintf(
		"Summarize the following web page into a concise markdown note (under 300 words). "+
			"Answer with the summary text only — do not use any tools.\n\n"+
			"Title: %s\nSource: %s\n\nPage content:\n%s",
		title, url, text)
	var sb strings.Builder
	w := agentlib.WriterFunc(func(e agentlib.Event) error {
		if e.Type == agentlib.EventDelta {
			sb.WriteString(e.Text)
		}
		return nil
	})
	if _, err := d.Claude.Start(d.ctx(), "capture-summary", prompt, w); err != nil {
		return internalError(c, d, "summarize", err)
	}
	summary := strings.TrimSpace(sb.String())
	if summary == "" {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "no summary produced"})
	}
	noteTitle := "Summary: " + title
	rel, err := d.Wiki.Save(wiki.SaveOptions{
		Folder:    "knowledge",
		Title:     noteTitle,
		Body:      summary,
		SourceURL: url,
	})
	if err != nil {
		return internalError(c, d, "summarize save", err)
	}
	return c.JSON(http.StatusCreated, captureResponse{Path: rel, Title: noteTitle, Type: "knowledge"})
}
