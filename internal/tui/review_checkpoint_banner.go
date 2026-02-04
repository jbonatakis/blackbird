package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderActionRequiredBanner(m Model) string {
	run := pendingDecisionRun(m)
	if run == nil {
		return ""
	}

	title := strings.TrimSpace(run.Context.Task.Title)
	if title == "" {
		if item, ok := m.plan.Items[run.TaskID]; ok {
			title = item.Title
		}
	}

	detail := run.TaskID
	if title != "" {
		detail = fmt.Sprintf("%s - %s", run.TaskID, title)
	}

	message := fmt.Sprintf("ACTION REQUIRED: Review %s", detail)

	width := m.windowWidth
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("214")).
		Padding(0, 1)

	if width > 0 {
		trimmed := truncateField(message, width-2)
		style = style.Width(width)
		return style.Render(trimmed)
	}

	return style.Render(message)
}
