package openai_test

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
	"github.com/shiv-source/thoth/agent/provider/openai"
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

func streamTurn(t *testing.T, c *openai.Client) []agent.Delta {
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
			usage: agent.Usage{InputTokens: 10, OutputTokens: 4},
		},
		{
			name:    "reasoning",
			fixture: "reasoning.json",
			want: []agent.Delta{
				agent.ThinkingDelta("Let me think"),
				agent.ThinkingDelta(" deeper."),
				agent.TextDelta("The answer is 4."),
				agent.StopDelta("end_turn"),
			},
			usage: agent.Usage{InputTokens: 8, OutputTokens: 7},
		},
		{
			name:    "tool_calls",
			fixture: "tool_calls.json",
			want: []agent.Delta{
				agent.TextDelta("I'll read that file and search."),
				agent.ToolInputDelta("call_01", "Read", `{"path":"a.md"`),
				agent.ToolInputDelta("call_01", "Read", `{"path":"a.md","lines":1}`),
				agent.ToolInputDelta("call_02", "Search", `{"query":"a.md"`),
				agent.ToolInputDelta("call_02", "Search", `{"query":"a.md"}`),
				agent.StopDelta("tool_use"),
			},
			usage: agent.Usage{InputTokens: 12, OutputTokens: 9},
		},
		{
			name:    "length",
			fixture: "length.json",
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
			c := openai.New("test-key", openai.WithBaseURL(srv.URL), openai.WithModel("gpt-test"))
			got := streamTurn(t, c)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("deltas: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestStreamAccumulatesMultiToolUseMessage(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "tool_calls.json")})
	defer srv.Close()
	c := openai.New("test-key", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	resp, err := agent.Accumulate(s)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	want := agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
		agent.NewTextBlock("I'll read that file and search."),
		agent.NewToolUseBlock("call_01", "Read", map[string]any{"path": "a.md", "lines": json.Number("1")}),
		agent.NewToolUseBlock("call_02", "Search", map[string]any{"query": "a.md"}),
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
	c := openai.New("test-key", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	_, err = s.Next()
	if err == nil {
		t.Fatal("expected an error from the error event")
	}
	if !strings.Contains(err.Error(), "rate_limit_error") || !strings.Contains(err.Error(), "rate limit hit") {
		t.Fatalf("got %v", err)
	}
}

func TestStreamHTTPErrors(t *testing.T) {
	tests := []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusUnauthorized, `{"error":{"message":"bad key","type":"authentication_error"}}`, "401"},
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
			c := openai.New("test-key", openai.WithBaseURL(srv.URL))
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
	c := openai.New("test-key", openai.WithBaseURL("http://127.0.0.1:1"))
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
	c := openai.New("secret-key", openai.WithBaseURL(srv.URL))
	req := agent.Request{
		System: "sys prompt",
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("hi")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolUseBlock("call_1", "Read", map[string]any{"path": "x.md"})}},
			{Role: agent.RoleTool, Content: []agent.Block{agent.NewToolResultBlock("call_1", "file contents", false)}},
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
	if got := h.req.URL.Path; got != "/v1/chat/completions" {
		t.Fatalf("path %q, want /v1/chat/completions", got)
	}
	if got := h.req.Header.Get("Authorization"); got != "Bearer secret-key" {
		t.Fatalf("authorization %q", got)
	}
	if got := h.req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(h.reqBody, &body); err != nil {
		t.Fatalf("request body: %v", err)
	}
	if body["model"] != "gpt-5.6-mini" {
		t.Fatalf("model %v", body["model"])
	}
	if body["max_tokens"] != float64(2048) {
		t.Fatalf("max_tokens %v", body["max_tokens"])
	}
	if body["stream"] != true {
		t.Fatalf("stream %v", body["stream"])
	}
	if body["tool_choice"] != "auto" {
		t.Fatalf("tool_choice %v", body["tool_choice"])
	}

	msgs := body["messages"].([]any)
	if len(msgs) != 4 {
		t.Fatalf("messages %+v", body["messages"])
	}
	m0 := msgs[0].(map[string]any)
	if m0["role"] != "system" || m0["content"] != "sys prompt" {
		t.Fatalf("message 0 %+v", m0)
	}
	m1 := msgs[1].(map[string]any)
	part := m1["content"].([]any)[0].(map[string]any)
	if m1["role"] != "user" || part["type"] != "text" || part["text"] != "hi" {
		t.Fatalf("message 1 %+v", m1)
	}
	m2 := msgs[2].(map[string]any)
	tc := m2["tool_calls"].([]any)[0].(map[string]any)
	fn := tc["function"].(map[string]any)
	if m2["role"] != "assistant" || tc["id"] != "call_1" || fn["name"] != "Read" || fn["arguments"] != `{"path":"x.md"}` {
		t.Fatalf("message 2 %+v", m2)
	}
	m3 := msgs[3].(map[string]any)
	if m3["role"] != "tool" || m3["tool_call_id"] != "call_1" || m3["content"] != "file contents" {
		t.Fatalf("message 3 %+v", m3)
	}

	tl := body["tools"].([]any)[0].(map[string]any)
	if tl["type"] != "function" {
		t.Fatalf("tools %+v", body["tools"])
	}
	tfn := tl["function"].(map[string]any)
	if tfn["name"] != "Read" || tfn["description"] != "reads" || tfn["parameters"].(map[string]any)["type"] != "object" {
		t.Fatalf("tool function %+v", tfn)
	}
}

func TestStreamDefaultsMaxTokens(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := openai.New("test-key", openai.WithBaseURL(srv.URL), openai.WithMaxTokens(2048))
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
	fixture := "data: {\"id\":\"c\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":\"call_9\",\"type\":\"function\",\"function\":{\"name\":\"Noop\",\"arguments\":\"\"}}]}}]}\n\ndata: {\"id\":\"c\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(fixture)})
	defer srv.Close()
	c := openai.New("test-key", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	resp, err := agent.Accumulate(s)
	if err != nil {
		t.Fatalf("Accumulate: %v", err)
	}
	want := agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
		agent.NewToolUseBlock("call_9", "Noop", map[string]any{}),
	}}
	if !reflect.DeepEqual(resp.Message, want) {
		t.Fatalf("got %+v, want %+v", resp.Message, want)
	}
}

func TestStreamToolArgumentsBeforeToolStart(t *testing.T) {
	fixture := "data: {\"id\":\"c\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"a\\\":1}\"}}]}}]}\n\ndata: [DONE]\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(fixture)})
	defer srv.Close()
	c := openai.New("test-key", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	_, err = s.Next()
	if err == nil {
		t.Fatal("expected an error for arguments before tool id/name")
	}
	if !strings.Contains(err.Error(), "arguments before id/name") {
		t.Fatalf("got %v", err)
	}
}
