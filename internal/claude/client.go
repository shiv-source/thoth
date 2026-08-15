package claude

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// Client starts a Claude turn for a conversation and streams parsed events.
type Client interface {
	Start(ctx context.Context, sessionID, prompt string, w EventWriter) error
}

type CLIClient struct {
	Bin            string
	Dir            string // cwd for the CLI process — the wiki path
	PermissionMode string
	Model          string

	dirProvider func() string // dynamic cwd (wiki path changes at runtime)

	mu      sync.Mutex  // guards process for the cancel path
	process *os.Process // windows: the running CLI, for cancels
}

type Option func(*CLIClient)

func WithPermissionMode(m string) Option { return func(c *CLIClient) { c.PermissionMode = m } }
func WithModel(m string) Option          { return func(c *CLIClient) { c.Model = m } }
func WithDirProvider(p func() string) Option {
	return func(c *CLIClient) { c.dirProvider = p }
}

// dir returns the cwd for the CLI process: the live provider value when set,
// otherwise the static Dir.
func (c *CLIClient) dir() string {
	if c.dirProvider != nil {
		return c.dirProvider()
	}
	return c.Dir
}

func New(bin, dir string, opts ...Option) *CLIClient {
	c := &CLIClient{Bin: bin, Dir: dir}
	for _, o := range opts {
		o(c)
	}
	return c
}

// args builds the CLI argument list. ALL flag knowledge lives here.
// Verified against `claude --help` v2.1.232/v2.1.233: with --print,
// --output-format stream-json requires --verbose (without it the CLI exits 1
// before streaming). The event parser tolerates the extra events --verbose
// emits. Permissions: with no configured mode the CLI runs fully unattended
// (--dangerously-skip-permissions — headless mode cannot answer prompts, and
// note-saving is the app's core feature); a configured permission_mode
// switches to that named mode.
func (c *CLIClient) args(sessionID, prompt string) []string {
	a := []string{"-p", "--output-format", "stream-json", "--verbose", "--session-id", sessionID}
	if c.PermissionMode != "" {
		a = append(a, "--permission-mode", c.PermissionMode)
	} else {
		a = append(a, "--dangerously-skip-permissions")
	}
	if c.Model != "" {
		a = append(a, "--model", c.Model)
	}
	return append(a, prompt)
}

// stderrTail keeps the last max bytes of the CLI's stderr for error reports.
// Write is only called by os/exec's internal copy goroutine; Cmd.Wait joins
// that goroutine before returning, so reading after Wait is race-free without
// a lock. Write always returns len(p): a short write would surface as a copy
// error in Wait.
type stderrTail struct {
	max     int
	dropped bool
	buf     []byte
}

func (t *stderrTail) Write(p []byte) (int, error) {
	if len(p) >= t.max { // single write overflows the cap: keep its tail
		t.buf = append(t.buf[:0], p[len(p)-t.max:]...)
		t.dropped = true
		return len(p), nil
	}
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max { // accumulated overflow: drop the head
		t.buf = append(t.buf[:0], t.buf[len(t.buf)-t.max:]...)
		t.dropped = true
	}
	return len(p), nil
}

func (t *stderrTail) String() string {
	if t.dropped {
		return "[stderr truncated, showing tail] " + string(t.buf)
	}
	return string(t.buf)
}

func (c *CLIClient) Start(ctx context.Context, sessionID, prompt string, w EventWriter) error {
	cmd := exec.CommandContext(ctx, c.Bin, c.args(sessionID, prompt)...)
	cmd.Dir = c.dir()
	// Put the CLI in its own process group and kill the whole group on cancel:
	// the CLI may spawn children that inherit stdout, and the stream reader
	// below only sees EOF once every writer closes. Killing just the direct
	// child can leave a grandchild holding the pipe open (seen with sh forking
	// `sleep` on macOS), hanging Start until the child exits on its own.
	// Windows has no killable process groups, so cancels kill the direct child.
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		c.mu.Lock()
		var pgid int
		if c.process != nil {
			pgid = c.process.Pid
		}
		c.mu.Unlock()
		if pgid != 0 {
			return c.killProcessGroup(pgid)
		}
		return cmd.Process.Kill()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("claude stdout pipe: %w", err)
	}
	// Capture stderr so CLI failures explain themselves instead of surfacing
	// as a bare "claude exited: exit status 1" (which previously went only to
	// the server terminal).
	tail := &stderrTail{max: 4 << 10}
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}
	c.mu.Lock()
	c.process = cmd.Process
	c.mu.Unlock()

	streamDone := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for sc.Scan() {
			ev, err := ParseLine(sc.Bytes())
			if errors.Is(err, ErrIgnore) {
				continue
			}
			if err != nil {
				_ = w.Write(Event{Type: EventError, Detail: err.Error()})
				continue
			}
			if err := w.Write(ev); err != nil {
				streamDone <- err
				return
			}
		}
		streamDone <- sc.Err()
	}()

	streamErr := <-streamDone
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err() // cancelled: that is not a failure of the CLI
	}
	if waitErr != nil {
		if s := tail.String(); s != "" {
			return fmt.Errorf("claude exited: %w (stderr: %s)", waitErr, s)
		}
		return fmt.Errorf("claude exited: %w", waitErr)
	}
	if streamErr != nil {
		return fmt.Errorf("claude stream: %w", streamErr)
	}
	return nil
}
