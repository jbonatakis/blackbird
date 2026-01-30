package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func RenderHomeView(m Model) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	taglineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	shortcutStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1)

	statusLine := mutedStyle.Render("⊘ No plan found")
	if m.planExists {
		readyCount := len(execution.ReadyTasks(m.plan))
		blockedCount := countBlockedItems(m.plan)
		statusLine = successStyle.Render(fmt.Sprintf(
			"✓ Plan found: %d items, %d ready, %d blocked",
			len(m.plan.Items),
			readyCount,
			blockedCount,
		))
	}

	lines := []string{
		titleStyle.Render("blackbird"),
		taglineStyle.Render("Durable, dependency-aware planning and execution"),
		"",
		statusLine,
	}

	if m.planValidationErr != "" {
		lines = append(lines,
			"",
			errorStyle.Render(fmt.Sprintf("⚠ Plan has errors: %s\nPress [g] to regenerate or [v] to view and fix", m.planValidationErr)),
		)
	}

	lines = append(lines,
		"",
		renderActionLine("[g]", "Generate plan", true, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[v]", "View plan", m.planExists, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[r]", "Refine plan", m.planExists, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[e]", "Execute", m.canExecute(), shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[ctrl+c]", "Quit", true, shortcutStyle, actionStyle, mutedStyle),
	)

	content := strings.Join(lines, "\n")
	if m.windowWidth <= 0 || m.windowHeight <= 0 {
		return content
	}
	return lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, lipgloss.Center, content)
}

func renderActionLine(shortcut string, label string, enabled bool, shortcutStyle lipgloss.Style, actionStyle lipgloss.Style, mutedStyle lipgloss.Style) string {
	if !enabled {
		return mutedStyle.Render(shortcut + " " + label)
	}
	return shortcutStyle.Render(shortcut) + " " + actionStyle.Render(label)
}

func countBlockedItems(g plan.WorkGraph) int {
	count := 0
	for _, it := range g.Items {
		depsOK := len(plan.UnmetDeps(g, it)) == 0
		label := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
		if label == "BLOCKED" {
			count++
		}
	}
	return count
}
