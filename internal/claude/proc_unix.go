//go:build unix

package claude

import (
	"os/exec"
	"syscall"
)

// setProcessGroup makes the child run in its own process group so a cancel
// can kill the whole group: the CLI may spawn children that inherit stdout
// and killing just the direct child can leave a grandchild holding the pipe
// open, hanging the stream reader until it exits on its own. Must be called
// before Start; the group id (the child's pid) is read off the started cmd.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup kills the whole process group (children included). The
// child's pgid equals its pid when setProcessGroup armed Setpgid.
func (c *CLIClient) killProcessGroup(pgid int) error {
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
