package acp

import (
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolInfoFromToolUse_Bash(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		info := toolInfoFromToolUse("Bash", map[string]any{
			"command":     "ls -la /tmp",
			"description": "List files in /tmp",
		})
		assert.Equal(t, "List files in /tmp", info.Title)
		assert.Equal(t, acpsdk.ToolKindExecute, info.Kind)
		require.Len(t, info.Content, 1)
		require.NotNil(t, info.Content[0].Content)
		assert.Equal(t, "List files in /tmp", info.Content[0].Content.Content.Text.Text)
	})

	t.Run("without description", func(t *testing.T) {
		info := toolInfoFromToolUse("Bash", map[string]any{
			"command": "ls -la /tmp",
		})
		assert.Equal(t, "`ls -la /tmp`", info.Title)
		assert.Equal(t, acpsdk.ToolKindExecute, info.Kind)
	})

	t.Run("long command truncated", func(t *testing.T) {
		longCmd := "some-command --with-very-long-flags --and-more-flags --even-more-flags-here /path/to/something"
		info := toolInfoFromToolUse("Bash", map[string]any{
			"command": longCmd,
		})
		assert.Contains(t, info.Title, "...")
		assert.LessOrEqual(t, len(info.Title), 65) // backticks + truncated
	})

	t.Run("empty input", func(t *testing.T) {
		info := toolInfoFromToolUse("Bash", map[string]any{})
		assert.Equal(t, "Bash", info.Title)
		assert.Equal(t, acpsdk.ToolKindExecute, info.Kind)
	})
}

func TestToolInfoFromToolUse_Read(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		info := toolInfoFromToolUse("Read", map[string]any{
			"file_path": "src/main.go",
		})
		assert.Equal(t, "Read src/main.go", info.Title)
		assert.Equal(t, acpsdk.ToolKindRead, info.Kind)
		require.Len(t, info.Locations, 1)
		assert.Equal(t, "src/main.go", info.Locations[0].Path)
		assert.Nil(t, info.Locations[0].Line)
	})

	t.Run("with offset and limit", func(t *testing.T) {
		info := toolInfoFromToolUse("Read", map[string]any{
			"file_path": "src/main.go",
			"offset":    float64(10),
			"limit":     float64(40),
		})
		assert.Equal(t, "Read src/main.go (10-50)", info.Title)
		require.Len(t, info.Locations, 1)
		assert.Equal(t, "src/main.go", info.Locations[0].Path)
		require.NotNil(t, info.Locations[0].Line)
		assert.Equal(t, 10, *info.Locations[0].Line)
	})
}

func TestToolInfoFromToolUse_Edit(t *testing.T) {
	info := toolInfoFromToolUse("Edit", map[string]any{
		"file_path":  "src/main.go",
		"old_string": "foo",
		"new_string": "bar",
	})
	assert.Equal(t, "Edit `src/main.go`", info.Title)
	assert.Equal(t, acpsdk.ToolKindEdit, info.Kind)
	require.Len(t, info.Content, 1)
	require.NotNil(t, info.Content[0].Diff)
	assert.Equal(t, "src/main.go", info.Content[0].Diff.Path)
	assert.Equal(t, "bar", info.Content[0].Diff.NewText)
	require.NotNil(t, info.Content[0].Diff.OldText)
	assert.Equal(t, "foo", *info.Content[0].Diff.OldText)
	require.Len(t, info.Locations, 1)
	assert.Equal(t, "src/main.go", info.Locations[0].Path)
}

func TestToolInfoFromToolUse_Write(t *testing.T) {
	info := toolInfoFromToolUse("Write", map[string]any{
		"file_path": "src/new.go",
		"content":   "package main\n",
	})
	assert.Equal(t, "Write src/new.go", info.Title)
	assert.Equal(t, acpsdk.ToolKindEdit, info.Kind)
	require.Len(t, info.Content, 1)
	require.NotNil(t, info.Content[0].Diff)
	assert.Equal(t, "src/new.go", info.Content[0].Diff.Path)
	assert.Equal(t, "package main\n", info.Content[0].Diff.NewText)
	assert.Nil(t, info.Content[0].Diff.OldText) // new file
	require.Len(t, info.Locations, 1)
}

func TestToolInfoFromToolUse_Glob(t *testing.T) {
	t.Run("with path", func(t *testing.T) {
		info := toolInfoFromToolUse("Glob", map[string]any{
			"pattern": "*.go",
			"path":    "src/",
		})
		assert.Equal(t, "Find `src/` `*.go`", info.Title)
		assert.Equal(t, acpsdk.ToolKindSearch, info.Kind)
		require.Len(t, info.Locations, 1)
		assert.Equal(t, "src/", info.Locations[0].Path)
	})

	t.Run("pattern only", func(t *testing.T) {
		info := toolInfoFromToolUse("Glob", map[string]any{
			"pattern": "*.go",
		})
		assert.Equal(t, "Find `*.go`", info.Title)
		assert.Empty(t, info.Locations)
	})
}

func TestToolInfoFromToolUse_Grep(t *testing.T) {
	info := toolInfoFromToolUse("Grep", map[string]any{
		"pattern": "TODO",
		"path":    "src/",
	})
	assert.Equal(t, `grep "TODO" src/`, info.Title)
	assert.Equal(t, acpsdk.ToolKindSearch, info.Kind)
	require.Len(t, info.Locations, 1)
}

func TestToolInfoFromToolUse_WebFetch(t *testing.T) {
	info := toolInfoFromToolUse("WebFetch", map[string]any{
		"url": "https://example.com/docs",
	})
	assert.Equal(t, "Fetch https://example.com/docs", info.Title)
	assert.Equal(t, acpsdk.ToolKindFetch, info.Kind)
}

func TestToolInfoFromToolUse_WebSearch(t *testing.T) {
	info := toolInfoFromToolUse("WebSearch", map[string]any{
		"query": "golang context",
	})
	assert.Equal(t, `"golang context"`, info.Title)
	assert.Equal(t, acpsdk.ToolKindFetch, info.Kind)
}

func TestToolInfoFromToolUse_Task(t *testing.T) {
	info := toolInfoFromToolUse("Task", map[string]any{
		"description": "Explore codebase",
	})
	assert.Equal(t, "Explore codebase", info.Title)
	assert.Equal(t, acpsdk.ToolKindThink, info.Kind)
}

func TestToolInfoFromToolUse_TodoWrite(t *testing.T) {
	t.Run("with todos", func(t *testing.T) {
		info := toolInfoFromToolUse("TodoWrite", map[string]any{
			"todos": []any{
				map[string]any{"subject": "Fix bug"},
				map[string]any{"subject": "Add tests"},
			},
		})
		assert.Contains(t, info.Title, "Fix bug")
		assert.Contains(t, info.Title, "Add tests")
		assert.Equal(t, acpsdk.ToolKindThink, info.Kind)
	})

	t.Run("empty todos", func(t *testing.T) {
		info := toolInfoFromToolUse("TodoWrite", map[string]any{})
		assert.Equal(t, "Update TODOs", info.Title)
	})
}

func TestToolInfoFromToolUse_ExitPlanMode(t *testing.T) {
	t.Run("with plan", func(t *testing.T) {
		info := toolInfoFromToolUse("ExitPlanMode", map[string]any{
			"plan": "Here is my implementation plan...",
		})
		assert.Equal(t, "Ready to code?", info.Title)
		assert.Equal(t, acpsdk.ToolKindSwitchMode, info.Kind)
		require.Len(t, info.Content, 1)
		require.NotNil(t, info.Content[0].Content)
	})

	t.Run("without plan", func(t *testing.T) {
		info := toolInfoFromToolUse("ExitPlanMode", map[string]any{})
		assert.Equal(t, "Ready to code?", info.Title)
		assert.Empty(t, info.Content)
	})
}

func TestToolInfoFromToolUse_Unknown(t *testing.T) {
	info := toolInfoFromToolUse("SomeCustomTool", map[string]any{})
	assert.Equal(t, "SomeCustomTool", info.Title)
	assert.Equal(t, acpsdk.ToolKindOther, info.Kind)
}

func TestNewClaudeCodeMeta(t *testing.T) {
	m := newClaudeCodeMeta("Read")
	assert.Equal(t, "Read", m.ClaudeCode.ToolName)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abc", 10))
	assert.Equal(t, "abcdefg...", truncate("abcdefghijklmnop", 10))
	assert.Equal(t, "ab", truncate("abcdef", 2))
}
