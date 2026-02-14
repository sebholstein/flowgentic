package acp

import (
	"fmt"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
)

// toolMetadata holds ACP-enriched metadata derived from a Claude tool name and its input.
type toolMetadata struct {
	Title     string
	Kind      acpsdk.ToolKind
	Content   []acpsdk.ToolCallContent
	Locations []acpsdk.ToolCallLocation
}

// claudeCodeMeta is the _meta payload attached to tool calls, identifying the
// originating Claude Code tool name.
type claudeCodeMeta struct {
	ClaudeCode struct {
		ToolName string `json:"toolName"`
	} `json:"claudeCode"`
}

// newClaudeCodeMeta creates a _meta value for the given tool name.
func newClaudeCodeMeta(toolName string) claudeCodeMeta {
	var m claudeCodeMeta
	m.ClaudeCode.ToolName = toolName
	return m
}

// toolInfoFromToolUse maps a Claude Code tool name and its input to ACP-enriched metadata.
func toolInfoFromToolUse(toolName string, input map[string]any) toolMetadata {
	switch toolName {
	case "Bash":
		return bashToolInfo(input)
	case "Read":
		return readToolInfo(input)
	case "Edit":
		return editToolInfo(input)
	case "Write":
		return writeToolInfo(input)
	case "Glob":
		return globToolInfo(input)
	case "Grep":
		return grepToolInfo(input)
	case "WebFetch":
		return webFetchToolInfo(input)
	case "WebSearch":
		return webSearchToolInfo(input)
	case "Task":
		return taskToolInfo(input)
	case "TodoWrite":
		return todoWriteToolInfo(input)
	case "ExitPlanMode":
		return exitPlanModeToolInfo(input)
	default:
		return toolMetadata{
			Title: toolName,
			Kind:  acpsdk.ToolKindOther,
		}
	}
}

func bashToolInfo(input map[string]any) toolMetadata {
	cmd, _ := input["command"].(string)
	title := "Bash"
	if cmd != "" {
		// Use the description field if available, otherwise show truncated command.
		if desc, ok := input["description"].(string); ok && desc != "" {
			title = desc
		} else {
			title = fmt.Sprintf("`%s`", truncate(cmd, 60))
		}
	}

	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindExecute,
	}

	// Add text content from description.
	if desc, _ := input["description"].(string); desc != "" {
		tm.Content = []acpsdk.ToolCallContent{
			acpsdk.ToolContent(acpsdk.ContentBlock{
				Text: &acpsdk.ContentBlockText{Text: desc, Type: "text"},
			}),
		}
	}

	return tm
}

func readToolInfo(input map[string]any) toolMetadata {
	path, _ := input["file_path"].(string)
	title := "Read"
	if path != "" {
		title = fmt.Sprintf("Read %s", path)
		if offset, ok := input["offset"].(float64); ok {
			limit, _ := input["limit"].(float64)
			if limit > 0 {
				title = fmt.Sprintf("Read %s (%d-%d)", path, int(offset), int(offset+limit))
			}
		}
	}

	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindRead,
	}
	if path != "" {
		loc := acpsdk.ToolCallLocation{Path: path}
		if offset, ok := input["offset"].(float64); ok {
			line := int(offset)
			loc.Line = &line
		}
		tm.Locations = []acpsdk.ToolCallLocation{loc}
	}
	return tm
}

func editToolInfo(input map[string]any) toolMetadata {
	path, _ := input["file_path"].(string)
	title := "Edit"
	if path != "" {
		title = fmt.Sprintf("Edit `%s`", path)
	}

	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindEdit,
	}

	// Build diff content.
	oldStr, _ := input["old_string"].(string)
	newStr, _ := input["new_string"].(string)
	if path != "" {
		tm.Content = []acpsdk.ToolCallContent{
			acpsdk.ToolDiffContent(path, newStr, oldStr),
		}
	}

	if path != "" {
		tm.Locations = []acpsdk.ToolCallLocation{{Path: path}}
	}
	return tm
}

func writeToolInfo(input map[string]any) toolMetadata {
	path, _ := input["file_path"].(string)
	title := "Write"
	if path != "" {
		title = fmt.Sprintf("Write %s", path)
	}

	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindEdit,
	}

	// Build diff content (new file — no old text).
	content, _ := input["content"].(string)
	if path != "" && content != "" {
		tm.Content = []acpsdk.ToolCallContent{
			acpsdk.ToolDiffContent(path, content),
		}
	}

	if path != "" {
		tm.Locations = []acpsdk.ToolCallLocation{{Path: path}}
	}
	return tm
}

func globToolInfo(input map[string]any) toolMetadata {
	pattern, _ := input["pattern"].(string)
	path, _ := input["path"].(string)
	title := "Find"
	if path != "" && pattern != "" {
		title = fmt.Sprintf("Find `%s` `%s`", path, pattern)
	} else if pattern != "" {
		title = fmt.Sprintf("Find `%s`", pattern)
	}

	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindSearch,
	}
	if path != "" {
		tm.Locations = []acpsdk.ToolCallLocation{{Path: path}}
	}
	return tm
}

func grepToolInfo(input map[string]any) toolMetadata {
	pattern, _ := input["pattern"].(string)
	path, _ := input["path"].(string)
	title := "grep"
	if pattern != "" && path != "" {
		title = fmt.Sprintf("grep %q %s", pattern, path)
	} else if pattern != "" {
		title = fmt.Sprintf("grep %q", pattern)
	}

	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindSearch,
	}
	if path != "" {
		tm.Locations = []acpsdk.ToolCallLocation{{Path: path}}
	}
	return tm
}

func webFetchToolInfo(input map[string]any) toolMetadata {
	url, _ := input["url"].(string)
	title := "Fetch"
	if url != "" {
		title = fmt.Sprintf("Fetch %s", truncate(url, 60))
	}
	return toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindFetch,
	}
}

func webSearchToolInfo(input map[string]any) toolMetadata {
	query, _ := input["query"].(string)
	title := "Search"
	if query != "" {
		title = fmt.Sprintf("%q", query)
	}
	return toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindFetch,
	}
}

func taskToolInfo(input map[string]any) toolMetadata {
	desc, _ := input["description"].(string)
	title := "Task"
	if desc != "" {
		title = desc
	}
	return toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindThink,
	}
}

func todoWriteToolInfo(input map[string]any) toolMetadata {
	title := "Update TODOs"
	if todos, ok := input["todos"].([]any); ok && len(todos) > 0 {
		var subjects []string
		for _, t := range todos {
			if m, ok := t.(map[string]any); ok {
				if s, ok := m["subject"].(string); ok {
					subjects = append(subjects, s)
				}
			}
		}
		if len(subjects) > 0 {
			title = fmt.Sprintf("Update TODOs: %s", truncate(strings.Join(subjects, ", "), 60))
		}
	}
	return toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindThink,
	}
}

func exitPlanModeToolInfo(input map[string]any) toolMetadata {
	title := "Ready to code?"
	tm := toolMetadata{
		Title: title,
		Kind:  acpsdk.ToolKindSwitchMode,
	}
	if plan, ok := input["plan"].(string); ok && plan != "" {
		tm.Content = []acpsdk.ToolCallContent{
			acpsdk.ToolContent(acpsdk.ContentBlock{
				Text: &acpsdk.ContentBlockText{Text: plan, Type: "text"},
			}),
		}
	}
	return tm
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
