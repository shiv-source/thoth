// Package transport holds the wire-format-free transport helpers shared by
// every provider: the SSE reader that splits a streaming HTTP response into
// one decoded JSON event per frame. It has no dependencies on the agent core.
package transport

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Frame is one decoded SSE event: the event name (the "event:" field, empty
// when omitted) and the raw JSON payload assembled from the frame's "data:"
// lines. Providers decode the payload into their wire structs with Decode.
type Frame struct {
	Event string
	Data  []byte
}

// Decode unmarshals the frame's JSON payload into v.
func (f Frame) Decode(v any) error {
	if err := json.Unmarshal(f.Data, v); err != nil {
		return fmt.Errorf("transport: sse: decode %q frame: %w", f.Event, err)
	}
	return nil
}

// SSEReader splits an SSE byte stream into JSON frames. It accepts LF and
// CRLF line endings, ignores ":" comments (keep-alives) and blank lines, and
// reassembles multi-line "data:" fields, so frames arrive whole regardless of
// write chunk boundaries. It is not safe for concurrent use.
type SSEReader struct {
	br *bufio.Reader
}

// NewSSEReader returns a reader that splits r into frames.
func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{br: bufio.NewReaderSize(r, 64*1024)}
}

// Next returns the next frame, or io.EOF when the stream ends cleanly. A
// frame cut off by EOF (data without its terminating blank line) or a data
// payload that is not valid JSON is an error.
func (r *SSEReader) Next() (Frame, error) {
	var name string
	var data []byte
	pending := false
	for {
		line, err := r.br.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if err != nil {
			if pending {
				return Frame{}, fmt.Errorf("transport: sse: stream ended mid-frame: %w", err)
			}
			if errors.Is(err, io.EOF) {
				return Frame{}, io.EOF
			}
			return Frame{}, err
		}
		switch {
		case line == "":
			if pending {
				if len(data) > 0 && !json.Valid(data) {
					return Frame{}, fmt.Errorf("transport: sse: %q frame has invalid JSON", name)
				}
				return Frame{Event: name, Data: data}, nil
			}
		case strings.HasPrefix(line, ":"):
			// comment (keep-alive); ignored
		default:
			field, value, _ := strings.Cut(line, ":")
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				name = value
				pending = true
			case "data":
				data = append(data, value...)
				data = append(data, '\n')
				pending = true
			}
		}
	}
}
