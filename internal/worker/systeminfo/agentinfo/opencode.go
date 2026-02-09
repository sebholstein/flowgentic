package agentinfo

import "context"

// opencodeDetector detects the OpenCode CLI.
// Expected output: "1.1.49" (plain version string)
type opencodeDetector struct {
	version versionFunc
}

func (d opencodeDetector) detect(ctx context.Context) (Agent, bool) {
	out, ok := d.version(ctx, "opencode")
	if !ok {
		return Agent{}, false
	}

	return Agent{
		ID:      "opencode",
		Name:    "OpenCode",
		Version: out,
	}, true
}
