package anthropic_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/provider/anthropic"
)

// fixtureHandler serves a canned SSE fixture and captures the incoming request.
type fixtureHandler struct {
	t       *testing.T
	body    []byte
	req     *http.Request
	reqBody []byte
}

func (h *fixtureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.req = r
	b, err := io.ReadAll(r.Body)
	if err != nil {
		h.t.Fatal(err)
	}
	h.reqBody = b
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write(h.body)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func turnRequest() agent.Request {
	return agent.Request{
		System: "You are a helpful assistant.",
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("Read a.md")}},
		},
		Tools:     []agent.Tool{{Name: "Read", Description: "Reads a file", Schema: map[string]any{"type": "object"}}},
		MaxTokens: 512,
	}
}

func streamTurn(t *testing.T, srv *httptest.Server, c *anthropic.Client) []agent.Delta {
	t.Helper()
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	var got []agent.Delta
	for {
		d, err := s.Next()
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		got = append(got, d)
	}
}

func TestStreamFixtures(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    []agent.Delta
		usage   agent.Usage
	}{
		{
			name:    "text",
			fixture: "text.json",
			want: []agent.Delta{
				agent.TextDelta("Hello"),
				agent.TextDelta(" world"),
				agent.StopDelta("end_turn"),
			},
			usage: agent.Usage{InputTokens: 10, OutputTokens: 4, CacheWriteTokens: 3, CacheReadTokens: 5},
		},
		{
			name:    "thinking",
			fixture: "thinking.json",
			want: []agent.Delta{
				agent.ThinkingDelta("Let me think"),
				agent.ThinkingDelta(" deeper."),
				agent.TextDelta("The answer is 4."),
				agent.StopDelta("end_turn"),
			},
			usage: agent.Usage{InputTokens: 8, OutputTokens: 7},
		},
		{
			name:    "tool_use",
			fixture: "tool_use.json",
			want: []agent.Delta{
				agent.TextDelta("I'll read that file."),
				agent.ToolInputDelta("toolu_01", "Read", `{"path":"a.md"`),
				agent.ToolInputDelta("toolu_01", "Read", `{"path":"a.md","lines":1}`),
				agent.StopDelta("tool_use"),
			},
			usage: agent.Usage{InputTokens: 12, OutputTokens: 9},
		},
		{
			name:    "max_tokens",
			fixture: "max_tokens.json",
			want: []agent.Delta{
				agent.TextDelta("Let me"),
				agent.StopDelta("max_tokens"),
			},
			usage: agent.Usage{InputTokens: 6, OutputTokens: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, tt.fixture)})
			defer srv.Close()
			c := anthropic.New("test-key", anthropic.WithBaseURL(srv.URL), anthropic.WithModel("claude-test"))
			got := streamTurn(t, srv, c)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("deltas: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestStreamAccumulatesToolUseMessage(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "tool_use.json")})
	defer srv.Close()
	c := anthropic.New("test-key", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	resp, err := agent.Accumulate(s)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	want := agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
		agent.NewTextBlock("I'll read that file."),
		agent.NewToolUseBlock("toolu_01", "Read", map[string]any{"path": "a.md", "lines": json.Number("1")}),
	}}
	if !reflect.DeepEqual(resp.Message, want) {
		t.Fatalf("got %+v, want %+v", resp.Message, want)
	}
	if resp.Usage != (agent.Usage{InputTokens: 12, OutputTokens: 9}) {
		t.Fatalf("got usage %+v", resp.Usage)
	}
}

func TestStreamErrorEvent(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "error.json")})
	defer srv.Close()
	c := anthropic.New("test-key", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	_, err = s.Next()
	if err == nil {
		t.Fatal("expected an error from the error event")
	}
	if !strings.Contains(err.Error(), "overloaded_error") || !strings.Contains(err.Error(), "try again") {
		t.Fatalf("got %v", err)
	}
}

func TestStreamHTTPErrors(t *testing.T) {
	tests := []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusUnauthorized, `{"type":"error","error":{"type":"authentication_error","message":"bad key"}}`, "401"},
		{http.StatusTooManyRequests, `{"error":{"message":"rate limited"}}`, "429"},
		{http.StatusInternalServerError, "boom", "500"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := anthropic.New("test-key", anthropic.WithBaseURL(srv.URL))
			s, err := c.Stream(context.Background(), turnRequest())
			if err == nil {
				_ = s.Close()
				t.Fatal("expected an error for non-200 response")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestStreamCancel(t *testing.T) {
	c := anthropic.New("test-key", anthropic.WithBaseURL("http://127.0.0.1:1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Stream(ctx, turnRequest())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestStreamBuildsRequest(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := anthropic.New("secret-key", anthropic.WithBaseURL(srv.URL))
	req := agent.Request{
		System: "sys prompt",
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("hi")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolUseBlock("toolu_1", "Read", map[string]any{"path": "x.md"})}},
			{Role: agent.RoleTool, Content: []agent.Block{agent.NewToolResultBlock("toolu_1", "file contents", false)}},
		},
		Tools:     []agent.Tool{{Name: "Read", Description: "reads", Schema: map[string]any{"type": "object"}}},
		MaxTokens: 2048,
	}
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()

	h := srv.Config.Handler.(*fixtureHandler)
	if h.req == nil {
		t.Fatal("no request captured")
	}
	if got := h.req.URL.Path; got != "/v1/messages" {
		t.Fatalf("path %q, want /v1/messages", got)
	}
	if got := h.req.Header.Get("x-api-key"); got != "secret-key" {
		t.Fatalf("x-api-key %q", got)
	}
	if got := h.req.Header.Get("anthropic-version"); got != "2023-06-01" {
		t.Fatalf("anthropic-version %q", got)
	}
	if got := h.req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(h.reqBody, &body); err != nil {
		t.Fatalf("request body: %v", err)
	}
	if body["model"] != "claude-sonnet-5" {
		t.Fatalf("model %v", body["model"])
	}
	if body["max_tokens"] != float64(2048) {
		t.Fatalf("max_tokens %v", body["max_tokens"])
	}
	if body["stream"] != true {
		t.Fatalf("stream %v", body["stream"])
	}

	sys := body["system"].([]any)[0].(map[string]any)
	if sys["type"] != "text" || sys["text"] != "sys prompt" {
		t.Fatalf("system %+v", sys)
	}
	if sys["cache_control"].(map[string]any)["type"] != "ephemeral" {
		t.Fatalf("system cache_control %+v", sys)
	}

	msgs := body["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("messages %+v", body["messages"])
	}
	m0 := msgs[0].(map[string]any)
	if m0["role"] != "user" || m0["content"].([]any)[0].(map[string]any)["text"] != "hi" {
		t.Fatalf("message 0 %+v", m0)
	}
	m1 := msgs[1].(map[string]any)
	blk1 := m1["content"].([]any)[0].(map[string]any)
	if m1["role"] != "assistant" || blk1["type"] != "tool_use" || blk1["id"] != "toolu_1" || blk1["name"] != "Read" {
		t.Fatalf("message 1 %+v", m1)
	}
	m2 := msgs[2].(map[string]any)
	blk2 := m2["content"].([]any)[0].(map[string]any)
	if m2["role"] != "user" || blk2["type"] != "tool_result" || blk2["tool_use_id"] != "toolu_1" || blk2["content"] != "file contents" {
		t.Fatalf("message 2 %+v", m2)
	}

	tl := body["tools"].([]any)[0].(map[string]any)
	if tl["name"] != "Read" || tl["description"] != "reads" || tl["input_schema"].(map[string]any)["type"] != "object" {
		t.Fatalf("tools %+v", body["tools"])
	}
}

func TestStreamDefaultsMaxTokens(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := anthropic.New("test-key", anthropic.WithBaseURL(srv.URL), anthropic.WithMaxTokens(2048))
	req := turnRequest()
	req.MaxTokens = 0
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()
	var body map[string]any
	if err := json.Unmarshal(srv.Config.Handler.(*fixtureHandler).reqBody, &body); err != nil {
		t.Fatalf("request body: %v", err)
	}
	if body["max_tokens"] != float64(2048) {
		t.Fatalf("max_tokens %v, want 2048", body["max_tokens"])
	}
}

func TestStreamToolUseWithEmptyInput(t *testing.T) {
	fixture := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_06\",\"usage\":{\"input_tokens\":3}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_9\",\"name\":\"Noop\",\"input\":{}}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(fixture)})
	defer srv.Close()
	c := anthropic.New("test-key", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	resp, err := agent.Accumulate(s)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	want := agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
		agent.NewToolUseBlock("toolu_9", "Noop", map[string]any{}),
	}}
	if !reflect.DeepEqual(resp.Message, want) {
		t.Fatalf("got %+v, want %+v", resp.Message, want)
	}
}
