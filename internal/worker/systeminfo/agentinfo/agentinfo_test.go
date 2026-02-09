package agentinfo

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverer(t *testing.T) {
	allOutputs := map[string]string{
		"claude":   "2.1.34 (Claude Code)",
		"codex":    "codex-cli 0.80.0",
		"opencode": "1.1.49",
		"amp":      "0.0.1769673940-g18d695 (released 2026-01-29T08:10:32.452Z, 8d ago)",
		"gemini":   "0.17.1",
	}

	allDetectors := func(v versionFunc) []detector {
		return []detector{
			claudeCodeDetector{version: v},
			codexDetector{version: v},
			opencodeDetector{version: v},
			ampDetector{version: v},
			geminiDetector{version: v},
		}
	}

	t.Run("all agents found", func(t *testing.T) {
		v := func(_ context.Context, binary string) (string, bool) {
			out, ok := allOutputs[binary]
			return out, ok
		}
		d := &Discoverer{detectors: allDetectors(v)}

		agents, err := d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)

		ids := make([]string, len(agents))
		for i, a := range agents {
			ids[i] = a.ID
		}
		assert.ElementsMatch(t, []string{"claude-code", "codex", "opencode", "amp", "gemini"}, ids)
	})

	t.Run("no agents found", func(t *testing.T) {
		d := &Discoverer{detectors: allDetectors(notFound)}

		agents, err := d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)
		assert.Empty(t, agents)
	})

	t.Run("partial agents found", func(t *testing.T) {
		v := func(_ context.Context, binary string) (string, bool) {
			if binary == "claude" {
				return "2.1.34 (Claude Code)", true
			}
			if binary == "gemini" {
				return "0.17.1", true
			}
			return "", false
		}
		d := &Discoverer{detectors: allDetectors(v)}

		agents, err := d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)

		ids := make([]string, len(agents))
		for i, a := range agents {
			ids[i] = a.ID
		}
		assert.ElementsMatch(t, []string{"claude-code", "gemini"}, ids)
	})

	t.Run("returns cached result on second call", func(t *testing.T) {
		var calls atomic.Int32
		v := func(_ context.Context, binary string) (string, bool) {
			calls.Add(1)
			out, ok := allOutputs[binary]
			return out, ok
		}

		d := &Discoverer{detectors: allDetectors(v)}

		agents1, err := d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)
		require.Len(t, agents1, 5)
		firstCalls := calls.Load()

		// Second call should return cached result.
		agents2, err := d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)
		assert.Equal(t, agents1, agents2)
		assert.Equal(t, firstCalls, calls.Load(), "detectors should not be called again")
	})

	t.Run("disableCache bypasses cache and updates it", func(t *testing.T) {
		var calls atomic.Int32
		v := func(_ context.Context, binary string) (string, bool) {
			calls.Add(1)
			out, ok := allOutputs[binary]
			return out, ok
		}

		d := &Discoverer{detectors: allDetectors(v)}

		_, err := d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)
		firstCalls := calls.Load()

		// disableCache=true should re-run detectors.
		_, err = d.DiscoverAgents(context.Background(), true)
		require.NoError(t, err)
		assert.Greater(t, calls.Load(), firstCalls, "detectors should be called again with disableCache")

		secondCalls := calls.Load()

		// Next call without disableCache should use the refreshed cache.
		_, err = d.DiscoverAgents(context.Background(), false)
		require.NoError(t, err)
		assert.Equal(t, secondCalls, calls.Load(), "detectors should not be called when cache is valid")
	})
}
