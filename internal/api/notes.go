package api

import (
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

func search(c echo.Context, d Deps) error {
	q := c.QueryParam("q")
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "q is required"})
	}
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}
	// Treat the query as a literal phrase: strip embedded double quotes
	// (which would end the phrase early) and wrap in quotes so FTS5
	// operators (*, NOT, OR, -, …) in user input are matched literally.
	cleaned := strings.ReplaceAll(q, `"`, "")
	results, err := d.Index.Search(`"`+cleaned+`"`, limit)
	if err != nil {
		return internalError(c, d, "search", err)
	}
	if results == nil {
		results = []index.Result{} // no matches serializes as [], never null — the client types it as an array
	}
	return c.JSON(http.StatusOK, map[string]any{"results": results})
}

func note(c echo.Context, d Deps) error {
	rel := c.QueryParam("path")
	if rel == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "path is required"})
	}
	if _, err := wiki.SafePath(d.Wiki.Root(), rel); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	content, err := d.Wiki.Read(rel)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "note not found"})
	}
	if wiki.IsMarkdownPath(rel) {
		return c.JSON(http.StatusOK, map[string]string{"path": rel, "content": string(content)})
	}
	// Attachments (images, scripts, configs, …) are served as raw bytes so
	// search results are usable from the dashboard. The note-vs-attachment
	// boundary is still wiki.IsMarkdownPath, shared with the tree and the
	// index. Images render inline (an <img> can point at this URL);
	// everything else is a download with its basename.
	ctype := mime.TypeByExtension(filepath.Ext(rel))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	if !wiki.IsImagePath(rel) {
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+strings.ReplaceAll(filepath.Base(rel), `"`, ``)+`"`)
	}
	return c.Blob(http.StatusOK, ctype, content)
}

// saveNoteRequest is the POST /api/v1/notes body: the markdown content to
// promote plus an optional target folder (the UI's "Save as note"). The
// folder defaults to the first configured folder; the title defaults to the
// content's first heading/line; the type derives from the folder per the
// rulebook.
type saveNoteRequest struct {
	Content string `json:"content"`
	Folder  string `json:"folder"`
}

// createNote promotes content into a permanent wiki note: it files the
// content via the same save path the assistant's own saves use (wiki.Save —
// frontmatter, folder-appropriate type, kebab filename), so promotion cannot
// drift from the rulebook. Returns the created note's wiki-relative path and
// the derived title/type. 400 for empty content or an unknown/unsafe folder.
func createNote(c echo.Context, d Deps) error {
	var req saveNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if strings.TrimSpace(req.Content) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}
	folder := strings.TrimSpace(req.Folder)
	if folder == "" {
		folder = defaultSaveFolder(d)
	}
	rel, err := d.Wiki.Save(wiki.SaveOptions{Folder: folder, Body: req.Content})
	if err != nil {
		// The save's own validations (empty folder, unsafe folder) are
		// client errors; filesystem failures are internal. Distinguish by
		// re-running the folder gate, which is the only client-facing check
		// beyond content.
		if verr := wiki.ValidFolder(folder); verr != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": verr.Error()})
		}
		return internalError(c, d, "save note", err)
	}
	// Title mirrors the save's derivation; the type is the folder's rulebook
	// type. Both are advisory for the toast/UI, never authoritative for the
	// file (FormatNote wrote the real frontmatter).
	title := wiki.DefaultTitle(req.Content)
	if title == "" {
		title = filepath.Base(rel)
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"path":  rel,
		"title": title,
		"type":  wiki.NoteType(folder),
	})
}

// defaultSaveFolder returns the first configured folder, falling back to the
// wiki's default set; the attachments dir is never a save target.
func defaultSaveFolder(d Deps) string {
	folders, err := d.Settings.Folders()
	if err != nil || len(folders) == 0 {
		folders = wiki.Folders()
	}
	for _, f := range folders {
		if f == wiki.AttachmentsDir {
			continue
		}
		return f
	}
	return "inbox"
}

func tree(c echo.Context, d Deps) error {
	nodes, err := d.Wiki.Tree()
	if err != nil {
		return internalError(c, d, "tree", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"nodes": nodes})
}

// internalError logs err and replies with a generic body so implementation
// details never leak to clients.
func internalError(c echo.Context, d Deps, op string, err error) error {
	if d.Log != nil {
		d.Log.Error(op, "err", err)
	}
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
