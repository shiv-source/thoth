package api

import (
	"net/http"
	"os/exec"

	"github.com/labstack/echo/v4"
)

type healthResponse struct {
	Status          string      `json:"status"`
	Claude          claudeState `json:"claude"`
	Wiki            wikiState   `json:"wiki"`
	Version         string      `json:"version"`
	Dev             bool        `json:"dev"`
	Commit          string      `json:"commit"`
	DefaultWikiPath string      `json:"default_wiki_path"`
}

type claudeState struct {
	Found bool   `json:"found"`
	Path  string `json:"path"`
}

type wikiState struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func health(c echo.Context, d Deps) error {
	// The claude binary always comes from PATH (config.toml is gone).
	bin := "claude"
	found := false
	if p, err := exec.LookPath(bin); err == nil {
		found = true
		bin = p
	}
	return c.JSON(http.StatusOK, healthResponse{
		Status:          "ok",
		Claude:          claudeState{Found: found, Path: bin},
		Wiki:            wikiState{Path: d.Wiki.Root, Exists: d.Wiki.Exists()},
		Version:         d.Version,
		Dev:             d.Dev,
		Commit:          d.Commit,
		DefaultWikiPath: d.DefaultWikiPath,
	})
}
