package agentinfo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpencodeDetector(t *testing.T) {
	t.Run("parses version from real output", func(t *testing.T) {
		d := opencodeDetector{version: fakeVersion("opencode", "1.1.49")}
		agent, ok := d.detect(context.Background())
		require.True(t, ok)
		assert.Equal(t, "opencode", agent.ID)
		assert.Equal(t, "OpenCode", agent.Name)
		assert.Equal(t, "1.1.49", agent.Version)
	})

	t.Run("returns false when binary not found", func(t *testing.T) {
		d := opencodeDetector{version: notFound}
		_, ok := d.detect(context.Background())
		assert.False(t, ok)
	})
}
