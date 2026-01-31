package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestUpdateQuitCommand(t *testing.T) {
	model := Model{}

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit command to return tea.QuitMsg")
	}
}

func TestUpdateQuitCancelsAction(t *testing.T) {
	cancelCalled := false
	model := Model{
		actionInProgress: true,
		actionCancel: func() {
			cancelCalled = true
		},
	}

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !cancelCalled {
		t.Fatalf("expected cancel func to be invoked on quit")
	}
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit command to return tea.QuitMsg")
	}
}

func TestExecuteActionCompleteClearsInProgress(t *testing.T) {
	model := Model{
		actionInProgress: true,
		actionName:       "Executing...",
		actionCancel:     func() {},
	}

	updated, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Output:  "no ready tasks remaining",
	})
	next := updated.(Model)
	if next.actionInProgress {
		t.Fatalf("expected actionInProgress to be false after completion")
	}
	if next.actionCancel != nil {
		t.Fatalf("expected actionCancel to be cleared after completion")
	}
	if next.actionOutput == nil || next.actionOutput.IsError {
		t.Fatalf("expected non-error action output")
	}
}

func TestWindowSizeMsgUpdatesDimensions(t *testing.T) {
	model := Model{}
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	updated, _ := model.Update(msg)
	updatedModel := updated.(Model)

	if updatedModel.windowWidth != 120 {
		t.Fatalf("expected width 120, got %d", updatedModel.windowWidth)
	}
	if updatedModel.windowHeight != 40 {
		t.Fatalf("expected height 40, got %d", updatedModel.windowHeight)
	}
}

func TestViewRendersPlaceholderText(t *testing.T) {
	// Need windowHeight >= 6 so availableHeight (windowHeight-5) >= 1 for content
	model := Model{windowHeight: 6}
	// Default view is Home; empty plan shows home screen with "No plan found"
	view := model.View()
	if !strings.Contains(view, "No plan found") {
		t.Fatalf("expected home view 'No plan found' for empty plan, got %q", view)
	}
}

func TestHasPlanAndCanExecute(t *testing.T) {
	readyPlan := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:     "task-1",
				Status: plan.StatusTodo,
			},
		},
	}

	model := Model{
		plan:       readyPlan,
		planExists: true,
	}

	if !model.hasPlan() {
		t.Fatalf("expected hasPlan to be true")
	}
	if !model.canExecute() {
		t.Fatalf("expected canExecute to be true with ready tasks and plan exists")
	}

	model.planExists = false
	if model.hasPlan() {
		t.Fatalf("expected hasPlan to be false when planExists is false")
	}
	if model.canExecute() {
		t.Fatalf("expected canExecute to be false when planExists is false")
	}

	model.planExists = true
	model.plan = plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:     "task-1",
				Status: plan.StatusDone,
			},
		},
	}
	if model.canExecute() {
		t.Fatalf("expected canExecute to be false when no ready tasks")
	}
}

func TestPlanDataLoadedErrorUpdatesState(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			Items: map[string]plan.WorkItem{
				"task-1": {ID: "task-1"},
			},
		},
		planExists: true,
	}

	msg := PlanDataLoaded{
		Plan:       plan.NewEmptyWorkGraph(),
		PlanExists: true,
		Err:        errors.New("invalid plan"),
	}

	updated, _ := model.Update(msg)
	updatedModel := updated.(Model)

	if !updatedModel.planExists {
		t.Fatalf("expected planExists to remain true on validation error")
	}
	if len(updatedModel.plan.Items) != 0 {
		t.Fatalf("expected plan to be reset to empty graph on error")
	}
	if updatedModel.actionOutput == nil || !updatedModel.actionOutput.IsError {
		t.Fatalf("expected actionOutput error to be set")
	}
	if !strings.Contains(updatedModel.actionOutput.Message, "invalid plan") {
		t.Fatalf("expected actionOutput to contain error message, got %q", updatedModel.actionOutput.Message)
	}
}

func TestPlanDataLoadedMissingPlanKeepsPlanExistsFalse(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			Items: map[string]plan.WorkItem{
				"task-1": {ID: "task-1"},
			},
		},
		planExists: true,
	}

	msg := PlanDataLoaded{
		Plan:       plan.NewEmptyWorkGraph(),
		PlanExists: false,
		Err:        nil,
	}

	updated, _ := model.Update(msg)
	updatedModel := updated.(Model)

	if updatedModel.planExists {
		t.Fatalf("expected planExists to be false when plan is missing")
	}
	if len(updatedModel.plan.Items) != 0 {
		t.Fatalf("expected empty plan when missing, got %d items", len(updatedModel.plan.Items))
	}
	if updatedModel.actionOutput != nil {
		t.Fatalf("expected no action output when plan is missing without error")
	}
	if updatedModel.planValidationErr != "" {
		t.Fatalf("expected empty validation error, got %q", updatedModel.planValidationErr)
	}
}
