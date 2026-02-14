package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCutEnv(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantKey string
		wantVal string
		wantOk  bool
	}{
		{
			name:    "standard key=value pair",
			entry:   "FOO=bar",
			wantKey: "FOO",
			wantVal: "bar",
			wantOk:  true,
		},
		{
			name:    "key with empty value",
			entry:   "EMPTY=",
			wantKey: "EMPTY",
			wantVal: "",
			wantOk:  true,
		},
		{
			name:    "value containing equals signs",
			entry:   "PATH=/usr/bin:/bin",
			wantKey: "PATH",
			wantVal: "/usr/bin:/bin",
			wantOk:  true,
		},
		{
			name:    "key only without equals",
			entry:   "NOEQUALS",
			wantKey: "NOEQUALS",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "empty string",
			entry:   "",
			wantKey: "",
			wantVal: "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val, ok := cutEnv(tt.entry)
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestBuildEnv(t *testing.T) {
	t.Run("returns base environment when extra is empty", func(t *testing.T) {
		// We can't easily test os.Environ() content, but we can verify
		// the function doesn't panic and returns non-nil
		result := BuildEnv(nil)
		assert.NotNil(t, result)

		result = BuildEnv(map[string]string{})
		assert.NotNil(t, result)
	})

	t.Run("adds extra variables to environment", func(t *testing.T) {
		extra := map[string]string{
			"TEST_VAR": "test_value",
		}
		result := BuildEnv(extra)

		// Check that our extra var is in the result
		found := false
		for _, entry := range result {
			if entry == "TEST_VAR=test_value" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected TEST_VAR=test_value in result")
	})

	t.Run("extra overrides existing variables", func(t *testing.T) {
		// Set an environment variable that we'll override
		t.Setenv("OVERRIDE_ME", "original")

		extra := map[string]string{
			"OVERRIDE_ME": "overridden",
		}
		result := BuildEnv(extra)

		// Count occurrences - should be exactly 1 with the new value
		count := 0
		var foundValue string
		for _, entry := range result {
			key, _, _ := cutEnv(entry)
			if key == "OVERRIDE_ME" {
				count++
				foundValue = entry
			}
		}

		assert.Equal(t, 1, count, "expected exactly one OVERRIDE_ME entry")
		assert.Equal(t, "OVERRIDE_ME=overridden", foundValue)
	})

	t.Run("handles multiple extra variables", func(t *testing.T) {
		extra := map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		}
		result := BuildEnv(extra)

		// Build a map of our extra vars found in result
		found := make(map[string]bool)
		for _, entry := range result {
			if entry == "VAR1=value1" || entry == "VAR2=value2" || entry == "VAR3=value3" {
				key, _, _ := cutEnv(entry)
				found[key] = true
			}
		}

		assert.Len(t, found, 3, "expected all three extra variables to be present")
		assert.True(t, found["VAR1"])
		assert.True(t, found["VAR2"])
		assert.True(t, found["VAR3"])
	})
}
