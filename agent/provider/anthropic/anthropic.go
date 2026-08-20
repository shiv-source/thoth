// Package anthropic is a Provider over the Anthropic Messages API (POST
// /v1/messages with stream:true). It is the first concrete provider and proves
// the Provider seam: it imports only the public agent API and the shared SSE
// transport — no internal packages, no third-party SDKs.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/transport"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultModel     = "claude-sonnet-5"
	defaultMaxTokens = 4096
	apiVersion       = "2023-06-01"
	messagesPath     = "/v1/messages"
)

// Option configures a Client.
type Option func(*options)

type options struct {
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// WithBaseURL overrides the API base URL (default https://api.anthropic.com).
func WithBaseURL(u string) Option {
	return func(o *options) { o.baseURL = u }
}

// WithModel sets the model id (default claude-sonnet-5).
func WithModel(m string) Option {
	return func(o *options) { o.model = m }
}

// WithMaxTokens sets the max_tokens bound used when the Request leaves it zero
// (default 4096).
func WithMaxTokens(n int) Option {
	return func(o *options) { o.maxTokens = n }
}

// WithHTTPClient overrides the HTTP client used for requests (default
// http.DefaultClient).
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// Client is an Anthropic provider. Create one with New; it is safe for
// concurrent use.
type Client struct {
	apiKey    string
	baseURL   string
	model     string
	maxTokens int
	http      *http.Client
}

// New returns an Anthropic client for apiKey. Configure it with the With*
// options.
func New(apiKey string, opts ...Option) *Client {
	o := options{
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		maxTokens:  defaultMaxTokens,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{apiKey: apiKey, baseURL: o.baseURL, model: o.model, maxTokens: o.maxTokens, http: o.httpClient}
}

var _ agent.Provider = (*Client)(nil)

// Stream starts a turn and returns a stream of normalized deltas. The caller
// must Close the stream; cancelling ctx aborts the underlying request.
func (c *Client) Stream(ctx context.Context, req agent.Request) (agent.Stream, error) {
	wire, err := buildRequest(c, req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("anthropic: encode request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+messagesPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("anthropic: request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic: api returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return newStream(resp.Body), nil
}

// wireRequest is the Anthropic Messages API request body.
type wireRequest struct {
	Model     string        `json:"model"`
	System    []wireSystem  `json:"system,omitempty"`
	Messages  []wireMessage `json:"messages"`
	Tools     []wireTool    `json:"tools,omitempty"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
}

type wireSystem struct {
	Type         string            `json:"type"`
	Text         string            `json:"text"`
	CacheControl *wireCacheControl `json:"cache_control,omitempty"`
}

type wireCacheControl struct {
	Type string `json:"type"`
}

type wireMessage struct {
	Role    string `json:"role"`
	Content []any  `json:"content"`
}

type wireTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type wireTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type wireThinkingBlock struct {
	Type     string `json:"type"`
	Thinking string `json:"thinking"`
}

type wireToolUseBlock struct {
	Type  string         `json:"type"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

type wireToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// buildRequest maps a normalized Request onto the Anthropic wire shape. The
// system prompt becomes a single stable text block marked for prompt caching.
func buildRequest(c *Client, req agent.Request) (wireRequest, error) {
	messages, err := wireMessages(req.Messages)
	if err != nil {
		return wireRequest{}, err
	}
	w := wireRequest{
		Model:     c.model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
		Stream:    true,
	}
	if w.MaxTokens == 0 {
		w.MaxTokens = c.maxTokens
	}
	if req.System != "" {
		w.System = []wireSystem{{
			Type:         "text",
			Text:         req.System,
			CacheControl: &wireCacheControl{Type: "ephemeral"},
		}}
	}
	if len(req.Tools) > 0 {
		w.Tools = make([]wireTool, 0, len(req.Tools))
		for _, t := range req.Tools {
			w.Tools = append(w.Tools, wireTool{Name: t.Name, Description: t.Description, InputSchema: t.Schema})
		}
	}
	return w, nil
}

func wireMessages(msgs []agent.Message) ([]wireMessage, error) {
	out := make([]wireMessage, 0, len(msgs))
	for _, m := range msgs {
		role, err := wireRole(m.Role)
		if err != nil {
			return nil, err
		}
		content, err := wireBlocks(m.Content)
		if err != nil {
			return nil, err
		}
		out = append(out, wireMessage{Role: role, Content: content})
	}
	return out, nil
}

// wireRole maps a normalized role onto the Anthropic role. Tool results ride
// inside a user message on the Anthropic wire.
func wireRole(role string) (string, error) {
	switch role {
	case agent.RoleUser:
		return "user", nil
	case agent.RoleAssistant:
		return "assistant", nil
	case agent.RoleTool:
		return "user", nil
	default:
		return "", fmt.Errorf("anthropic: unknown role %q", role)
	}
}

func wireBlocks(blocks []agent.Block) ([]any, error) {
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		w, err := wireBlock(b)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func wireBlock(b agent.Block) (any, error) {
	switch b.Type {
	case agent.BlockText:
		return wireTextBlock{Type: "text", Text: b.Text}, nil
	case agent.BlockThinking:
		return wireThinkingBlock{Type: "thinking", Thinking: b.Text}, nil
	case agent.BlockToolUse:
		input := b.Input
		if input == nil {
			input = map[string]any{}
		}
		return wireToolUseBlock{Type: "tool_use", ID: b.ID, Name: b.ToolName, Input: input}, nil
	case agent.BlockToolResult:
		return wireToolResultBlock{Type: "tool_result", ToolUseID: b.ToolUseID, Content: b.Text, IsError: b.IsError}, nil
	default:
		return nil, fmt.Errorf("anthropic: unknown block type %q", b.Type)
	}
}

// stream decodes the Anthropic SSE response into normalized deltas. Tool_use
// input arrives as input_json_delta fragments, accumulated per content block
// index and re-emitted whole on each fragment; a tool_use with no fragments
// closes as an empty object.
type stream struct {
	frames       *transport.SSEReader
	body         io.Closer
	usage        agent.Usage
	stopReason   string
	toolInput    map[int]string
	toolID       map[int]string
	toolName     map[int]string
	inputEmitted map[int]bool
	ended        bool
}

func newStream(body io.ReadCloser) *stream {
	return &stream{
		frames:       transport.NewSSEReader(body),
		body:         body,
		toolInput:    map[int]string{},
		toolID:       map[int]string{},
		toolName:     map[int]string{},
		inputEmitted: map[int]bool{},
	}
}

// Next returns the next delta in order, then io.EOF when the turn ends.
func (s *stream) Next() (agent.Delta, error) {
	if s.ended {
		return agent.Delta{}, io.EOF
	}
	for {
		f, err := s.frames.Next()
		if err != nil {
			s.ended = true
			if errors.Is(err, io.EOF) {
				return agent.Delta{}, io.EOF
			}
			return agent.Delta{}, fmt.Errorf("anthropic: sse: %w", err)
		}
		d, stop, err := s.handle(f)
		if err != nil {
			s.ended = true
			return agent.Delta{}, err
		}
		if stop || d.Kind != "" {
			if stop {
				s.ended = true
			}
			return d, nil
		}
	}
}

// Usage is valid once Next has returned io.EOF or a stop delta.
func (s *stream) Usage() agent.Usage { return s.usage }

// Close releases the response body.
func (s *stream) Close() error { return s.body.Close() }

func (s *stream) handle(f transport.Frame) (agent.Delta, bool, error) {
	var e struct {
		Type string `json:"type"`
	}
	if err := f.Decode(&e); err != nil {
		return agent.Delta{}, false, fmt.Errorf("anthropic: %w", err)
	}
	switch e.Type {
	case "message_start":
		return agent.Delta{}, false, s.handleMessageStart(f)
	case "content_block_start":
		return agent.Delta{}, false, s.handleBlockStart(f)
	case "content_block_delta":
		d, err := s.handleBlockDelta(f)
		return d, false, err
	case "content_block_stop":
		d, err := s.handleBlockStop(f)
		return d, false, err
	case "message_delta":
		return agent.Delta{}, false, s.handleMessageDelta(f)
	case "message_stop":
		reason := s.stopReason
		if reason == "" {
			reason = "end_turn"
		}
		return agent.StopDelta(reason), true, nil
	case "error":
		return agent.Delta{}, false, s.handleError(f)
	default: // ping and anything unrecognized carry no delta
		return agent.Delta{}, false, nil
	}
}

func (s *stream) handleMessageStart(f transport.Frame) error {
	var e struct {
		Message struct {
			Usage struct {
				InputTokens          int `json:"input_tokens"`
				CacheReadInputTokens int `json:"cache_read_input_tokens"`
				CacheCreationInput   int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if err := f.Decode(&e); err != nil {
		return fmt.Errorf("anthropic: %w", err)
	}
	s.usage.InputTokens = e.Message.Usage.InputTokens
	s.usage.CacheReadTokens = e.Message.Usage.CacheReadInputTokens
	s.usage.CacheWriteTokens = e.Message.Usage.CacheCreationInput
	return nil
}

func (s *stream) handleBlockStart(f transport.Frame) error {
	var e struct {
		Index        int `json:"index"`
		ContentBlock struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"content_block"`
	}
	if err := f.Decode(&e); err != nil {
		return fmt.Errorf("anthropic: %w", err)
	}
	if e.ContentBlock.Type == "tool_use" {
		s.toolID[e.Index] = e.ContentBlock.ID
		s.toolName[e.Index] = e.ContentBlock.Name
	}
	return nil
}

func (s *stream) handleBlockDelta(f transport.Frame) (agent.Delta, error) {
	var e struct {
		Index int `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Thinking    string `json:"thinking"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	}
	if err := f.Decode(&e); err != nil {
		return agent.Delta{}, fmt.Errorf("anthropic: %w", err)
	}
	switch e.Delta.Type {
	case "text_delta":
		return agent.TextDelta(e.Delta.Text), nil
	case "thinking_delta":
		return agent.ThinkingDelta(e.Delta.Thinking), nil
	case "input_json_delta":
		id := s.toolID[e.Index]
		if id == "" {
			return agent.Delta{}, fmt.Errorf("anthropic: input_json_delta before tool_use start at index %d", e.Index)
		}
		s.toolInput[e.Index] += e.Delta.PartialJSON
		s.inputEmitted[e.Index] = true
		return agent.ToolInputDelta(id, s.toolName[e.Index], s.toolInput[e.Index]), nil
	default:
		return agent.Delta{}, nil
	}
}

func (s *stream) handleBlockStop(f transport.Frame) (agent.Delta, error) {
	var e struct {
		Index int `json:"index"`
	}
	if err := f.Decode(&e); err != nil {
		return agent.Delta{}, fmt.Errorf("anthropic: %w", err)
	}
	id := s.toolID[e.Index]
	if id == "" || s.inputEmitted[e.Index] {
		return agent.Delta{}, nil
	}
	return agent.ToolInputDelta(id, s.toolName[e.Index], "{}"), nil
}

func (s *stream) handleMessageDelta(f transport.Frame) error {
	var e struct {
		Delta struct {
			StopReason string `json:"stop_reason"`
		} `json:"delta"`
		Usage struct {
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := f.Decode(&e); err != nil {
		return fmt.Errorf("anthropic: %w", err)
	}
	s.stopReason = e.Delta.StopReason
	s.usage.OutputTokens = e.Usage.OutputTokens
	return nil
}

func (s *stream) handleError(f transport.Frame) error {
	var e struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := f.Decode(&e); err != nil {
		return fmt.Errorf("anthropic: %w", err)
	}
	return fmt.Errorf("anthropic: %s: %s", e.Error.Type, e.Error.Message)
}
