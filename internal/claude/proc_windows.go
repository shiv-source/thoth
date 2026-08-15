//go:build windows

package claude

import (
	"os"
	"os/exec"
)

// setProcessGroup is a no-op on windows: there are no process groups to kill,
// so cancels kill the direct child via CLIClient.process instead.
func setProcessGroup(_ *exec.Cmd) {}

// killProcess kills the direct child on windows. The pgid argument is
// ignored; on unix it names the process group to kill instead.
func killProcess(_ int, proc *os.Process) error {
	if proc == nil {
		return nil
	}
	return proc.Kill()
}

// killProcessGroup kills the direct child on windows: the running process is
// the one stored on the client at Start.
func (c *CLIClient) killProcessGroup(_ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return killProcess(0, c.process)
}
