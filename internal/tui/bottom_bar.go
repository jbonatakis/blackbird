package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

var spinnerFrames = []string{"|", "/", "-", "\\"}

func RenderBottomBar(model Model) string {
	readyCount := len(execution.ReadyTasks(model.plan))
	blockedCount := blockedCount(model.plan)
	agent := agentLabel(model)

	actions := actionHints(model, readyCount)
	contentWidth, padding := contentWidth(model.windowWidth)
	if shouldTrimActions(model, contentWidth, actions, agent, readyCount, blockedCount) {
		actions = trimBottomBarActions(actions, contentWidth, agent, readyCount, blockedCount)
	}
	left := strings.Join(actions, " ")
	if model.actionInProgress {
		frame := spinnerFrames[model.spinnerIndex%len(spinnerFrames)]
		left = fmt.Sprintf("%s | %s %s", left, frame, bottomBarActionText(model))
	}
	right := selectBottomBarRight(model, agent, readyCount, blockedCount, left, contentWidth)
	bar := layoutBar(left, right, contentWidth)

	style := lipgloss.NewStyle().Reverse(true).Padding(0, padding)
	return style.Render(bar)
}

func bottomBarActionText(model Model) string {
	if model.executionState.Stage == execution.ExecutionStageReviewing {
		return "Reviewing..."
	}
	return model.actionName
}

func selectBottomBarRight(model Model, agent string, readyCount int, blockedCount int, left string, width int) string {
	if model.viewMode == ViewModeHome && !model.planExists {
		return fmt.Sprintf("agent:%s", agent)
	}
	full := fmt.Sprintf("agent:%s | ready:%d blocked:%d", agent, readyCount, blockedCount)
	compact := fmt.Sprintf("agent:%s r:%d b:%d", agent, readyCount, blockedCount)
	minimal := fmt.Sprintf("agent:%s", agent)
	if width <= 0 {
		return full
	}
	leftWidth := lipgloss.Width(left)
	if leftWidth+1+lipgloss.Width(full) <= width {
		return full
	}
	if leftWidth+1+lipgloss.Width(compact) <= width {
		return compact
	}
	return minimal
}

func shouldTrimActions(model Model, width int, actions []string, agent string, readyCount int, blockedCount int) bool {
	if width <= 0 || model.viewMode != ViewModeMain || model.actionMode != ActionModeNone || model.actionInProgress {
		return false
	}
	left := strings.Join(actions, " ")
	compact := fmt.Sprintf("agent:%s r:%d b:%d", agent, readyCount, blockedCount)
	return lipgloss.Width(left)+1+lipgloss.Width(compact) > width
}

func trimBottomBarActions(actions []string, width int, agent string, readyCount int, blockedCount int) []string {
	priorities := []string{
		"[f]ilter",
		"[t]ab",
		"[s]et-status",
		"[u]resume",
		"[e]xecute",
		"[r]efine",
		"[g]enerate",
		"[c]hange",
		"[h]ome",
	}
	compact := fmt.Sprintf("agent:%s r:%d b:%d", agent, readyCount, blockedCount)
	for _, remove := range priorities {
		if lipgloss.Width(strings.Join(actions, " "))+1+lipgloss.Width(compact) <= width {
			break
		}
		actions = removeAction(actions, remove)
	}
	return actions
}

func contentWidth(windowWidth int) (int, int) {
	padding := 1
	width := windowWidth
	if width > 0 {
		width = width - padding*2
		if width < 0 {
			width = 0
		}
	}
	return width, padding
}

func actionHints(model Model, readyCount int) []string {
	var actions []string
	if model.actionMode == ActionModeSetStatus {
		return []string{"[ctrl+c]quit"}
	}
	if model.actionMode == ActionModeGeneratePlan {
		return []string{"[esc]cancel", "[tab]next", "[shift+tab]prev", "[ctrl+c]quit"}
	}
	if model.actionMode == ActionModePlanRefine {
		return []string{"[esc]cancel", "[tab]next", "[shift+tab]prev", "[ctrl+c]quit"}
	}
	if model.actionMode == ActionModeAgentQuestion {
		if model.agentQuestionForm != nil {
			currentQ := model.agentQuestionForm.CurrentQuestion()
			if len(currentQ.Options) > 0 {
				return []string{"[↑/↓]navigate", "[1-9]select", "[enter]confirm", "[esc]cancel", "[ctrl+c]quit"}
			}
			return []string{"[enter]submit", "[esc]cancel", "[ctrl+c]quit"}
		}
	}
	if model.actionMode == ActionModePlanReview {
		if model.planReviewForm != nil && model.planReviewForm.mode == ReviewModeChooseAction {
			return []string{"[↑/↓]navigate", "[1-3]select", "[enter]confirm", "[esc]cancel", "[ctrl+c]quit"}
		} else if model.planReviewForm != nil && model.planReviewForm.mode == ReviewModeRevisionPrompt {
			return []string{"[ctrl+s]submit", "[esc]back", "[ctrl+c]quit"}
		}
	}
	if model.actionMode == ActionModeReviewCheckpoint {
		if model.reviewCheckpointForm != nil && model.reviewCheckpointForm.mode == ReviewCheckpointChooseAction {
			return []string{"[↑/↓]navigate", "[1-4]select", "[enter]confirm", "[ctrl+c]quit"}
		} else if model.reviewCheckpointForm != nil && model.reviewCheckpointForm.mode == ReviewCheckpointRequestChanges {
			return []string{"[ctrl+s]submit", "[esc]back", "[ctrl+c]quit"}
		}
	}
	if model.actionMode == ActionModeParentReview {
		if model.parentReviewForm != nil && model.parentReviewForm.Mode() == ParentReviewModalModeConfirmDiscard {
			return []string{"[↑/↓]navigate", "[1-2]select", "[enter]confirm", "[esc]back", "[ctrl+c]quit"}
		}
		return []string{"[↑/↓]navigate", "[1-4]select", "[enter]confirm", "[esc]back", "[ctrl+c]quit"}
	}
	if model.actionMode == ActionModeSelectAgent {
		return []string{"[↑/↓]move", "[enter]select", "[esc]cancel", "[ctrl+c]quit"}
	}
	if model.actionInProgress {
		return []string{"[ctrl+c]quit"}
	}
	if model.viewMode == ViewModeSettings {
		return []string{"[esc]back", "[h]ome", "[ctrl+c]quit"}
	}
	if model.viewMode == ViewModeHome {
		actions = []string{
			"[g]enerate",
			"[v]iew",
			"[r]efine",
			"[e]xecute",
			"[s]ettings",
			"[c]hange",
			"[ctrl+c]quit",
		}
		if agentIsFromEnv() {
			actions = removeAction(actions, "[c]hange")
		}
		if !model.planExists {
			actions = removeAction(actions, "[v]iew")
			actions = removeAction(actions, "[r]efine")
		}
		if !model.canExecute() {
			actions = removeAction(actions, "[e]xecute")
		}
		return actions
	}
	actions = []string{
		"[h]ome",
		"[g]enerate",
		"[r]efine",
		"[e]xecute",
		"[s]et-status",
		"[t]ab",
		"[f]ilter",
		"[ctrl+c]quit",
	}
	if readyCount == 0 {
		actions = removeAction(actions, "[e]xecute")
	}
	if model.selectedID == "" {
		actions = removeAction(actions, "[s]et-status")
	}
	if CanResume(model) {
		// Insert resume action after execute
		idx := -1
		for i, action := range actions {
			if action == "[e]xecute" {
				idx = i + 1
				break
			}
		}
		if idx == -1 {
			// If execute not found, insert before set-status
			for i, action := range actions {
				if action == "[s]et-status" {
					idx = i
					break
				}
			}
		}
		if idx > 0 && idx <= len(actions) {
			actions = append(actions[:idx], append([]string{"[u]resume"}, actions[idx:]...)...)
		} else {
			actions = append([]string{"[u]resume"}, actions...)
		}
	}
	return actions
}

func removeAction(actions []string, remove string) []string {
	filtered := make([]string, 0, len(actions))
	for _, action := range actions {
		if action == remove {
			continue
		}
		filtered = append(filtered, action)
	}
	return filtered
}

func blockedCount(g plan.WorkGraph) int {
	count := 0
	for _, it := range g.Items {
		if it.Status == plan.StatusBlocked {
			count++
			continue
		}
		if it.Status == plan.StatusTodo && len(plan.UnmetDeps(g, it)) != 0 {
			count++
		}
	}
	return count
}

func layoutBar(left string, right string, width int) string {
	if width <= 0 {
		return left + " " + right
	}
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		availableLeft := width - rightWidth - 1
		if availableLeft < 0 {
			return truncate(right, width)
		}
		left = truncate(left, availableLeft)
		leftWidth = lipgloss.Width(left)
		gap = width - leftWidth - rightWidth
		if gap < 1 {
			gap = 1
		}
	}
	bar := left + strings.Repeat(" ", gap) + right
	return truncate(bar, width)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width])
}
