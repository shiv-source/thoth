package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// HealthReport is one health check: the check's name, whether it passed, and
// a human-readable message explaining the outcome or how to fix a failure.
type HealthReport struct {
	Name    string
	OK      bool
	Message string
}

// HealthFunc runs the host's health checks and returns every report. A host
// injects its concrete suite (Thoth wraps doctor.Run); the system_health tool
// renders whatever checks return, so the agent can self-diagnose.
type HealthFunc func(ctx context.Context) ([]HealthReport, error)

// SystemHealth is the "system_health" tool: it runs the host-injected health
// checks and reports the state of the installation — wiki, index, provider,
// database and server — so the agent can diagnose setup problems itself.
// Read-only.
type SystemHealth struct {
	health HealthFunc
}

// NewSystemHealth returns the system_health tool backed by fn.
func NewSystemHealth(fn HealthFunc) *SystemHealth { return &SystemHealth{health: fn} }

// Name implements Tool.
func (t *SystemHealth) Name() string { return "system_health" }

// Description implements Tool.
func (t *SystemHealth) Description() string {
	return "Run the host health checks and report the state of the wiki, search index, provider, database and server, so you can diagnose setup problems. Read-only."
}

// Schema implements Tool.
func (t *SystemHealth) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Run implements Tool.
func (t *SystemHealth) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.health == nil {
		return "", errors.New("system_health: no health checks are configured")
	}
	reports, err := t.health(ctx)
	if err != nil {
		return "", fmt.Errorf("system_health: %w", err)
	}
	var sb strings.Builder
	for _, r := range reports {
		status := "ok"
		if !r.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&sb, "%s: %s", r.Name, status)
		if r.Message != "" {
			fmt.Fprintf(&sb, " — %s", r.Message)
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}
