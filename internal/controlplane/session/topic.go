package session

import "strings"

const maxInitialTopicRunes = 100

func deriveInitialTopic(prompt string) string {
	cleaned := strings.Join(strings.Fields(prompt), " ")
	cleaned = strings.Trim(cleaned, " \t\r\n\"'`")
	if cleaned == "" {
		return ""
	}

	runes := []rune(cleaned)
	if len(runes) <= maxInitialTopicRunes {
		return cleaned
	}

	truncated := strings.TrimSpace(string(runes[:maxInitialTopicRunes-3]))
	if truncated == "" {
		return ""
	}
	return truncated + "..."
}
