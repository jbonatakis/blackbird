package contextpack

import (
	"fmt"
	"strings"
)

// Render formats the context pack into a human-readable text block.
func Render(pack ContextPack) string {
	var b strings.Builder
	b.WriteString("Session context pack")
	if pack.SchemaVersion != 0 {
		b.WriteString(fmt.Sprintf(" (schema %d)", pack.SchemaVersion))
	}
	b.WriteString("\n")

	if pack.SessionID != "" {
		b.WriteString("Session ID: ")
		b.WriteString(pack.SessionID)
		b.WriteString("\n")
	}
	if pack.SessionGoal != "" {
		b.WriteString(goalLine(pack.SessionGoal))
		b.WriteString("\n")
	}

	writeSection(&b, "Instructions", pack.Instructions)
	writeSection(&b, "Decisions", pack.Decisions.Items)
	writeSection(&b, "Constraints", pack.Constraints.Items)
	writeSection(&b, "Recent outcomes", pack.Implemented.Items)
	writeSection(&b, "Open threads", pack.OpenThreads.Items)
	writeSection(&b, "Artifact IDs", pack.ArtifactIDs.Items)

	return strings.TrimSpace(b.String())
}

func writeSection(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString(":\n")
	for _, item := range items {
		line := strings.TrimSpace(item)
		if line == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteString("\n")
	}
}
