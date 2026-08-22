package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// openAPISpecPath is the canonical spec file inside the embedded docs dir.
// Paths are relative to the docs sub-filesystem returned by docsSub, so the
// "docs/" prefix is already stripped.
const openAPISpecPath = "openapi.json"

// docsSub is the embedded docs dir exposed as a filesystem. The spec lives
// beside the viewer and its vendored swagger-ui assets, so one embed serves
// everything the API reference page needs.
func docsSub() (fs.FS, error) {
	return fs.Sub(docsFS, "docs")
}

// registerAPIDocs wires the API reference page onto the dev server. It is
// called only when ServeAPIDocs is true — a dev convenience that never exists
// in normal serve.
func registerAPIDocs(e *echo.Echo, d Deps, sub fs.FS) {
	// The OpenAPI document is machine-readable, always JSON.
	e.GET("/swagger.json", func(c echo.Context) error {
		spec, err := fs.ReadFile(sub, openAPISpecPath)
		if err != nil {
			return internalError(c, d, "read openapi spec", err)
		}
		return c.Blob(http.StatusOK, "application/json", spec)
	})
	// The viewer and its assets; unknown paths under /api/docs are 404s.
	e.GET("/api/docs", func(c echo.Context) error {
		index, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			return internalError(c, d, "read docs index", err)
		}
		return c.Blob(http.StatusOK, "text/html; charset=utf-8", index)
	})
	e.GET("/api/docs/swagger-ui/*", func(c echo.Context) error {
		p := strings.TrimPrefix(c.Param("*"), "/")
		if p == "" || strings.Contains(p, "..") {
			return echo.ErrNotFound
		}
		asset, err := fs.ReadFile(sub, "swagger-ui/"+p)
		if err != nil {
			return echo.ErrNotFound
		}
		ctype := contentTypeByExt(p)
		return c.Blob(http.StatusOK, ctype, asset)
	})
}

func contentTypeByExt(name string) string {
	switch {
	case strings.HasSuffix(name, ".js"):
		return "application/javascript"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
