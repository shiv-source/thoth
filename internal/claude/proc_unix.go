//go:build unix

package claude

import (
	"os"
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

// killProcess kills the CLI process: on unix the whole process group
// (children included — the child's pgid equals its pid when setProcessGroup
// armed Setpgid), on windows the direct child. Shared by the per-turn
// CLIClient cancels and the persistent pool's eviction.
func killProcess(pgid int, _ *os.Process) error {
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// killProcessGroup is the CLIClient-shaped wrapper around killProcess.
func (c *CLIClient) killProcessGroup(pgid int) error {
	return killProcess(pgid, nil)
}
