//go:build darwin

package procutil

import "os/exec"

// StartWithCleanup starts the command. On macOS there is no kernel-level
// mechanism like Linux's Pdeathsig to kill a child when the parent dies.
// Graceful shutdown is handled by exec.CommandContext (context cancellation
// sends a signal to the child). Ungraceful worker death (SIGKILL) will
// leave orphaned children — there is no reliable in-process fix for this
// on macOS.
func StartWithCleanup(cmd *exec.Cmd) error {
	return cmd.Start()
}
