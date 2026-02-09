package agentinfo

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// versionFunc runs a binary with --version and returns the trimmed output.
// The second return value is false when the binary is not found, the command
// fails, or the output is empty.
type versionFunc func(ctx context.Context, binary string) (string, bool)

// runVersionCommand is the default versionFunc. It checks whether binary
// exists on PATH and executes "binary --version" with a 5-second timeout.
func runVersionCommand(ctx context.Context, binary string) (string, bool) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", false
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", false
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", false
	}
	return out, true
}
