package claude

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
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
}

type Option func(*CLIClient)

func WithPermissionMode(m string) Option { return func(c *CLIClient) { c.PermissionMode = m } }
func WithModel(m string) Option          { return func(c *CLIClient) { c.Model = m } }

func New(bin, dir string, opts ...Option) *CLIClient {
	c := &CLIClient{Bin: bin, Dir: dir}
	for _, o := range opts {
		o(c)
	}
	return c
}

// args builds the CLI argument list. ALL flag knowledge lives here.
// (Verified against `claude --help` at implementation time — see Task 10 step 1.)
func (c *CLIClient) args(sessionID, prompt string) []string {
	a := []string{"-p", "--output-format", "stream-json", "--session-id", sessionID}
	if c.PermissionMode != "" {
		a = append(a, "--permission-mode", c.PermissionMode)
	}
	if c.Model != "" {
		a = append(a, "--model", c.Model)
	}
	return append(a, prompt)
}

func (c *CLIClient) Start(ctx context.Context, sessionID, prompt string, w EventWriter) error {
	cmd := exec.CommandContext(ctx, c.Bin, c.args(sessionID, prompt)...)
	cmd.Dir = c.Dir
	// Put the CLI in its own process group and kill the whole group on cancel:
	// the CLI may spawn children that inherit stdout, and the stream reader
	// below only sees EOF once every writer closes. Killing just the direct
	// child can leave a grandchild holding the pipe open (seen with sh forking
	// `sleep` on macOS), hanging Start until the child exits on its own.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("claude stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

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
		return fmt.Errorf("claude exited: %w", waitErr)
	}
	if streamErr != nil {
		return fmt.Errorf("claude stream: %w", streamErr)
	}
	return nil
}
