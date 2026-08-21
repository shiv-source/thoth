package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSystemHealth(t *testing.T) {
	fail := errors.New("boom")
	tests := []struct {
		name    string
		health  HealthFunc
		wantOut string
		wantErr string
	}{
		{
			name: "all ok",
			health: func(ctx context.Context) ([]HealthReport, error) {
				return []HealthReport{{Name: "wiki", OK: true, Message: "wiki exists"}}, nil
			},
			wantOut: "wiki: ok — wiki exists",
		},
		{
			name: "mixed with failures",
			health: func(ctx context.Context) ([]HealthReport, error) {
				return []HealthReport{
					{Name: "wiki", OK: true, Message: "wiki exists"},
					{Name: "index", OK: false, Message: "index is stale"},
				}, nil
			},
			wantOut: "wiki: ok — wiki exists\nindex: FAIL — index is stale",
		},
		{
			name: "empty message renders bare status",
			health: func(ctx context.Context) ([]HealthReport, error) {
				return []HealthReport{{Name: "database", OK: true}}, nil
			},
			wantOut: "database: ok",
		},
		{
			name: "health func error propagates",
			health: func(ctx context.Context) ([]HealthReport, error) {
				return nil, fail
			},
			wantErr: "boom",
		},
		{
			name:    "nil health func is a clean error",
			health:  nil,
			wantErr: "no health checks are configured",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewSystemHealth(tt.health)
			if tool.Name() != "system_health" || tool.Description() == "" {
				t.Fatalf("system_health missing name or description")
			}
			if schema := tool.Schema(); schema["type"] != "object" {
				t.Fatalf("system_health schema type = %v, want object", schema["type"])
			} else if _, present := schema["required"]; present {
				t.Fatalf("system_health schema should not require arguments")
			}
			out, err := tool.Run(context.Background(), map[string]any{})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Run error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out != tt.wantOut {
				t.Fatalf("Run out = %q, want %q", out, tt.wantOut)
			}
		})
	}
}

func TestSystemHealthChecksContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tool := NewSystemHealth(func(ctx context.Context) ([]HealthReport, error) {
		t.Fatal("health func must not run on a cancelled context")
		return nil, nil
	})
	if _, err := tool.Run(ctx, map[string]any{}); err == nil {
		t.Fatal("Run on cancelled context should error")
	}
}
