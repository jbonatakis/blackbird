package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const filePickerEmptyMessage = "No matches. Keep typing to filter."

// RenderFilePickerList renders a compact list of file matches sized for plan modals.
// Returns an empty string when the picker is closed.
func RenderFilePickerList(state FilePickerState, width, height int) string {
	if !state.Open || width <= 0 || height <= 0 {
		return ""
	}

	itemStyle, selectedStyle, emptyStyle := filePickerListStyles(width)
	lines := make([]string, 0, height)

	if len(state.Matches) == 0 {
		lines = append(lines, emptyStyle.Render(truncateField(filePickerEmptyMessage, width)))
		content := strings.Join(lines, "\n")
		return lipgloss.NewStyle().Width(width).Height(height).Render(content)
	}

	start, end := filePickerVisibleWindow(len(state.Matches), state.Selected, height)
	for idx := start; idx < end; idx++ {
		selected := idx == state.Selected
		lines = append(lines, filePickerLine(state.Matches[idx], selected, width, itemStyle, selectedStyle))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func filePickerListStyles(width int) (itemStyle, selectedStyle, emptyStyle lipgloss.Style) {
	base := lipgloss.NewStyle().Width(width)
	itemStyle = base.Copy().Foreground(lipgloss.Color("15"))
	selectedStyle = base.Copy().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("69"))
	emptyStyle = base.Copy().Foreground(lipgloss.Color("240"))
	return
}

func filePickerLine(value string, selected bool, width int, itemStyle, selectedStyle lipgloss.Style) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	available := width - lipgloss.Width(prefix)
	if available < 1 {
		available = 1
	}
	line := prefix + truncateField(value, available)
	if selected {
		return selectedStyle.Render(line)
	}
	return itemStyle.Render(line)
}

func filePickerVisibleWindow(total, selected, height int) (start, end int) {
	if height <= 0 || total <= 0 {
		return 0, 0
	}
	if selected < 0 {
		selected = 0
	}
	start = 0
	if selected >= height {
		start = selected - height + 1
	}
	maxStart := total - height
	if maxStart < 0 {
		maxStart = 0
	}
	if start > maxStart {
		start = maxStart
	}
	end = start + height
	if end > total {
		end = total
	}
	return start, end
}
