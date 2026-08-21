package openai_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/provider/openai"
)

func TestWithHTTPClientApplies(t *testing.T) {
	srv := httptest.NewServer(&fixtureHandler{t: t, body: readFixture(t, "text.json")})
	defer srv.Close()
	c := openai.New("k", openai.WithBaseURL(srv.URL), openai.WithHTTPClient(&http.Client{}))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_ = s.Close()
}

func TestStreamRequestBuildErrors(t *testing.T) {
	c := openai.New("k", openai.WithBaseURL("http://127.0.0.1:1"))
	cases := []agent.Request{
		// Unknown role fails wire mapping before any HTTP is attempted.
		{Messages: []agent.Message{{Role: "admin", Content: []agent.Block{agent.NewTextBlock("x")}}}},
		// Unknown block type.
		{Messages: []agent.Message{{Role: agent.RoleUser, Content: []agent.Block{{Type: "bogus"}}}}},
		// tool_use is only valid in assistant messages.
		{Messages: []agent.Message{{Role: agent.RoleUser, Content: []agent.Block{agent.NewToolUseBlock("t1", "Read", nil)}}}},
		// tool_result rides a dedicated tool message, not a user/assistant one.
		{Messages: []agent.Message{{Role: agent.RoleAssistant, Content: []agent.Block{agent.NewToolResultBlock("t1", "ok", false)}}}},
	}
	for _, req := range cases {
		if _, err := c.Stream(context.Background(), req); err == nil {
			t.Fatalf("expected an error for request %+v", req)
		}
	}
}

func TestStreamHTTPTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	c := openai.New("k", openai.WithBaseURL(url))
	if _, err := c.Stream(context.Background(), turnRequest()); err == nil {
		t.Fatal("expected a transport error from a closed server")
	}
}

func TestStreamPlainEOFWithoutFinishReason(t *testing.T) {
	body := "data: {\"id\":\"x\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := openai.New("k", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	d, err := s.Next()
	if err != nil || d.Kind != agent.DeltaText || d.Text != "hi" {
		t.Fatalf("first delta = %+v, %v", d, err)
	}
	if _, err := s.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("stream must end with io.EOF, got %v", err)
	}
}

func TestStreamSSEFrameError(t *testing.T) {
	body := "data: not-json\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := openai.New("k", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.Next(); err == nil || !strings.Contains(err.Error(), "openai: sse") {
		t.Fatalf("err = %v, want an openai: sse error", err)
	}
}

func TestStreamHandleDecodeError(t *testing.T) {
	body := "data: {\"choices\":\"not-a-list\"}\n\n"
	srv := httptest.NewServer(&fixtureHandler{t: t, body: []byte(body)})
	defer srv.Close()
	c := openai.New("k", openai.WithBaseURL(srv.URL))
	s, err := c.Stream(context.Background(), turnRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.Next(); err == nil {
		t.Fatal("Next succeeded, want a chunk decode error")
	}
}
