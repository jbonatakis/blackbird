package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestParentReviewRunMsgOpensModalImmediatelyForPassedReview(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		windowWidth:      120,
		windowHeight:     32,
	}

	run := testExecuteActionCompleteParentReviewPassRun("review-pass-live")
	updatedModel, _ := model.Update(parentReviewRunMsg{run: run})
	updated := updatedModel.(Model)

	if !updated.actionInProgress {
		t.Fatalf("expected execute action to remain in progress while live review modal is shown")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeParentReview)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review form to open")
	}
	if updated.parentReviewForm.run.ID != "review-pass-live" {
		t.Fatalf("parentReviewForm.run.ID = %q, want %q", updated.parentReviewForm.run.ID, "review-pass-live")
	}
	if updated.parentReviewForm.HasFailedTasks() {
		t.Fatalf("expected passed review modal to have no failed tasks")
	}
}

func TestParentReviewRunMsgOpensModalImmediatelyForFailedReview(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		windowWidth:      120,
		windowHeight:     32,
	}

	run := testExecuteActionCompleteParentReviewRun("review-fail-live", []string{"child-a"})
	updatedModel, _ := model.Update(parentReviewRunMsg{run: run})
	updated := updatedModel.(Model)

	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeParentReview)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review form to open")
	}
	if updated.parentReviewForm.run.ID != "review-fail-live" {
		t.Fatalf("parentReviewForm.run.ID = %q, want %q", updated.parentReviewForm.run.ID, "review-fail-live")
	}
	if !updated.parentReviewForm.HasFailedTasks() {
		t.Fatalf("expected failed review modal to include failed tasks")
	}
}

func TestParentReviewRunMsgQueuesAndShowsNextModalOnDismiss(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		windowWidth:      120,
		windowHeight:     32,
	}

	first := testExecuteActionCompleteParentReviewPassRun("review-first")
	second := testExecuteActionCompleteParentReviewRun("review-second", []string{"child-a"})

	updatedModel, _ := model.Update(parentReviewRunMsg{run: first})
	updated := updatedModel.(Model)
	if updated.parentReviewForm == nil || updated.parentReviewForm.run.ID != "review-first" {
		t.Fatalf("expected first modal to open")
	}

	updatedModel, _ = updated.Update(parentReviewRunMsg{run: second})
	updated = updatedModel.(Model)
	if updated.parentReviewForm == nil || updated.parentReviewForm.run.ID != "review-first" {
		t.Fatalf("expected current modal to remain first while second is queued")
	}
	if len(updated.queuedParentReviewRuns) != 1 {
		t.Fatalf("queued parent review count = %d, want 1", len(updated.queuedParentReviewRuns))
	}
	if updated.queuedParentReviewRuns[0].ID != "review-second" {
		t.Fatalf("queued run ID = %q, want %q", updated.queuedParentReviewRuns[0].ID, "review-second")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("expected next queued parent review modal to open, got action mode %v", updated.actionMode)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected second parent review modal to open")
	}
	if updated.parentReviewForm.run.ID != "review-second" {
		t.Fatalf("parentReviewForm.run.ID = %q, want %q", updated.parentReviewForm.run.ID, "review-second")
	}
	if len(updated.queuedParentReviewRuns) != 0 {
		t.Fatalf("expected queue to drain after opening second modal, got %d", len(updated.queuedParentReviewRuns))
	}
}

func TestExecuteActionCompleteParentReviewFallbackDoesNotDuplicateLiveModal(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		windowWidth:      120,
		windowHeight:     32,
	}

	run := testExecuteActionCompleteParentReviewPassRun("review-live-1")
	updatedModel, _ := model.Update(parentReviewRunMsg{run: run})
	updated := updatedModel.(Model)
	if updated.parentReviewForm == nil || updated.parentReviewForm.run.ID != "review-live-1" {
		t.Fatalf("expected live parent review modal to open")
	}

	updatedModel, _ = updated.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonCompleted,
			Run:    &run,
		},
	})
	updated = updatedModel.(Model)

	if updated.parentReviewForm == nil || updated.parentReviewForm.run.ID != "review-live-1" {
		t.Fatalf("expected same live modal to remain without duplicate reopen")
	}
	if len(updated.queuedParentReviewRuns) != 0 {
		t.Fatalf("expected no duplicate queued parent reviews, got %d", len(updated.queuedParentReviewRuns))
	}
}

func TestExecuteActionCompleteParentReviewFallbackDoesNotReopenAfterLiveDismiss(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		windowWidth:      120,
		windowHeight:     32,
	}

	run := testExecuteActionCompleteParentReviewPassRun("review-live-dismissed")
	updatedModel, _ := model.Update(parentReviewRunMsg{run: run})
	updated := updatedModel.(Model)
	if updated.parentReviewForm == nil || updated.parentReviewForm.run.ID != "review-live-dismissed" {
		t.Fatalf("expected live parent review modal to open")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected modal dismissed before execute completion, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared after live dismiss")
	}

	updatedModel, _ = updated.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonCompleted,
			Run:    &run,
		},
	})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected no duplicate modal reopen after execute completion, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parent review modal to remain closed after fallback result")
	}
}

func TestParentReviewContinueWhileExecutingSignalsAck(t *testing.T) {
	ackCh := make(chan struct{}, 1)
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"parent-1": {ID: "parent-1", Title: "Parent"},
			},
		},
		viewMode:            ViewModeMain,
		planExists:          true,
		actionInProgress:    true,
		actionName:          "Executing...",
		windowWidth:         120,
		windowHeight:        32,
		parentReviewAckChan: ackCh,
	}

	run := testExecuteActionCompleteParentReviewPassRun("review-pass-ack")
	updatedModel, _ := model.Update(parentReviewRunMsg{run: run})
	updated := updatedModel.(Model)
	if updated.parentReviewForm == nil {
		t.Fatalf("expected live parent review modal to open")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected modal to close on continue, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared on continue")
	}

	select {
	case <-ackCh:
		// expected
	default:
		t.Fatalf("expected continue action to signal parent review ack while executing")
	}
}

func TestStopAfterEachTaskDefersParentReviewUntilAfterCheckpointContinue(t *testing.T) {
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"child-1": {
					ID:    "child-1",
					Title: "Child 1",
				},
				"parent-1": {
					ID:    "parent-1",
					Title: "Parent 1",
				},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		windowWidth:      120,
		windowHeight:     32,
		config: config.ResolvedConfig{
			Execution: config.ResolvedExecution{
				StopAfterEachTask: true,
			},
		},
	}

	reviewRun := testExecuteActionCompleteParentReviewPassRun("review-deferred")
	updatedModel, _ := model.Update(parentReviewRunMsg{run: reviewRun})
	updated := updatedModel.(Model)
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected parent review to remain deferred while execute is in progress, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected no parent review modal before decision checkpoint")
	}
	if len(updated.queuedParentReviewRuns) != 1 {
		t.Fatalf("queued parent reviews = %d, want 1", len(updated.queuedParentReviewRuns))
	}

	decisionRun := execution.RunRecord{
		ID:               "run-decision-1",
		TaskID:           "child-1",
		Status:           execution.RunStatusSuccess,
		DecisionRequired: true,
		DecisionState:    execution.DecisionStatePending,
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "child-1",
				Title: "Child 1",
			},
		},
	}
	updatedModel, _ = updated.Update(ExecuteActionComplete{
		Action:  "execute",
		Success: true,
		Result: &execution.ExecuteResult{
			Reason: execution.ExecuteReasonDecisionRequired,
			TaskID: "child-1",
			Run:    &decisionRun,
		},
	})
	updated = updatedModel.(Model)
	if updated.actionMode != ActionModeReviewCheckpoint {
		t.Fatalf("expected review checkpoint first, got mode %v", updated.actionMode)
	}
	if updated.reviewCheckpointForm == nil {
		t.Fatalf("expected review checkpoint modal to be open")
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parent review modal to remain deferred while checkpoint is active")
	}

	// Decision completions are delivered after the checkpoint modal has been
	// closed and decision action has been started.
	updated.actionMode = ActionModeNone
	updated.reviewCheckpointForm = nil
	updated.actionInProgress = true
	updated.actionName = "Recording decision..."

	updatedModel, cmd := updated.Update(DecisionActionComplete{
		Action: execution.DecisionStateApprovedContinue,
		Result: execution.DecisionResult{
			Action:   execution.DecisionStateApprovedContinue,
			Continue: true,
		},
	})
	updated = updatedModel.(Model)
	if cmd != nil {
		t.Fatalf("expected no execute command until deferred parent review is resolved")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("expected deferred parent review modal after checkpoint continue, got mode %v", updated.actionMode)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parent review modal to open after checkpoint continue")
	}
	if updated.parentReviewForm.run.ID != "review-deferred" {
		t.Fatalf("parentReviewForm.run.ID = %q, want %q", updated.parentReviewForm.run.ID, "review-deferred")
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if !updated.actionInProgress {
		t.Fatalf("expected execute to restart after deferred parent review continue")
	}
	if updated.actionName != "Executing..." {
		t.Fatalf("actionName = %q, want %q", updated.actionName, "Executing...")
	}
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected parent review modal to close before execute restart, got mode %v", updated.actionMode)
	}
	if cmd == nil {
		t.Fatalf("expected execute command to be started after deferred parent review continue")
	}
}
