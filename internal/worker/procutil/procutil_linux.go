//go:build linux

package procutil

import (
	"os/exec"
	"syscall"
)

// StartWithCleanup configures the command so the child is killed when the
// worker dies (via Pdeathsig on Linux), then starts it.
func StartWithCleanup(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
	return cmd.Start()
}
