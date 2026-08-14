//go:build windows

package claude

import "os/exec"

// setProcessGroup is a no-op on windows: there are no process groups to kill,
// so cancels kill the direct child via CLIClient.process instead.
func setProcessGroup(_ *exec.Cmd) {}

// killProcessGroup kills the direct child on windows. The pgid argument is
// ignored; the running process is the one stored on the client at Start.
func (c *CLIClient) killProcessGroup(_ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.process != nil {
		return c.process.Kill()
	}
	return nil
}
