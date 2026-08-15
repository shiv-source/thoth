package api

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/doctor"
)

// doctorTimeout bounds the whole check suite; the claude probes inside already
// carry their own 2s timeouts, so this is a backstop for a wedged exec.
const doctorTimeout = 10 * time.Second

// doctorHandler runs the shared installation checks against the thoth dir the
// server runs from.
func doctorHandler(c echo.Context, d Deps) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), doctorTimeout)
	defer cancel()
	// The api check probes the configured address — in-flight that is this
	// very server, so it self-checks the health payload and websocket.
	checks := doctor.Run(ctx, d.DataDir, "", d.Log)
	return c.JSON(http.StatusOK, map[string]any{"checks": checks})
}
