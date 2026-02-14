package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestUpdateParentReviewContinueClosesScreenWithoutStartingAction(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeNone)
	}
	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false")
	}
	if updated.actionName != "" {
		t.Fatalf("actionName = %q, want empty", updated.actionName)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared when continue closes screen")
	}
	if cmd != nil {
		t.Fatalf("expected no command for continue")
	}
}

func TestUpdateParentReviewResumeOneStartsAction(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeNone)
	}
	if !updated.actionInProgress {
		t.Fatalf("expected actionInProgress=true")
	}
	if updated.actionName != "Resuming..." {
		t.Fatalf("actionName = %q, want %q", updated.actionName, "Resuming...")
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared when resume starts")
	}
	if cmd == nil {
		t.Fatalf("expected resume-one command to be returned")
	}
}

func TestUpdateParentReviewResumeAllStartsAction(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeNone)
	}
	if !updated.actionInProgress {
		t.Fatalf("expected actionInProgress=true")
	}
	if updated.actionName != "Resuming..." {
		t.Fatalf("actionName = %q, want %q", updated.actionName, "Resuming...")
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared when resume starts")
	}
	if cmd == nil {
		t.Fatalf("expected resume-all command to be returned")
	}
}

func TestUpdateParentReviewQuitClosesScreenWithoutRestartingExecute(t *testing.T) {
	model := testParentReviewStateModel()
	model.resumeExecuteAfterParentReview = true

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v after quit", updated.actionMode, ActionModeNone)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared on quit")
	}
	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false on quit")
	}
	if cmd != nil {
		t.Fatalf("expected no command for quit")
	}
	if updated.resumeExecuteAfterParentReview {
		t.Fatalf("expected resumeExecuteAfterParentReview cleared on quit")
	}
}

func TestUpdateParentReviewQuitCancelsExecutingAction(t *testing.T) {
	model := testParentReviewStateModel()
	canceled := false
	model.actionInProgress = true
	model.actionName = "Executing..."
	model.actionCancel = func() { canceled = true }
	model.resumeExecuteAfterParentReview = true
	model.queuedParentReviewRuns = []execution.RunRecord{
		{ID: "queued-review-1", TaskID: "parent-queued"},
	}

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if cmd != nil {
		t.Fatalf("expected no command for quit while executing")
	}
	if !canceled {
		t.Fatalf("expected quit to cancel in-flight execute action")
	}
	if updated.actionCancel != nil {
		t.Fatalf("expected actionCancel cleared after quit cancel path")
	}
	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v after quit", updated.actionMode, ActionModeNone)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared on quit")
	}
	if updated.resumeExecuteAfterParentReview {
		t.Fatalf("expected resumeExecuteAfterParentReview cleared on quit")
	}
	if len(updated.queuedParentReviewRuns) != 0 {
		t.Fatalf("expected queued parent review runs cleared on quit")
	}
}

func testParentReviewStateModel() Model {
	now := time.Date(2026, 2, 9, 9, 30, 0, 0, time.UTC)
	passed := false
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"parent-1": {
				ID:        "parent-1",
				Title:     "Parent Review",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	run := execution.RunRecord{
		ID:                        "review-1",
		TaskID:                    "parent-1",
		StartedAt:                 now,
		Status:                    execution.RunStatusSuccess,
		ParentReviewPassed:        &passed,
		ParentReviewResumeTaskIDs: []string{"child-a", "child-b"},
		ParentReviewFeedback:      "retry children",
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "parent-1",
				Title: "Parent Review",
			},
			ParentReview: &execution.ParentReviewContext{
				ParentTaskID:    "parent-1",
				ParentTaskTitle: "Parent Review",
			},
		},
	}
	form := NewParentReviewForm(run, g)
	return Model{
		plan:             g,
		viewMode:         ViewModeMain,
		planExists:       true,
		actionMode:       ActionModeParentReview,
		parentReviewForm: &form,
		windowWidth:      120,
		windowHeight:     32,
	}
}
