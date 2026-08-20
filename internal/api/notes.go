package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
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
	return c.JSON(http.StatusOK, map[string]any{"results": results})
}

func note(c echo.Context, d Deps) error {
	rel := c.QueryParam("path")
	if rel == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "path is required"})
	}
	if _, err := wiki.SafePath(d.Wiki.Root, rel); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if !wiki.IsMarkdownPath(rel) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "path is not a note"})
	}
	content, err := d.Wiki.Read(rel)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "note not found"})
	}
	return c.JSON(http.StatusOK, map[string]string{"path": rel, "content": string(content)})
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
