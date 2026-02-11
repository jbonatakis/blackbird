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

func TestUpdateParentReviewDiscardRequiresConfirmationAndCancelReturnsToScreen(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if cmd != nil {
		t.Fatalf("expected no command while opening discard confirmation")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("actionMode = %v, want %v while confirming discard", updated.actionMode, ActionModeParentReview)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parentReviewForm to remain active during discard confirmation")
	}
	if updated.parentReviewForm.Mode() != ParentReviewModalModeConfirmDiscard {
		t.Fatalf(
			"parentReviewForm.Mode() = %v, want %v",
			updated.parentReviewForm.Mode(),
			ParentReviewModalModeConfirmDiscard,
		)
	}
	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false while confirming discard")
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd != nil {
		t.Fatalf("expected no command when canceling discard confirmation")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("actionMode = %v, want %v after discard cancel", updated.actionMode, ActionModeParentReview)
	}
	if updated.parentReviewForm == nil {
		t.Fatalf("expected parentReviewForm to remain after discard cancel")
	}
	if updated.parentReviewForm.Mode() != ParentReviewModalModeActions {
		t.Fatalf("mode after discard cancel = %v, want actions", updated.parentReviewForm.Mode())
	}
}

func TestUpdateParentReviewDiscardConfirmClosesScreen(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated := updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)

	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v after discard confirm", updated.actionMode, ActionModeNone)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared on discard confirm")
	}
	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false on discard confirm")
	}
	if cmd != nil {
		t.Fatalf("expected no command for discard confirm")
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
