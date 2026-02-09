package agentinfo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiDetector(t *testing.T) {
	t.Run("parses version from real output", func(t *testing.T) {
		d := geminiDetector{version: fakeVersion("gemini", "0.17.1")}
		agent, ok := d.detect(context.Background())
		require.True(t, ok)
		assert.Equal(t, "gemini", agent.ID)
		assert.Equal(t, "Gemini", agent.Name)
		assert.Equal(t, "0.17.1", agent.Version)
	})

	t.Run("returns false when binary not found", func(t *testing.T) {
		d := geminiDetector{version: notFound}
		_, ok := d.detect(context.Background())
		assert.False(t, ok)
	})
}
