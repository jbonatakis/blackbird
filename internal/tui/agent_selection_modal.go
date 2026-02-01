package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/agent"
)

// agentSelectionHighlightIndex returns the index into AgentRegistry for the current
// selection, clamped to valid range. Used when opening the modal and for rendering.
func agentSelectionHighlightIndex(m Model) int {
	n := len(agent.AgentRegistry)
	if n == 0 {
		return 0
	}
	for i, info := range agent.AgentRegistry {
		if info.ID == m.agentSelection.Agent.ID {
			return i
		}
	}
	return 0
}

// RenderAgentSelectionModal renders the agent selection modal.
func RenderAgentSelectionModal(m Model) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	itemStyle := lipgloss.NewStyle().Padding(0, 2)
	highlightStyle := itemStyle.Copy().Foreground(lipgloss.Color("46")).Background(lipgloss.Color("236"))
	currentStyle := itemStyle.Copy().Foreground(lipgloss.Color("240"))

	title := titleStyle.Render("Select agent:")
	current := fmt.Sprintf("Current: %s", agentLabel(m))

	lines := []string{
		title,
		current,
		"",
		"Choose a runtime:",
		"",
	}

	idx := m.agentSelectionHighlight
	if idx < 0 || idx >= len(agent.AgentRegistry) {
		idx = 0
	}
	for i, info := range agent.AgentRegistry {
		line := info.Label
		if info.ID == m.agentSelection.Agent.ID {
			line += " (current)"
		}
		if i == idx {
			lines = append(lines, highlightStyle.Render("  "+line))
		} else {
			lines = append(lines, itemStyle.Render("  "+line))
		}
	}

	lines = append(lines, "", currentStyle.Render("[↑/↓] move  [enter] select  [esc] cancel"))

	modalWidth := 50
	if m.windowWidth > 0 && m.windowWidth < modalWidth+4 {
		modalWidth = m.windowWidth - 4
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	if m.windowHeight > 0 {
		topPadding := (m.windowHeight - lipgloss.Height(modal)) / 2
		if topPadding > 0 {
			padding := lipgloss.NewStyle().PaddingTop(topPadding).Render(modal)
			return padding
		}
	}

	return modal
}

// HandleAgentSelectionKey handles key presses in agent selection mode.
func HandleAgentSelectionKey(m Model, key string) (Model, tea.Cmd) {
	if key == "esc" {
		m.actionMode = ActionModeNone
		return m, nil
	}

	n := len(agent.AgentRegistry)
	if n == 0 {
		return m, nil
	}

	switch key {
	case "up", "k":
		m.agentSelectionHighlight--
		if m.agentSelectionHighlight < 0 {
			m.agentSelectionHighlight = n - 1
		}
		return m, nil
	case "down", "j":
		m.agentSelectionHighlight++
		if m.agentSelectionHighlight >= n {
			m.agentSelectionHighlight = 0
		}
		return m, nil
	case "enter", " ":
		idx := m.agentSelectionHighlight
		if idx < 0 || idx >= n {
			idx = 0
		}
		selected := agent.AgentRegistry[idx]
		m.actionMode = ActionModeNone
		return m, SaveAgentSelectionCmd(string(selected.ID))
	default:
		return m, nil
	}
}
