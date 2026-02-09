package agentinfo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmpDetector(t *testing.T) {
	t.Run("parses version from real output", func(t *testing.T) {
		d := ampDetector{version: fakeVersion("amp", "0.0.1769673940-g18d695 (released 2026-01-29T08:10:32.452Z, 8d ago)")}
		agent, ok := d.detect(context.Background())
		require.True(t, ok)
		assert.Equal(t, "amp", agent.ID)
		assert.Equal(t, "Amp", agent.Name)
		assert.Equal(t, "0.0.1769673940-g18d695", agent.Version)
	})

	t.Run("returns false when binary not found", func(t *testing.T) {
		d := ampDetector{version: notFound}
		_, ok := d.detect(context.Background())
		assert.False(t, ok)
	})
}
