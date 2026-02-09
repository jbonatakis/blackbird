package tui

import (
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestExecuteActionCompleteParentReviewRequiredOpensModalAndResetsActionProgress(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {
					ID:    "parent-1",
					Title: "Parent review task",
				},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		windowWidth:      120,
		windowHeight:     32,
		actionInProgress: true,
		actionName:       "Executing...",
		actionCancel:     func() {},
		reviewCheckpointForm: &ReviewCheckpointForm{
			run: execution.RunRecord{ID: "stale-review"},
		},
		actionOutput: &ActionOutput{
			Message: "stale output",
			IsError: false,
		},
	}
	parentRun := testExecuteActionCompleteParentReviewRun("review-1", []string{"child-b", "child-a"})

	updatedModel, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonParentReviewRequired,
			TaskID: "parent-1",
			Run:    &parentRun,
		},
	})
	updated := updatedModel.(Model)

	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false after parent review stop")
	}
	if updated.actionName != "" {
		t.Fatalf("expected actionName to clear, got %q", updated.actionName)
	}
	if updated.actionCancel != nil {
		t.Fatalf("expected actionCancel to clear")
	}
	if updated.actionOutput != nil {
		t.Fatalf("expected actionOutput cleared when parent review modal opens")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("expected action mode %v, got %v", ActionModeParentReview, updated.actionMode)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parentReviewForm to be set")
	}
	if updated.parentReviewForm.run.ID != "review-1" {
		t.Fatalf("parentReviewForm.run.ID = %q, want review-1", updated.parentReviewForm.run.ID)
	}
	if got, want := updated.parentReviewForm.ResumeTargets(), []string{"child-a", "child-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resume targets = %#v, want %#v", got, want)
	}
	if updated.reviewCheckpointForm != nil {
		t.Fatalf("expected review checkpoint form to clear when parent review modal opens")
	}
}

func TestExecuteActionCompleteWaitingUserHandlingUnchanged(t *testing.T) {
	model := Model{
		actionInProgress: true,
		actionName:       "Executing...",
		actionCancel:     func() {},
	}

	updatedModel, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Output:  "task-7 is waiting for user input",
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonWaitingUser,
			TaskID: "task-7",
		},
	})
	updated := updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected action mode none for waiting_user, got %v", updated.actionMode)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parent review modal to remain closed for waiting_user")
	}
	if updated.reviewCheckpointForm != nil {
		t.Fatalf("expected review checkpoint modal to remain closed for waiting_user")
	}
	if updated.actionOutput == nil || updated.actionOutput.IsError {
		t.Fatalf("expected non-error action output for waiting_user")
	}
	if updated.actionOutput.Message != "task-7 is waiting for user input" {
		t.Fatalf("action output = %q", updated.actionOutput.Message)
	}
}

func TestExecuteActionCompleteDecisionRequiredHandlingUnchanged(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID:    "task-1",
					Title: "Task 1",
				},
			},
		},
		actionInProgress: true,
		actionName:       "Executing...",
		actionCancel:     func() {},
		parentReviewForm: &ParentReviewForm{
			run: execution.RunRecord{ID: "stale-parent"},
		},
	}
	decisionRun := execution.RunRecord{
		ID:               "run-decision",
		TaskID:           "task-1",
		Status:           execution.RunStatusSuccess,
		DecisionRequired: true,
		DecisionState:    execution.DecisionStatePending,
		Context: execution.ContextPack{
			Task: execution.TaskContext{ID: "task-1", Title: "Task 1"},
		},
	}

	updatedModel, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonDecisionRequired,
			TaskID: "task-1",
			Run:    &decisionRun,
		},
	})
	updated := updatedModel.(Model)

	if updated.actionMode != ActionModeReviewCheckpoint {
		t.Fatalf("expected review checkpoint modal for decision_required, got mode %v", updated.actionMode)
	}
	if updated.reviewCheckpointForm == nil {
		t.Fatalf("expected reviewCheckpointForm to be set")
	}
	if updated.reviewCheckpointForm.run.ID != "run-decision" {
		t.Fatalf("reviewCheckpointForm.run.ID = %q, want run-decision", updated.reviewCheckpointForm.run.ID)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected stale parent review form cleared when decision modal opens")
	}
	if updated.actionOutput != nil {
		t.Fatalf("expected no actionOutput when decision checkpoint modal opens")
	}
}

func TestExecuteActionCompleteParentReviewModalDismissClearsStaleData(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:     ViewModeMain,
		planExists:   true,
		windowWidth:  120,
		windowHeight: 32,
	}
	firstRun := testExecuteActionCompleteParentReviewRun("review-1", []string{"child-a"})
	secondRun := testExecuteActionCompleteParentReviewRun("review-2", []string{"child-z"})

	updatedModel, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonParentReviewRequired,
			Run:    &firstRun,
		},
	})
	updated := updatedModel.(Model)
	if updated.parentReviewForm == nil || updated.parentReviewForm.run.ID != "review-1" {
		t.Fatalf("expected first parent review modal state to open")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected action mode none after parent review dismiss, got %v", updated.actionMode)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parent review modal data cleared on dismiss")
	}

	updatedModel, _ = updated.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonParentReviewRequired,
			Run:    &secondRun,
		},
	})
	updated = updatedModel.(Model)
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review modal to reopen")
	}
	if updated.parentReviewForm.run.ID != "review-2" {
		t.Fatalf("expected reopened modal to use fresh run data, got %q", updated.parentReviewForm.run.ID)
	}
	if got, want := updated.parentReviewForm.ResumeTargets(), []string{"child-z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resume targets after reopen = %#v, want %#v", got, want)
	}
}

func TestExecuteActionCompleteParentReviewModalResumeSelectedStartsAction(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:     ViewModeMain,
		planExists:   true,
		windowWidth:  120,
		windowHeight: 32,
	}
	parentRun := testExecuteActionCompleteParentReviewRun("review-1", []string{"child-a", "child-b"})

	updatedModel, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonParentReviewRequired,
			Run:    &parentRun,
		},
	})
	updated := updatedModel.(Model)
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("expected parent review modal to open, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review form to be set")
	}

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v after resume start", updated.actionMode, ActionModeNone)
	}
	if !updated.actionInProgress {
		t.Fatalf("expected actionInProgress=true after parent-review resume key")
	}
	if updated.actionName != "Resuming..." {
		t.Fatalf("actionName = %q, want %q", updated.actionName, "Resuming...")
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parent review form to clear when resume starts")
	}
	if cmd == nil {
		t.Fatalf("expected resume command to be returned")
	}
}

func TestExecuteActionCompleteParentReviewModalPausesNormalExecuteShortcut(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-ready": {
					ID:     "task-ready",
					Title:  "Ready task",
					Status: plan.StatusTodo,
				},
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		selectedID:       "task-ready",
		viewMode:         ViewModeMain,
		planExists:       true,
		windowWidth:      120,
		windowHeight:     32,
		actionInProgress: false,
	}
	parentRun := testExecuteActionCompleteParentReviewRun("review-1", []string{"child-a", "child-b"})

	updatedModel, _ := model.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonParentReviewRequired,
			Run:    &parentRun,
		},
	})
	updated := updatedModel.(Model)
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("expected parent review modal open before key test, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review form")
	}
	initialTarget := updated.parentReviewForm.SelectedTarget()

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("expected parent review modal to remain active after shortcut key")
	}
	if updated.actionInProgress {
		t.Fatalf("expected execute shortcut to be paused while parent modal is open")
	}
	if updated.actionName != "" {
		t.Fatalf("expected no action name change while modal active, got %q", updated.actionName)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review form to stay mounted")
	}
	if updated.parentReviewForm.SelectedTarget() != initialTarget {
		t.Fatalf("expected execute shortcut key not to mutate parent-review selection")
	}
}

func testExecuteActionCompleteParentReviewRun(runID string, resumeTargets []string) execution.RunRecord {
	passed := false
	now := time.Date(2026, 2, 9, 8, 0, 0, 0, time.UTC)
	return execution.RunRecord{
		ID:                        runID,
		TaskID:                    "parent-1",
		StartedAt:                 now,
		Status:                    execution.RunStatusSuccess,
		ParentReviewPassed:        &passed,
		ParentReviewResumeTaskIDs: append([]string{}, resumeTargets...),
		ParentReviewFeedback:      "Fix parent review findings and retry.",
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "parent-1",
				Title: "Parent",
			},
			ParentReview: &execution.ParentReviewContext{
				ParentTaskID:    "parent-1",
				ParentTaskTitle: "Parent",
			},
		},
	}
}
