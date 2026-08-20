package transport

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func readAllFrames(t *testing.T, body string) []Frame {
	t.Helper()
	r := NewSSEReader(strings.NewReader(body))
	var frames []Frame
	for {
		f, err := r.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return frames
			}
			t.Fatalf("Next: %v", err)
		}
		frames = append(frames, f)
	}
}

func TestSSEReaderFrames(t *testing.T) {
	body := "event: message_start\r\n" +
		"data: {\"type\":\"message_start\"}\r\n" +
		"\r\n" +
		": keep-alive ping\r\n" +
		"\r\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\"}}\n" +
		"\n"
	want := []Frame{
		{Event: "message_start", Data: []byte("{\"type\":\"message_start\"}\n")},
		{Event: "content_block_delta", Data: []byte("{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\"}}\n")},
	}
	frames := readAllFrames(t, body)
	if !reflect.DeepEqual(frames, want) {
		t.Fatalf("got %+v, want %+v", frames, want)
	}
}

func TestSSEReaderCommentsIgnored(t *testing.T) {
	if frames := readAllFrames(t, ": keep-alive\n: keep-alive\n\n"); len(frames) != 0 {
		t.Fatalf("expected no frames, got %+v", frames)
	}
}

func TestSSEReaderBlankLinesIgnored(t *testing.T) {
	frames := readAllFrames(t, "\n\n\nevent: e\ndata: {}\n\n\n\n")
	if len(frames) != 1 || frames[0].Event != "e" {
		t.Fatalf("got %+v", frames)
	}
}

func TestSSEReaderMultilineData(t *testing.T) {
	body := "event: e\n" +
		"data: {\"a\":\n" +
		"data: 1}\n" +
		"\n"
	frames := readAllFrames(t, body)
	if len(frames) != 1 || string(frames[0].Data) != "{\"a\":\n1}\n" {
		t.Fatalf("got %+v", frames)
	}
}

func TestSSEReaderChunkBoundaries(t *testing.T) {
	body := "event: a\ndata: {\"x\":1}\n\nevent: b\ndata: {\"y\":2}\n\n"
	want := []Frame{
		{Event: "a", Data: []byte("{\"x\":1}\n")},
		{Event: "b", Data: []byte("{\"y\":2}\n")},
	}
	for _, size := range []int{1, 2, 3, 7, 17, len(body)} {
		r := NewSSEReader(&chunkReader{data: []byte(body), size: size})
		for i, w := range want {
			got, err := r.Next()
			if err != nil {
				t.Fatalf("chunk %d frame %d: %v", size, i, err)
			}
			if !reflect.DeepEqual(got, w) {
				t.Fatalf("chunk %d frame %d: got %+v, want %+v", size, i, got, w)
			}
		}
		if _, err := r.Next(); !errors.Is(err, io.EOF) {
			t.Fatalf("chunk %d: want io.EOF, got %v", size, err)
		}
	}
}

func TestSSEReaderMalformedJSON(t *testing.T) {
	r := NewSSEReader(strings.NewReader("event: e\ndata: not-json\n\n"))
	if _, err := r.Next(); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestSSEReaderDONETerminator(t *testing.T) {
	for _, body := range []string{
		"data: {\"id\":\"x\"}\n\ndata: [DONE]\n\n",
		"data: {\"id\":\"x\"}\n\ndata: [DONE]\n", // no trailing blank line
	} {
		r := NewSSEReader(strings.NewReader(body))
		f, err := r.Next()
		if err != nil {
			t.Fatalf("frame before [DONE]: %v", err)
		}
		if string(f.Data) != "{\"id\":\"x\"}\n" {
			t.Fatalf("got %q", f.Data)
		}
		if _, err := r.Next(); !errors.Is(err, io.EOF) {
			t.Fatalf("want io.EOF after [DONE], got %v", err)
		}
	}
}

func TestSSEReaderEOFMidFrame(t *testing.T) {
	for _, body := range []string{"data: {\"partial\":", "event: e\ndata: {\"partial\":"} {
		r := NewSSEReader(strings.NewReader(body))
		_, err := r.Next()
		if err == nil {
			t.Fatalf("expected error for truncated frame %q", body)
		}
		if errors.Is(err, io.EOF) {
			t.Fatalf("truncated frame %q must not read as io.EOF: %v", body, err)
		}
	}
}

func TestSSEReaderUnderlyingError(t *testing.T) {
	wantErr := errors.New("boom")
	r := NewSSEReader(errReader{err: wantErr})
	if _, err := r.Next(); !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}

func TestSSEReaderHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\"}\n\n: keep-alive\n\nevent: content_block_delta\ndata: {\"type\":\"text_delta\"}\n\n")
	}))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	r := NewSSEReader(resp.Body)
	n := 0
	for {
		_, err := r.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("Next: %v", err)
		}
		n++
	}
	if n != 2 {
		t.Fatalf("got %d frames, want 2", n)
	}
}

func TestFrameDecode(t *testing.T) {
	f := Frame{Event: "message_delta", Data: []byte("{\"stop_reason\":\"end_turn\"}\n")}
	var m struct {
		StopReason string `json:"stop_reason"`
	}
	if err := f.Decode(&m); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if m.StopReason != "end_turn" {
		t.Fatalf("got %q, want end_turn", m.StopReason)
	}
	if err := (Frame{Event: "e", Data: []byte("nope")}).Decode(&m); err == nil {
		t.Fatal("expected decode error for malformed JSON")
	}
}

// chunkReader yields at most size bytes per Read call.
type chunkReader struct {
	data []byte
	size int
	off  int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if c.off >= len(c.data) {
		return 0, io.EOF
	}
	n := len(p)
	if n > c.size {
		n = c.size
	}
	if rem := len(c.data) - c.off; n > rem {
		n = rem
	}
	copy(p[:n], c.data[c.off:c.off+n])
	c.off += n
	return n, nil
}

// errReader returns err from every Read.
type errReader struct {
	err error
}

func (e errReader) Read(p []byte) (int, error) { return 0, e.err }
