package agentinfo

import (
	"context"
	"strings"
)

// ampDetector detects the Amp CLI.
// Expected output: "0.0.17...-g18d695 (released ...)"
type ampDetector struct {
	version versionFunc
}

func (d ampDetector) detect(ctx context.Context) (Agent, bool) {
	out, ok := d.version(ctx, "amp")
	if !ok {
		return Agent{}, false
	}

	version := "unknown"
	if fields := strings.Fields(out); len(fields) > 0 {
		version = fields[0]
	}

	return Agent{
		ID:      "amp",
		Name:    "Amp",
		Version: version,
	}, true
}
