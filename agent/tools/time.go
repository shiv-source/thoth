package tools

import (
	"context"
	"fmt"
	"time"
)

// GetTime is the "get_time" tool: it returns the current date and time in UTC
// and in the host's local zone, optionally in a named time zone.
type GetTime struct {
	now func() time.Time
}

// NewGetTime returns the get_time tool. now supplies the clock; nil falls back
// to time.Now (tests inject a fixed clock).
func NewGetTime(now func() time.Time) *GetTime {
	if now == nil {
		now = time.Now
	}
	return &GetTime{now: now}
}

// Name implements Tool.
func (t *GetTime) Name() string { return "get_time" }

// Description implements Tool.
func (t *GetTime) Description() string {
	return "Get the current date and time, in UTC and the local zone, optionally in a named time zone."
}

// Schema implements Tool.
func (t *GetTime) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tz": map[string]any{
				"type":        "string",
				"description": "Optional IANA time zone name (e.g. America/New_York) to report the time in.",
			},
		},
	}
}

// Run implements Tool.
func (t *GetTime) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	tz, err := StringArgDefault(args, "tz", "")
	if err != nil {
		return "", err
	}
	now := t.now()
	utc := now.UTC()
	out := fmt.Sprintf("UTC: %s\nlocal: %s", utc.Format(time.RFC3339), now.Local().Format(time.RFC3339))
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return "", fmt.Errorf("get_time: unknown time zone %q", tz)
		}
		out += fmt.Sprintf("\n%s: %s", tz, now.In(loc).Format(time.RFC3339))
	}
	return out, nil
}
