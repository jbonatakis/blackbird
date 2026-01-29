package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// ActionOutput represents the result of an action to display to the user
type ActionOutput struct {
	Message string
	IsError bool
}

// RenderActionOutput renders action output or error messages
func RenderActionOutput(output *ActionOutput, width int) string {
	if output == nil {
		return ""
	}

	style := lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder())

	if output.IsError {
		style = style.BorderForeground(lipgloss.Color("196")) // red
	} else {
		style = style.BorderForeground(lipgloss.Color("46")) // green
	}

	if width > 0 {
		style = style.Width(width - 4)
	}

	return style.Render(output.Message)
}

// RenderSetStatusModal renders the status selection modal
func RenderSetStatusModal(m Model) string {
	taskID := m.pendingStatusID
	if taskID == "" {
		return ""
	}

	item, ok := m.plan.Items[taskID]
	if !ok {
		return ""
	}

	statuses := []plan.Status{
		plan.StatusTodo,
		plan.StatusQueued,
		plan.StatusInProgress,
		plan.StatusWaitingUser,
		plan.StatusBlocked,
		plan.StatusDone,
		plan.StatusFailed,
		plan.StatusSkipped,
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	itemStyle := lipgloss.NewStyle().Padding(0, 2)
	currentStyle := itemStyle.Copy().Foreground(lipgloss.Color("46"))

	title := titleStyle.Render(fmt.Sprintf("Set status for %s:", taskID))
	subtitle := fmt.Sprintf("Current: %s", item.Status)

	lines := []string{
		title,
		subtitle,
		"",
		"Select new status:",
		"",
	}

	for i, status := range statuses {
		key := fmt.Sprintf("%d", i+1)
		line := fmt.Sprintf("[%s] %s", key, status)

		if status == item.Status {
			lines = append(lines, currentStyle.Render(line)+" (current)")
		} else {
			lines = append(lines, itemStyle.Render(line))
		}
	}

	lines = append(lines, "", itemStyle.Render("[esc] Cancel"))

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

	// Center the modal
	if m.windowHeight > 0 {
		topPadding := (m.windowHeight - lipgloss.Height(modal)) / 2
		if topPadding > 0 {
			padding := lipgloss.NewStyle().PaddingTop(topPadding).Render(modal)
			return padding
		}
	}

	return modal
}

// HandleSetStatusKey handles key presses in set-status mode
func HandleSetStatusKey(m Model, key string) (Model, tea.Cmd) {
	if key == "esc" {
		m.actionMode = ActionModeNone
		m.pendingStatusID = ""
		return m, nil
	}

	// Handle status selection keys 1-8
	statusMap := map[string]plan.Status{
		"1": plan.StatusTodo,
		"2": plan.StatusQueued,
		"3": plan.StatusInProgress,
		"4": plan.StatusWaitingUser,
		"5": plan.StatusBlocked,
		"6": plan.StatusDone,
		"7": plan.StatusFailed,
		"8": plan.StatusSkipped,
	}

	status, ok := statusMap[key]
	if !ok {
		return m, nil
	}

	taskID := m.pendingStatusID
	m.actionMode = ActionModeNone
	m.pendingStatusID = ""
	m.actionInProgress = true
	m.actionName = fmt.Sprintf("Setting status to %s...", status)

	return m, tea.Batch(SetStatusCmd(taskID, string(status)), spinnerTickCmd())
}

// CanResume returns true if the selected task has a waiting run that can be resumed
func CanResume(m Model) bool {
	if m.selectedID == "" {
		return false
	}

	item, ok := m.plan.Items[m.selectedID]
	if !ok {
		return false
	}

	// Check if task is in waiting_user status
	if item.Status != plan.StatusWaitingUser {
		return false
	}

	// Check if there are any waiting runs for this task
	for _, run := range m.runData {
		if run.TaskID == m.selectedID && run.Status == "waiting_user" {
			return true
		}
	}

	return false
}
