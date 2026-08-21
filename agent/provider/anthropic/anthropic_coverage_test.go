package anthropic_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/provider/anthropic"
)

func TestWithHTTPClientApplies(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	client := &http.Client{}
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL), anthropic.WithHTTPClient(client))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()
}

func TestStreamRequestBuildErrors(t *testing.T) {
	c := anthropic.New("k", anthropic.WithBaseURL("http://127.0.0.1:1"))
	// Unknown role fails wire mapping before any HTTP is attempted.
	if _, err := c.Stream(context.Background(), agent.Request{
		Messages: []agent.Message{{Role: "admin", Content: []agent.Block{agent.NewTextBlock("x")}}},
	}); err == nil {
		t.Fatal("expected error for unknown role")
	}
	// Unknown block type fails wire mapping too.
	if _, err := c.Stream(context.Background(), agent.Request{
		Messages: []agent.Message{{Role: agent.RoleUser, Content: []agent.Block{{Type: "bogus"}}}},
	}); err == nil {
		t.Fatal("expected error for unknown block type")
	}
}

func TestStreamHTTPTransportError(t *testing.T) {
	// A server that closes before the request: the transport returns an error
	// that is not a cancellation, surfacing the wrapped request error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(url))
	if _, err := c.Stream(context.Background(), turnRequest()); err == nil {
		t.Fatal("expected a transport error from a closed server")
	}
}

func TestStreamCacheMarkerOnToolUseBlock(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	req := agent.Request{
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q1")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolUseBlock("t1", "Read", map[string]any{"path": "a.md"})}},
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q2")}},
		},
	}
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()
	var body map[string]any
	if err := json.Unmarshal(srv.Config.Handler.(*fixtureHandler).reqBody, &body); err != nil {
		t.Fatal(err)
	}
	msgs := body["messages"].([]any)
	blk := msgs[1].(map[string]any)["content"].([]any)[0].(map[string]any)
	if cc, ok := blk["cache_control"].(map[string]any); !ok || cc["type"] != "ephemeral" {
		t.Fatalf("tool_use at the cache boundary must carry cache_control: %+v", msgs[1])
	}
}

func TestStreamCacheMarkerOnToolResultBlock(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	req := agent.Request{
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q1")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolUseBlock("t1", "Read", map[string]any{"path": "a.md"})}},
			{Role: agent.RoleTool, Content: []agent.Block{agent.NewToolResultBlock("t1", "ok", false)}},
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q2")}},
		},
	}
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()
	var body map[string]any
	if err := json.Unmarshal(srv.Config.Handler.(*fixtureHandler).reqBody, &body); err != nil {
		t.Fatal(err)
	}
	msgs := body["messages"].([]any)
	blk := msgs[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	if cc, ok := blk["cache_control"].(map[string]any); !ok || cc["type"] != "ephemeral" {
		t.Fatalf("tool_result at the cache boundary must carry cache_control: %+v", msgs[2])
	}
}

func TestStreamUnknownEventsAndTextBlocks(t *testing.T) {
	body := "event: ping\ndata: {}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"signature_delta\",\"signature\":\"x\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	d, err := s.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if d.Kind != agent.DeltaStop || d.StopReason != "end_turn" {
		t.Fatalf("delta = %+v, want stop end_turn", d)
	}
}

func TestStreamInputJSONBeforeToolUseStart(t *testing.T) {
	body := "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{}\"}}\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.Next(); err == nil || !strings.Contains(err.Error(), "before tool_use start") {
		t.Fatalf("err = %v, want an input_json_delta-before-start error", err)
	}
}

func TestStreamWireBlockVariants(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	req := agent.Request{
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q1")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{
				agent.NewThinkingBlock("reason"),
				agent.NewToolUseBlock("t1", "Read", nil), // nil input → {} on the wire
			}},
			{Role: agent.RoleTool, Content: []agent.Block{agent.NewToolResultBlock("t1", "ok", true)}},
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q2")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{}}, // empty content at the cache marker
			{Role: agent.RoleUser, Content: []agent.Block{agent.NewTextBlock("q3")}},
		},
	}
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()
	var body map[string]any
	if err := json.Unmarshal(srv.Config.Handler.(*fixtureHandler).reqBody, &body); err != nil {
		t.Fatal(err)
	}
	msgs := body["messages"].([]any)
	m1 := msgs[1].(map[string]any)
	blocks := m1["content"].([]any)
	if blocks[0].(map[string]any)["type"] != "thinking" {
		t.Fatalf("message 1 block 0 = %+v, want a thinking block", blocks[0])
	}
	tu := blocks[1].(map[string]any)
	if tu["type"] != "tool_use" {
		t.Fatalf("message 1 tool_use = %+v, want a tool_use block", tu)
	}
	if input, ok := tu["input"].(map[string]any); !ok || len(input) != 0 {
		t.Fatalf("message 1 tool_use input = %+v, want {} (nil normalized)", tu["input"])
	}
}

func TestStreamDecodeErrors(t *testing.T) {
	// Valid JSON with a type-mismatched field fails the wire struct decode.
	for _, body := range []string{
		"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":\"many\"}}}\n\n",
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":1}}\n\n",
	} {
		srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
		c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
		s, err := c.Stream(context.Background(), turnRequest())
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if _, err := s.Next(); err == nil {
			t.Fatalf("Next succeeded, want a decode error for %q", body)
		}
		_ = s.Close()
		srv.Close()
	}
}

func TestStreamMessageStopWithoutReason(t *testing.T) {
	body := "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	d, err := s.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if d.Kind != agent.DeltaStop || d.StopReason != "end_turn" {
		t.Fatalf("delta = %+v, want a default end_turn stop", d)
	}
}

func TestStreamSSEFrameError(t *testing.T) {
	body := "event: e\ndata: not-json\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := anthropic.New("k", anthropic.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.Next(); err == nil || !strings.Contains(err.Error(), "anthropic: sse") {
		t.Fatalf("err = %v, want an anthropic: sse error", err)
	}
}
