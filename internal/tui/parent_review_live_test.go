package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
