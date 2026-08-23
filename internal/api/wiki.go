package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/wiki"
)

// importFileField is the multipart field name of the uploaded zip.
const importFileField = "file"

// exportWiki streams a zip of the wiki tree (GET /api/v1/wiki/export).
// history=1 includes dotfiles (.git) so git history can travel; the default
// export is the wiki proper. The zip is streamed so large wikis do not sit in
// memory; an unreadable file mid-stream truncates the archive rather than
// failing before headers are sent.
func exportWiki(c echo.Context, d Deps) error {
	if !d.Wiki.Exists() {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "wiki not found"})
	}
	filename := "thoth-wiki-" + time.Now().Format("2006-01-02") + ".zip"
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	if err := d.Wiki.ExportTo(c.Response(), wiki.ExportOptions{
		IncludeHidden: c.QueryParam("history") == "1",
	}); err != nil {
		return internalError(c, d, "export wiki", err)
	}
	return nil
}

// importWiki validates a multipart wiki zip and merges it onto the wiki root
// (POST /api/v1/wiki/import), then rebuilds the index so search and the tree
// reflect the imported files and pushes a wiki_changed frame so connected
// clients refetch the tree. The merge itself is backup-first (see
// wiki.ImportFrom); 400 for a missing file, a bad archive, path traversal, or
// a non-wiki archive, 413 when the archive exceeds the size cap.
func importWiki(c echo.Context, d Deps) error {
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, wiki.MaxImportZipBytes)
	file, err := c.FormFile(importFileField)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{"error": "archive is too large"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}
	src, err := file.Open()
	if err != nil {
		return internalError(c, d, "open upload", err)
	}
	defer func() { _ = src.Close() }()

	result, err := d.Wiki.ImportFrom(src, file.Size, d.Log)
	if err != nil {
		if errors.Is(err, wiki.ErrArchiveTooLarge) {
			return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	// Files are the source of truth; the index is derived, so an import is
	// files on disk + a reindex. The explicit sync makes the tree visible even
	// without the watcher (tests); in serve the watcher reconciles too.
	if err := d.Index.Sync(d.Wiki.Root(), d.Log); err != nil {
		return internalError(c, d, "reindex after import", err)
	}
	d.publishWikiChanged()
	return c.JSON(http.StatusOK, map[string]any{
		"files":  result.Files,
		"backup": result.Backup,
	})
}

// publishWikiChanged pushes an empty wiki_changed batch so connected clients
// refetch the tree after a bulk change the watcher may not have published yet.
// A nil bus or a closed bus is fine — the refresh is best-effort.
func (d Deps) publishWikiChanged() {
	if d.Events == nil {
		return
	}
	if err := d.Events.Publish(d.ctx(), wiki.Changed{}); err != nil && d.Log != nil {
		d.Log.Warn("publish wiki change", "err", err)
	}
}
