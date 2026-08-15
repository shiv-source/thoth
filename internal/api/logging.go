package api

import (
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// requestLog logs one Info line per API request with method, path, status,
// and duration — the evidence source for latency work. SPA assets and the
// /ws chat socket are skipped so the log stays readable. A nil log (tests,
// embedders) disables it.
func requestLog(log *slog.Logger) echo.MiddlewareFunc {
	if log == nil {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := c.Request().URL.Path
			if p != "/api" && !strings.HasPrefix(p, "/api/") {
				return next(c)
			}
			start := time.Now()
			err := next(c)
			// err is attached only on failure so clean requests log without it.
			attrs := []any{"method", c.Request().Method, "path", p, "status", c.Response().Status, "dur", time.Since(start)}
			if err != nil {
				attrs = append(attrs, "err", err)
			}
			log.Info("request", attrs...)
			return err
		}
	}
}
