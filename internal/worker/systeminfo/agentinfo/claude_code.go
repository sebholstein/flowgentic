package agentinfo

import (
	"context"
	"strings"
)

// claudeCodeDetector detects the Claude Code CLI.
// Expected output: "2.1.34 (Claude Code)"
type claudeCodeDetector struct {
	version versionFunc
}

func (d claudeCodeDetector) detect(ctx context.Context) (Agent, bool) {
	out, ok := d.version(ctx, "claude")
	if !ok {
		return Agent{}, false
	}

	version := "unknown"
	if fields := strings.Fields(out); len(fields) > 0 {
		version = fields[0]
	}

	return Agent{
		ID:      "claude-code",
		Name:    "Claude Code",
		Version: version,
	}, true
}
