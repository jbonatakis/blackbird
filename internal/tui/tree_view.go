package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
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

	taskTree := plan.BuildTaskTree(model.plan)
	if len(taskTree.Roots) == 0 {
		muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return muted.Render("No root items.")
	}

	root := tree.New()
	visited := map[string]bool{}
	for _, id := range taskTree.Roots {
		node, matched := buildTreeNode(model, taskTree, id, visited)
		if matched && node != nil {
			root.Child(node)
		}
	}

	if root.Children().Length() == 0 {
		muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return muted.Render("No matching items.")
	}

	return root.String()
}

func buildTreeNode(model Model, taskTree plan.TaskTree, id string, visited map[string]bool) (*tree.Tree, bool) {
	if visited[id] {
		return nil, false
	}
	visited[id] = true

	it, ok := model.plan.Items[id]
	if !ok {
		return nil, false
	}

	children := append([]string{}, taskTree.Children[it.ID]...)

	depsOK := len(plan.UnmetDeps(model.plan, it)) == 0
	label := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
	matchesSelf := filterMatch(model.filterMode, label)

	isExpanded := isExpanded(model, it.ID)
	hasChildren := len(children) > 0
	var childNodes []any
	var childMatched bool
	for _, childID := range children {
		childNode, matched := buildTreeNode(model, taskTree, childID, visited)
		if matched {
			childMatched = true
		}
		if isExpanded && childNode != nil {
			childNodes = append(childNodes, childNode)
		}
	}

	shouldRender := matchesSelf || childMatched
	if !shouldRender {
		return nil, false
	}

	line := renderTreeLine(model, it, label, hasChildren, isExpanded)
	node := tree.New().Root(line)
	if isExpanded {
		node.Child(childNodes...)
	}
	return node, true
}

func renderTreeLine(model Model, it plan.WorkItem, readiness string, hasChildren bool, isExpanded bool) string {
	indicator := " "
	if hasChildren {
		if isExpanded {
			indicator = "▼"
		} else {
			indicator = "▶"
		}
	}

	readinessStyle := readinessLabelStyle(readiness)
	indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	compactReadiness := readinessAbbrev(readiness)
	rawTitle := truncateField(it.Title, maxTitleWidth(model.windowWidth, indicator, compactReadiness))

	readinessLabel := readinessStyle.Render(compactReadiness)
	line := strings.Join([]string{
		indicatorStyle.Render(indicator),
		readinessLabel,
		rawTitle,
	}, " ")

	if it.ID == model.selectedID {
		selected := lipgloss.NewStyle().Reverse(true).Bold(true)
		return selected.Render(line)
	}
	return line
}

func readinessAbbrev(label string) string {
	switch label {
	case "READY":
		return "R"
	case "DONE":
		return "D"
	case "IN_PROGRESS":
		return "IP"
	case "BLOCKED":
		return "B"
	case "QUEUED":
		return "Q"
	case "WAITING_USER":
		return "W"
	case "FAILED":
		return "F"
	case "SKIPPED":
		return "S"
	default:
		return label
	}
}

func truncateField(value string, max int) string {
	if max <= 0 || value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func maxTitleWidth(windowWidth int, indicator string, readiness string) int {
	if windowWidth <= 0 {
		return 48
	}
	overhead := lipgloss.Width(strings.Join([]string{indicator, readiness}, " "))
	overhead += 4 // tree prefixes and padding
	available := windowWidth - overhead
	if available < 10 {
		return 10
	}
	return available
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
