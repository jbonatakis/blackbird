package tui

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestUpdateRunDataLoadedReplacesStateAndOpensReviewCheckpoint(t *testing.T) {
	now := time.Date(2026, 2, 9, 3, 0, 0, 0, time.UTC)
	decisionRun := execution.RunRecord{
		ID:               "run-new",
		TaskID:           "task-1",
		StartedAt:        now,
		Status:           execution.RunStatusSuccess,
		DecisionRequired: true,
		DecisionState:    execution.DecisionStatePending,
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "task-1",
				Title: "Task 1",
			},
		},
	}
	pendingFeedback := execution.PendingParentReviewFeedback{
		ParentTaskID: "parent-1",
		ReviewRunID:  "review-1",
		Feedback:     "address reviewer notes",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {ID: "task-1"},
			},
		},
		runData: map[string]execution.RunRecord{
			"task-old": {ID: "run-old", TaskID: "task-old"},
		},
		pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{
			"task-old": {ParentTaskID: "parent-old"},
		},
		actionMode: ActionModeNone,
	}

	updatedModel, _ := model.Update(RunDataLoaded{
		Data: map[string]execution.RunRecord{
			"task-1": decisionRun,
		},
		PendingParentFeedback: map[string]execution.PendingParentReviewFeedback{
			"task-1": pendingFeedback,
		},
	})
	updated := updatedModel.(Model)

	if got := updated.runData["task-1"].ID; got != "run-new" {
		t.Fatalf("runData[task-1].ID = %q, want run-new", got)
	}
	if _, ok := updated.runData["task-old"]; ok {
		t.Fatalf("expected runData to replace prior entries")
	}
	if !reflect.DeepEqual(updated.pendingParentFeedback, map[string]execution.PendingParentReviewFeedback{
		"task-1": pendingFeedback,
	}) {
		t.Fatalf("unexpected pending feedback state: %#v", updated.pendingParentFeedback)
	}
	if updated.actionMode != ActionModeReviewCheckpoint {
		t.Fatalf("expected review checkpoint modal to open, got mode %v", updated.actionMode)
	}
	if updated.reviewCheckpointForm == nil {
		t.Fatalf("expected review checkpoint form to be set")
	}
	if updated.reviewCheckpointForm.run.ID != "run-new" {
		t.Fatalf("review checkpoint run id = %q, want run-new", updated.reviewCheckpointForm.run.ID)
	}
}

func TestUpdateRunDataLoadedErrorPreservesExistingState(t *testing.T) {
	now := time.Date(2026, 2, 9, 3, 30, 0, 0, time.UTC)
	existingRunData := map[string]execution.RunRecord{
		"task-1": {
			ID:        "run-1",
			TaskID:    "task-1",
			StartedAt: now,
			Status:    execution.RunStatusRunning,
		},
	}
	existingPending := map[string]execution.PendingParentReviewFeedback{
		"task-1": {
			ParentTaskID: "parent-1",
			ReviewRunID:  "review-1",
			Feedback:     "feedback",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	model := Model{
		runData:               existingRunData,
		pendingParentFeedback: existingPending,
	}

	updatedModel, _ := model.Update(RunDataLoaded{
		Data: map[string]execution.RunRecord{
			"task-2": {ID: "run-2", TaskID: "task-2"},
		},
		PendingParentFeedback: map[string]execution.PendingParentReviewFeedback{
			"task-2": {ParentTaskID: "parent-2"},
		},
		Err: errors.New("load failed"),
	})
	updated := updatedModel.(Model)

	if !reflect.DeepEqual(updated.runData, existingRunData) {
		t.Fatalf("runData changed on load error: got %#v want %#v", updated.runData, existingRunData)
	}
	if !reflect.DeepEqual(updated.pendingParentFeedback, existingPending) {
		t.Fatalf("pending feedback changed on load error: got %#v want %#v", updated.pendingParentFeedback, existingPending)
	}
}

func TestUpdateRunDataLoadedClearsReviewCheckpointWhenNoPendingDecision(t *testing.T) {
	now := time.Date(2026, 2, 9, 4, 0, 0, 0, time.UTC)
	initialRun := execution.RunRecord{
		ID:               "run-pending",
		TaskID:           "task-1",
		StartedAt:        now,
		Status:           execution.RunStatusSuccess,
		DecisionRequired: true,
		DecisionState:    execution.DecisionStatePending,
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID: "task-1",
			},
		},
	}
	form := NewReviewCheckpointForm(initialRun, plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": {ID: "task-1"},
		},
	})

	model := Model{
		actionMode:           ActionModeReviewCheckpoint,
		reviewCheckpointForm: &form,
	}

	updatedModel, _ := model.Update(RunDataLoaded{
		Data:                  map[string]execution.RunRecord{},
		PendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
	})
	updated := updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected action mode reset to none, got %v", updated.actionMode)
	}
	if updated.reviewCheckpointForm != nil {
		t.Fatalf("expected review checkpoint form to close")
	}
}
