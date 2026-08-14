package api

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/doctor"
)

// doctorTimeout bounds the whole check suite; the claude probes inside already
// carry their own 2s timeouts, so this is a backstop for a wedged exec.
const doctorTimeout = 10 * time.Second

// doctorHandler runs the shared installation checks against the thoth dir the
// server runs from (derived from the config path, like the CLI's default).
func doctorHandler(c echo.Context, d Deps) error {
	dir := ""
	if d.ConfigPath != "" {
		dir = filepath.Dir(d.ConfigPath)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), doctorTimeout)
	defer cancel()
	// The port check is pre-launch only: this handler runs inside the very
	// process occupying the port, so it would always report a false positive.
	checks := doctor.Run(ctx, dir, d.Log, doctor.SkipPort())
	return c.JSON(http.StatusOK, map[string]any{"checks": checks})
}
