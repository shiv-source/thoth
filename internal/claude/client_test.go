package claude

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeFakeCLI writes an executable script that prints canned stream-json
// lines and appends its argv to argv.txt so tests can assert the exact flags.
func writeFakeCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	script := `#!/bin/sh
echo "$@" >> "$(dirname "$0")/argv.txt"
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"hi from cli"}]}}'
echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestStartStreamsEventsAndPassesFlags(t *testing.T) {
	bin := writeFakeCLI(t)
	c := New(bin, t.TempDir(), WithPermissionMode("acceptEdits"), WithModel("claude-opus-5"))

	var got []Event
	err := c.Start(context.Background(), "sess-1", "what is in my wiki?", WriterFunc(func(e Event) error {
		got = append(got, e)
		return nil
	}))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(got) != 2 || got[0].Type != EventDelta || got[0].Text != "hi from cli" || got[1].Type != EventDone {
		t.Fatalf("unexpected events: %+v", got)
	}

	argv, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(argv)
	for _, want := range []string{"-p", "--output-format", "stream-json", "--session-id", "sess-1", "--permission-mode", "acceptEdits", "--model", "claude-opus-5", "what is in my wiki?"} {
		if !strings.Contains(s, want) {
			t.Fatalf("argv %q missing %q", s, want)
		}
	}
}

func TestStartCancelKillsProcess(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := New(bin, dir)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx, "sess", "slow", WriterFunc(func(Event) error { return nil })) }()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after cancel")
	}
}

func TestStartMissingBinary(t *testing.T) {
	c := New(filepath.Join(t.TempDir(), "nope"), t.TempDir())
	if err := c.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return nil })); err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestFakeClient(t *testing.T) {
	f := &FakeClient{Script: []Event{{Type: EventDone}}}
	err := f.Start(context.Background(), "sess", "prompt", WriterFunc(func(Event) error { return nil }))
	if err != nil || len(f.Calls) != 1 || f.Calls[0].SessionID != "sess" || f.Calls[0].Prompt != "prompt" {
		t.Fatalf("fake client misuse: %v %+v", err, f.Calls)
	}
}
