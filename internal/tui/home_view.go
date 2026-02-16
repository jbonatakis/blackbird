package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// planStatusCounts holds per-status counts and leaf completion for the home view.
type planStatusCounts struct {
	Ready       int
	Blocked     int
	InProgress  int
	Queued      int
	WaitingUser int
	Done        int
	Failed      int
	Skipped     int
	LeafTotal   int
	LeafDone    int
}

func countPlanStatuses(g plan.WorkGraph) planStatusCounts {
	var c planStatusCounts
	for _, it := range g.Items {
		depsOK := len(plan.UnmetDeps(g, it)) == 0
		label := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
		switch label {
		case "READY":
			c.Ready++
		case "BLOCKED":
			c.Blocked++
		case "IN_PROGRESS":
			c.InProgress++
		case "QUEUED":
			c.Queued++
		case "WAITING_USER":
			c.WaitingUser++
		case "DONE":
			c.Done++
		case "FAILED":
			c.Failed++
		case "SKIPPED":
			c.Skipped++
		}
		// Leaf = no children; completion % is based on leaf tasks only.
		if len(it.ChildIDs) == 0 {
			c.LeafTotal++
			if it.Status == plan.StatusDone {
				c.LeafDone++
			}
		}
	}
	return c
}

// homeStatusStyle returns the same colors as tree view readinessLabelStyle
// so status counts on the home screen match the plan view.
func homeStatusStyle(theme Theme, label string) lipgloss.Style {
	return readinessLabelStyleForTheme(theme, label)
}

// formatPlanStatusLines returns multiple styled lines for the plan summary.
func formatPlanStatusLines(theme Theme, total int, c planStatusCounts) []string {
	checkStyle := successTextStyleForTheme(theme)
	mutedStyle := mutedTextStyleForTheme(theme)

	line1 := checkStyle.Render("✓") + " " + mutedStyle.Render(fmt.Sprintf("Plan found: %d items", total))

	var chunks []string
	if c.Ready > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "READY").Render(fmt.Sprintf("%d ready", c.Ready)))
	}
	if c.Blocked > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "BLOCKED").Render(fmt.Sprintf("%d blocked", c.Blocked)))
	}
	if c.InProgress > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "IN_PROGRESS").Render(fmt.Sprintf("%d in progress", c.InProgress)))
	}
	if c.Queued > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "QUEUED").Render(fmt.Sprintf("%d queued", c.Queued)))
	}
	if c.WaitingUser > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "WAITING_USER").Render(fmt.Sprintf("%d waiting", c.WaitingUser)))
	}
	if c.Done > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "DONE").Render(fmt.Sprintf("%d done", c.Done)))
	}
	if c.Failed > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "FAILED").Render(fmt.Sprintf("%d failed", c.Failed)))
	}
	if c.Skipped > 0 {
		chunks = append(chunks, homeStatusStyle(theme, "SKIPPED").Render(fmt.Sprintf("%d skipped", c.Skipped)))
	}

	lines := []string{line1}
	if len(chunks) > 0 {
		lines = append(lines, "  "+strings.Join(chunks, "  "))
	}
	if c.LeafTotal > 0 {
		pct := 100 * c.LeafDone / c.LeafTotal
		lines = append(lines, "  "+mutedStyle.Render(fmt.Sprintf("%d%% complete (%d/%d leaf tasks)", pct, c.LeafDone, c.LeafTotal)))
	}
	return lines
}

// planOperationInProgress returns true when generating or refining a plan (home shows as no-plan, gray out actions).
func planOperationInProgress(m Model) bool {
	if !m.actionInProgress {
		return false
	}
	return m.actionName == "Generating plan..." || m.actionName == "Refining plan..."
}

func RenderHomeView(m Model) string {
	theme := themeOrDefault(m.theme)
	titleStyle := sectionHeaderStyleForTheme(theme)
	taglineStyle := mutedTextStyleForTheme(theme)
	mutedStyle := mutedTextStyleForTheme(theme)
	warnStyle := dangerTextStyleForTheme(theme)
	actionStyle := successTextStyleForTheme(theme)
	shortcutStyle := accentTextStyleForTheme(theme)
	errorStyle := errorBannerStyleForTheme(theme)

	// When generating/refining, show progress message and spinner; otherwise show plan status or no plan
	showPlanInfo := m.planExists && !planOperationInProgress(m)
	var statusLines []string
	if planOperationInProgress(m) {
		progressStyle := accentTextStyleForTheme(theme)
		frame := spinnerFrames[m.spinnerIndex%len(spinnerFrames)]
		statusLines = []string{
			progressStyle.Render(frame + " " + m.actionName),
		}
	} else if showPlanInfo {
		c := countPlanStatuses(m.plan)
		statusLines = formatPlanStatusLines(theme, len(m.plan.Items), c)
	} else {
		statusLines = []string{mutedStyle.Render("⊘ No plan found")}
	}

	lines := []string{
		titleStyle.Render("blackbird"),
		taglineStyle.Render("Durable, dependency-aware planning and execution"),
		"",
	}
	lines = append(lines, statusLines...)

	lines = append(lines, "", mutedStyle.Render(fmt.Sprintf("Agent: %s", agentLabel(m))))
	if m.agentSelectionErr != "" {
		lines = append(lines, warnStyle.Render(fmt.Sprintf("Agent config warning: %s", m.agentSelectionErr)))
	}

	if m.planValidationErr != "" && !planOperationInProgress(m) {
		lines = append(lines,
			"",
			errorStyle.Render(fmt.Sprintf("⚠ Plan has errors: %s\nPress [g] to regenerate or [v] to view and fix", m.planValidationErr)),
		)
	}

	// When generating/refining, gray out all options except quit
	inProgress := planOperationInProgress(m)
	lines = append(lines,
		"",
		renderActionLine("[g]", "Generate plan", !inProgress, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[v]", "View plan", showPlanInfo, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[r]", "Refine plan", showPlanInfo, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[e]", "Execute", m.canExecute() && !inProgress, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[s]", "Settings", !inProgress, shortcutStyle, actionStyle, mutedStyle),
		renderActionLine("[c]", "Change agent", !inProgress, shortcutStyle, actionStyle, mutedStyle),
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
