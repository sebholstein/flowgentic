package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSessionMode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMode  SessionMode
		wantError bool
	}{
		{
			name:      "ask mode",
			input:     "ask",
			wantMode:  SessionModeAsk,
			wantError: false,
		},
		{
			name:      "architect mode",
			input:     "architect",
			wantMode:  SessionModeArchitect,
			wantError: false,
		},
		{
			name:      "code mode",
			input:     "code",
			wantMode:  SessionModeCode,
			wantError: false,
		},
		{
			name:      "invalid mode",
			input:     "invalid",
			wantMode:  "",
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantMode:  "",
			wantError: true,
		},
		{
			name:      "mixed case - should fail",
			input:     "Ask",
			wantMode:  "",
			wantError: true,
		},
		{
			name:      "uppercase - should fail",
			input:     "CODE",
			wantMode:  "",
			wantError: true,
		},
		{
			name:      "whitespace - should fail",
			input:     " code ",
			wantMode:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := ParseSessionMode(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown session mode")
				assert.Empty(t, mode)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMode, mode)
			}
		})
	}
}

func TestSessionModeConstants(t *testing.T) {
	t.Run("session mode constants have correct values", func(t *testing.T) {
		assert.Equal(t, SessionMode("ask"), SessionModeAsk)
		assert.Equal(t, SessionMode("architect"), SessionModeArchitect)
		assert.Equal(t, SessionMode("code"), SessionModeCode)
	})
}
