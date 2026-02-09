package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestUpdateParentReviewResumeSelectedStartsAction(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)

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
		t.Fatalf("expected resume command to be returned")
	}
}

func TestUpdateParentReviewResumeAllStartsAction(t *testing.T) {
	model := testParentReviewStateModel()

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated := updatedModel.(Model)

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
