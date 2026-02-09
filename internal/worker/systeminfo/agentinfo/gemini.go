package agentinfo

import "context"

// geminiDetector detects the Gemini CLI.
// Expected output: "0.17.1" (plain version string)
type geminiDetector struct {
	version versionFunc
}

func (d geminiDetector) detect(ctx context.Context) (Agent, bool) {
	out, ok := d.version(ctx, "gemini")
	if !ok {
		return Agent{}, false
	}

	return Agent{
		ID:      "gemini",
		Name:    "Gemini",
		Version: out,
	}, true
}
