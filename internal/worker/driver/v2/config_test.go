package v2

import (
	"testing"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/stretchr/testify/assert"
)

func TestDefaultMetaBuilder(t *testing.T) {
	t.Run("empty opts produce empty meta", func(t *testing.T) {
		meta := defaultMetaBuilder(LaunchOpts{})
		assert.Empty(t, meta)
	})

	t.Run("all fields populated", func(t *testing.T) {
		meta := defaultMetaBuilder(LaunchOpts{
			SystemPrompt: "be helpful",
			Model:        "claude-4",
			SessionMode:  "code",
			AllowedTools: []string{"Read", "Write"},
		})
		assert.Equal(t, "be helpful", meta["systemPrompt"])
		assert.Equal(t, "claude-4", meta["model"])
		assert.Equal(t, "code", meta["sessionMode"])
		assert.Equal(t, []string{"Read", "Write"}, meta["allowedTools"])
	})
}

func TestPredefinedConfigs(t *testing.T) {
	configs := []AgentConfig{ClaudeCodeConfig, CodexConfig, OpenCodeConfig, GeminiConfig}
	for _, cfg := range configs {
		t.Run(cfg.AgentID, func(t *testing.T) {
			assert.NotEmpty(t, cfg.AgentID)
			assert.NotEmpty(t, cfg.Capabilities)
			assert.NotNil(t, cfg.MetaBuilder)

			// Either subprocess command or adapter factory must be set (or neither for stub configs).
			if cfg.Command != "" {
				assert.Empty(t, cfg.AdapterFactory, "subprocess configs should not have AdapterFactory")
			}
		})
	}
}

func TestNewDriver(t *testing.T) {
	d := NewDriver(testLogger(), OpenCodeConfig)
	assert.Equal(t, string(driver.AgentTypeOpenCode), d.Agent())
	caps := d.Capabilities()
	assert.True(t, caps.Has(driver.CapStreaming))
}
