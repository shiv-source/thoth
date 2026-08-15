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
	for _, want := range []string{"-p", "--output-format", "stream-json", "--verbose", "--session-id", "sess-1", "--permission-mode", "acceptEdits", "--model", "claude-opus-5", "what is in my wiki?"} {
		if !strings.Contains(s, want) {
			t.Fatalf("argv %q missing %q", s, want)
		}
	}
}

func TestStartDefaultsToDangerouslySkipPermissions(t *testing.T) {
	// No configured permission mode: the CLI must run fully unattended so
	// headless note-saving works. A named mode replaces the flag.
	bin := writeFakeCLI(t)
	c := New(bin, t.TempDir())

	if err := c.Start(context.Background(), "sess-1", "p", WriterFunc(func(Event) error { return nil })); err != nil {
		t.Fatalf("Start: %v", err)
	}
	argv, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(argv)
	if !strings.Contains(s, "--dangerously-skip-permissions") {
		t.Fatalf("argv %q missing --dangerously-skip-permissions", s)
	}
	if strings.Contains(s, "--permission-mode") {
		t.Fatalf("argv %q must not carry --permission-mode without a configured mode", s)
	}
}

func TestDirProviderOverridesStaticDir(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	// The fake CLI emits its working directory as the assistant text, making
	// the actual cwd observable.
	script := `#!/bin/sh
printf '{"type":"assistant","message":{"content":[{"type":"text","text":"'$PWD'"}]}}'
printf '\n{"type":"result","subtype":"success","is_error":false,"result":"done"}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	staticDir := t.TempDir()
	newDir := t.TempDir()
	c := New(bin, staticDir, WithDirProvider(func() string { return newDir }))

	var got []Event
	if err := c.Start(context.Background(), "s", "p", WriterFunc(func(e Event) error {
		got = append(got, e)
		return nil
	})); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(got) == 0 || got[0].Type != EventDelta {
		t.Fatalf("no delta event: %+v", got)
	}
	if got[0].Text != newDir {
		t.Fatalf("CLI ran in %q, want provider dir %q", got[0].Text, newDir)
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

func TestFakeClientPropagatesError(t *testing.T) {
	f := &FakeClient{Err: errors.New("boom")}
	err := f.Start(context.Background(), "sess", "prompt", WriterFunc(func(Event) error { return nil }))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("call must be recorded even on error, got %+v", f.Calls)
	}
}

func TestFakeClientPropagatesWriterError(t *testing.T) {
	f := &FakeClient{Script: []Event{{Type: EventDone}}}
	err := f.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return errors.New("writer broke") }))
	if err == nil || err.Error() != "writer broke" {
		t.Fatalf("expected writer error, got %v", err)
	}
}

// TestStartEmitsErrorEventForGarbageLine: a malformed stream line must be
// converted into an EventError and the stream must keep going.
func TestStartEmitsErrorEventForGarbageLine(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	script := `#!/bin/sh
echo 'not json at all'
echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c := New(bin, dir)
	var got []Event
	err := c.Start(context.Background(), "s", "p", WriterFunc(func(e Event) error {
		got = append(got, e)
		return nil
	}))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(got) != 2 || got[0].Type != EventError || got[1].Type != EventDone {
		t.Fatalf("unexpected events: %+v", got)
	}
}

func TestStartPropagatesWriterError(t *testing.T) {
	bin := writeFakeCLI(t)
	c := New(bin, t.TempDir())
	err := c.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return errors.New("writer broke") }))
	if err == nil || !strings.Contains(err.Error(), "claude stream") {
		t.Fatalf("expected claude stream error, got %v", err)
	}
}

func TestStartReportsNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := New(bin, dir)
	err := c.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "claude exited") {
		t.Fatalf("expected claude exited error, got %v", err)
	}
}

func TestStartReportsStderrOnFailure(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	script := "#!/bin/sh\necho 'stream-json requires --verbose' >&2\nexit 1\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c := New(bin, dir)
	err := c.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "claude exited") {
		t.Fatalf("expected claude exited error, got %v", err)
	}
	if !strings.Contains(err.Error(), "stream-json requires --verbose") {
		t.Fatalf("error must carry the CLI stderr, got %v", err)
	}
}

func TestStartStderrTailTruncated(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	// One write larger than the cap (keeps only its tail) followed by many
	// small writes (accumulated overflow): the error must show only the tail.
	script := `#!/bin/sh
echo 'HEAD-` + strings.Repeat("X", 5000) + `-TAIL' >&2
i=0
while [ $i -lt 300 ]; do
  echo "line-$i-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
  i=$((i+1))
done >&2
exit 1
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c := New(bin, dir)
	err := c.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "claude exited") {
		t.Fatalf("expected claude exited error, got %v", err)
	}
	for _, want := range []string{"truncated", "line-299-"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error must contain %q (tail of stderr), got %v", want, err)
		}
	}
	for _, drop := range []string{"HEAD-", "line-0-"} {
		if strings.Contains(err.Error(), drop) {
			t.Fatalf("error must not contain dropped stderr head %q, got %v", drop, err)
		}
	}
}

func TestStartOmitsSessionIDWhenEmpty(t *testing.T) {
	bin := writeFakeCLI(t)
	c := New(bin, t.TempDir())
	if err := c.Start(context.Background(), "", "p", WriterFunc(func(Event) error { return nil })); err != nil {
		t.Fatalf("Start: %v", err)
	}
	argv, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(argv), "--session-id") {
		t.Fatalf("argv %q must omit --session-id for an empty session", argv)
	}
}

func TestStartWithResumeForksSession(t *testing.T) {
	bin := writeFakeCLI(t)
	c := New(bin, t.TempDir())
	err := c.Start(context.Background(), "new-id", "p", WriterFunc(func(Event) error { return nil }), WithResume("old-id"))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	argv, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(argv)
	for _, want := range []string{"--session-id", "new-id", "--resume", "old-id", "--fork-session"} {
		if !strings.Contains(s, want) {
			t.Fatalf("argv %q missing %q", s, want)
		}
	}
}

func TestFakeClientRecordsResume(t *testing.T) {
	f := &FakeClient{Script: []Event{{Type: EventDone}}}
	err := f.Start(context.Background(), "s", "p", WriterFunc(func(Event) error { return nil }), WithResume("old"))
	if err != nil || len(f.Calls) != 1 || f.Calls[0].SessionID != "s" || f.Calls[0].Prompt != "p" || f.Calls[0].Resume != "old" {
		t.Fatalf("fake client misuse: %v %+v", err, f.Calls)
	}
}
