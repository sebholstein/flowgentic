package thread

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveInitialTopic(t *testing.T) {
	t.Run("returns empty for blank input", func(t *testing.T) {
		assert.Equal(t, "", deriveInitialTopic(" \n\t "))
	})

	t.Run("cleans whitespace and wrapping quotes", func(t *testing.T) {
		got := deriveInitialTopic("  \"  build   login \n flow   \"  ")
		assert.Equal(t, "build login flow", got)
	})

	t.Run("truncates long input", func(t *testing.T) {
		in := strings.Repeat("a", 150)
		got := deriveInitialTopic(in)
		assert.Equal(t, 100, len([]rune(got)))
		assert.True(t, strings.HasSuffix(got, "..."))
	})
}
