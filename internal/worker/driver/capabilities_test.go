package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilities_Has(t *testing.T) {
	caps := Capabilities{
		Agent: "test",
		Supported: []Capability{
			CapStreaming,
			CapCostTracking,
		},
	}

	t.Run("returns true for supported capability", func(t *testing.T) {
		assert.True(t, caps.Has(CapStreaming))
		assert.True(t, caps.Has(CapCostTracking))
	})

	t.Run("returns false for unsupported capability", func(t *testing.T) {
		assert.False(t, caps.Has(CapSessionResume))
		assert.False(t, caps.Has(CapYolo))
	})
}

func TestCapabilities_Has_Empty(t *testing.T) {
	caps := Capabilities{
		Agent:     "test",
		Supported: []Capability{},
	}

	assert.False(t, caps.Has(CapStreaming))
}
