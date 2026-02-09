package agentinfo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexDetector(t *testing.T) {
	t.Run("parses version from real output", func(t *testing.T) {
		d := codexDetector{version: fakeVersion("codex", "codex-cli 0.80.0")}
		agent, ok := d.detect(context.Background())
		require.True(t, ok)
		assert.Equal(t, "codex", agent.ID)
		assert.Equal(t, "Codex", agent.Name)
		assert.Equal(t, "0.80.0", agent.Version)
	})

	t.Run("returns false when binary not found", func(t *testing.T) {
		d := codexDetector{version: notFound}
		_, ok := d.detect(context.Background())
		assert.False(t, ok)
	})
}
