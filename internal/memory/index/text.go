package index

import (
	"fmt"
	"strings"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

func indexText(art artifact.Artifact) string {
	parts := make([]string, 0, 8)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}

	add(art.Content.Text)
	add(art.Content.Role)
	add(art.Content.Status)
	add(art.Content.Rationale)
	add(art.Content.Scope)

	for _, summary := range art.Content.Summary {
		add(summary)
	}
	for _, file := range art.Content.Files {
		add(file)
	}
	for _, errText := range art.Content.Errors {
		add(errText)
	}
	for _, cmd := range art.Content.Commands {
		add(cmd.Command)
		if cmd.ExitCode != nil {
			add(fmt.Sprintf("exit %d", *cmd.ExitCode))
		}
	}

	return strings.Join(parts, "\n")
}

func boundSnippet(text string, maxLen int) string {
	cleaned := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if maxLen <= 0 {
		return cleaned
	}
	runes := []rune(cleaned)
	if len(runes) <= maxLen {
		return cleaned
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
