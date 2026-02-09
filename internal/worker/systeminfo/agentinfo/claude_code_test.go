package agentinfo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeCodeDetector(t *testing.T) {
	t.Run("parses version from real output", func(t *testing.T) {
		d := claudeCodeDetector{version: fakeVersion("claude", "2.1.34 (Claude Code)")}
		agent, ok := d.detect(context.Background())
		require.True(t, ok)
		assert.Equal(t, "claude-code", agent.ID)
		assert.Equal(t, "Claude Code", agent.Name)
		assert.Equal(t, "2.1.34", agent.Version)
	})

	t.Run("returns false when binary not found", func(t *testing.T) {
		d := claudeCodeDetector{version: notFound}
		_, ok := d.detect(context.Background())
		assert.False(t, ok)
	})
}
