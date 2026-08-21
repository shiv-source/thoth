package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGetTimeTool(t *testing.T) {
	fixed := time.Date(2026, 8, 21, 15, 4, 5, 0, time.UTC)
	tl := NewGetTime(func() time.Time { return fixed })

	out, err := tl.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("get_time: %v", err)
	}
	if !strings.Contains(out, "UTC: 2026-08-21T15:04:05Z") {
		t.Fatalf("UTC time missing: %q", out)
	}
	if !strings.Contains(out, "local: ") {
		t.Fatalf("local time missing: %q", out)
	}

	// Named time zone renders in that zone.
	out, err = tl.Run(context.Background(), map[string]any{"tz": "America/New_York"})
	if err != nil {
		t.Fatalf("get_time with tz: %v", err)
	}
	if !strings.Contains(out, "America/New_York: 2026-08-21T11:04:05") {
		t.Fatalf("tz time missing: %q", out)
	}

	if _, err := tl.Run(context.Background(), map[string]any{"tz": "Not/AZone"}); err == nil {
		t.Fatal("unknown time zone succeeded")
	}
}

func TestGetTimeToolCtxCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewGetTime(nil).Run(ctx, map[string]any{}); err == nil {
		t.Fatal("get_time on cancelled ctx succeeded")
	}
}
