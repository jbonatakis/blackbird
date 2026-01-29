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

	actions := actionHints(model, readyCount)
	left := strings.Join(actions, " ")

	if model.actionInProgress {
		frame := spinnerFrames[model.spinnerIndex%len(spinnerFrames)]
		left = fmt.Sprintf("%s | %s %s", left, frame, model.actionName)
	}

	right := fmt.Sprintf("ready:%d blocked:%d", readyCount, blockedCount)
	contentWidth := model.windowWidth
	padding := 1
	if contentWidth > 0 {
		contentWidth = contentWidth - padding*2
		if contentWidth < 0 {
			contentWidth = 0
		}
	}
	bar := layoutBar(left, right, contentWidth)

	style := lipgloss.NewStyle().Reverse(true).Padding(0, padding)
	return style.Render(bar)
}

func actionHints(model Model, readyCount int) []string {
	if model.actionInProgress {
		return []string{"[q]uit"}
	}
	actions := []string{
		"[g]enerate",
		"[r]efine",
		"[e]xecute",
		"[s]et-status",
		"[t]ab",
		"[f]ilter",
		"[q]uit",
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
	if model.actionMode == ActionModeSetStatus {
		actions = []string{"[q]uit"}
	}
	if model.actionMode == ActionModeGeneratePlan {
		actions = []string{"[enter]submit", "[esc]cancel", "[tab]next", "[shift+tab]prev", "[q]uit"}
	}
	if model.actionMode == ActionModeAgentQuestion {
		if model.agentQuestionForm != nil {
			currentQ := model.agentQuestionForm.CurrentQuestion()
			if len(currentQ.Options) > 0 {
				actions = []string{"[↑/↓]navigate", "[1-9]select", "[enter]confirm", "[esc]cancel", "[q]uit"}
			} else {
				actions = []string{"[enter]submit", "[esc]cancel", "[q]uit"}
			}
		}
	}
	if model.actionMode == ActionModePlanReview {
		if model.planReviewForm != nil && model.planReviewForm.mode == ReviewModeChooseAction {
			actions = []string{"[↑/↓]navigate", "[1-3]select", "[enter]confirm", "[esc]cancel", "[q]uit"}
		} else if model.planReviewForm != nil && model.planReviewForm.mode == ReviewModeRevisionPrompt {
			actions = []string{"[ctrl+s]submit", "[esc]back", "[q]uit"}
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
