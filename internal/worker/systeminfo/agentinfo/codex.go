package agentinfo

import (
	"context"
	"strings"
)

// codexDetector detects the OpenAI Codex CLI.
// Expected output: "codex-cli 0.80.0"
type codexDetector struct {
	version versionFunc
}

func (d codexDetector) detect(ctx context.Context) (Agent, bool) {
	out, ok := d.version(ctx, "codex")
	if !ok {
		return Agent{}, false
	}

	version := "unknown"
	if fields := strings.Fields(out); len(fields) > 0 {
		version = fields[len(fields)-1]
	}

	return Agent{
		ID:      "codex",
		Name:    "Codex",
		Version: version,
	}, true
}
