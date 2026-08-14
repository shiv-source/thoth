package api

import (
	"net/http"
	"os/exec"

	"github.com/labstack/echo/v4"
)

type healthResponse struct {
	Status string      `json:"status"`
	Claude claudeState `json:"claude"`
	Wiki   wikiState   `json:"wiki"`
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
	// Read under the read lock: putSettings replaces the whole config struct.
	d.ConfigMu.RLock()
	bin := d.Config.ClaudeBin
	d.ConfigMu.RUnlock()
	if bin == "" {
		bin = "claude"
	}
	found := false
	if p, err := exec.LookPath(bin); err == nil {
		found = true
		bin = p
	}
	return c.JSON(http.StatusOK, healthResponse{
		Status: "ok",
		Claude: claudeState{Found: found, Path: bin},
		Wiki:   wikiState{Path: d.Wiki.Root, Exists: d.Wiki.Exists()},
	})
}
