package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type FilterMode int

const (
	FilterModeAll FilterMode = iota
	FilterModeReady
	FilterModeBlocked
)

func RenderTreeView(model Model) string {
	if len(model.plan.Items) == 0 {
		muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return muted.Render("No items.")
	}

	roots := rootIDs(model.plan)
	if len(roots) == 0 {
		muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return muted.Render("No root items.")
	}

	var lines []string
	visited := map[string]bool{}
	for _, id := range roots {
		branchLines, _ := renderTreeItem(model, id, 0, visited)
		lines = append(lines, branchLines...)
	}

	if len(lines) == 0 {
		muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return muted.Render("No matching items.")
	}

	return strings.Join(lines, "\n")
}

func renderTreeItem(model Model, id string, depth int, visited map[string]bool) ([]string, bool) {
	if visited[id] {
		return nil, false
	}
	visited[id] = true

	it, ok := model.plan.Items[id]
	if !ok {
		return nil, false
	}

	children := append([]string{}, it.ChildIDs...)
	sort.Strings(children)

	depsOK := len(plan.UnmetDeps(model.plan, it)) == 0
	label := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
	matchesSelf := filterMatch(model.filterMode, label)

	isExpanded := isExpanded(model, it.ID)
	var childLines []string
	var childMatched bool
	for _, childID := range children {
		branchLines, matched := renderTreeItem(model, childID, depth+1, visited)
		if matched {
			childMatched = true
		}
		if isExpanded {
			childLines = append(childLines, branchLines...)
		}
	}

	shouldRender := matchesSelf || childMatched
	if !shouldRender {
		return nil, false
	}

	line := renderTreeLine(model, it, label, depth)
	lines := []string{line}
	if isExpanded {
		lines = append(lines, childLines...)
	}
	return lines, true
}

func renderTreeLine(model Model, it plan.WorkItem, readiness string, depth int) string {
	indent := strings.Repeat("  ", depth)
	indicator := " "
	if len(it.ChildIDs) > 0 {
		if isExpanded(model, it.ID) {
			indicator = "▼"
		} else {
			indicator = "▶"
		}
	}

	statusStyle := statusStyle(it.Status, readiness)
	readinessStyle := readinessLabelStyle(readiness)
	indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	status := statusStyle.Render(string(it.Status))
	readinessLabel := readinessStyle.Render(readiness)
	line := strings.Join([]string{
		indent + indicatorStyle.Render(indicator),
		it.ID,
		status,
		readinessLabel,
		it.Title,
	}, " "))

	if it.ID == model.selectedID {
		selected := lipgloss.NewStyle().Reverse(true).Bold(true)
		return selected.Render(line)
	}
	return line
}

func statusStyle(status plan.Status, readiness string) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	switch status {
	case plan.StatusDone:
		style = style.Foreground(lipgloss.Color("42"))
	case plan.StatusInProgress:
		style = style.Foreground(lipgloss.Color("214"))
	case plan.StatusBlocked:
		style = style.Foreground(lipgloss.Color("196"))
	case plan.StatusTodo:
		if readiness == "READY" {
			style = style.Foreground(lipgloss.Color("39"))
		} else if readiness == "BLOCKED" {
			style = style.Foreground(lipgloss.Color("196"))
		}
	}
	return style
}

func readinessLabelStyle(label string) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	switch label {
	case "READY":
		style = style.Foreground(lipgloss.Color("39"))
	case "DONE":
		style = style.Foreground(lipgloss.Color("42"))
	case "IN_PROGRESS":
		style = style.Foreground(lipgloss.Color("214"))
	case "BLOCKED":
		style = style.Foreground(lipgloss.Color("196"))
	}
	return style
}

func isExpanded(model Model, id string) bool {
	if model.expandedItems == nil {
		return true
	}
	expanded, ok := model.expandedItems[id]
	if !ok {
		return true
	}
	return expanded
}

func filterMatch(mode FilterMode, readiness string) bool {
	switch mode {
	case FilterModeReady:
		return readiness == "READY"
	case FilterModeBlocked:
		return readiness == "BLOCKED"
	default:
		return true
	}
}

func rootIDs(g plan.WorkGraph) []string {
	out := make([]string, 0)
	for id, it := range g.Items {
		if it.ParentID == nil || *it.ParentID == "" {
			out = append(out, id)
			continue
		}
		if _, ok := g.Items[*it.ParentID]; !ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
