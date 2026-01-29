package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// TestPlanReviewFormCreation tests that a new plan review form is created correctly
func TestPlanReviewFormCreation(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)

	if form.mode != ReviewModeChooseAction {
		t.Errorf("Expected initial mode to be ReviewModeChooseAction, got %v", form.mode)
	}

	if form.selectedAction != 0 {
		t.Errorf("Expected default selected action to be 0 (Accept), got %d", form.selectedAction)
	}

	if form.revisionCount != 0 {
		t.Errorf("Expected revision count to be 0, got %d", form.revisionCount)
	}

	if len(form.plan.Items) != len(testPlan.Items) {
		t.Errorf("Expected plan to have %d items, got %d", len(testPlan.Items), len(form.plan.Items))
	}
}

// TestPlanReviewNavigationUpDown tests navigation between actions
func TestPlanReviewNavigationUpDown(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)

	// Start at action 0 (Accept)
	if form.selectedAction != 0 {
		t.Errorf("Expected initial action to be 0, got %d", form.selectedAction)
	}

	// Press down - should move to action 1 (Revise)
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != 1 {
		t.Errorf("Expected action to be 1 after down, got %d", form.selectedAction)
	}

	// Press down again - should move to action 2 (Reject)
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != 2 {
		t.Errorf("Expected action to be 2 after second down, got %d", form.selectedAction)
	}

	// Press down at bottom - should stay at 2
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != 2 {
		t.Errorf("Expected action to stay at 2 at bottom, got %d", form.selectedAction)
	}

	// Press up - should move to action 1
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != 1 {
		t.Errorf("Expected action to be 1 after up, got %d", form.selectedAction)
	}

	// Press up again - should move to action 0
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != 0 {
		t.Errorf("Expected action to be 0 after second up, got %d", form.selectedAction)
	}

	// Press up at top - should stay at 0
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != 0 {
		t.Errorf("Expected action to stay at 0 at top, got %d", form.selectedAction)
	}
}

// TestPlanReviewQuickSelect tests number key quick selection
func TestPlanReviewQuickSelect(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)

	// Press 1 - select Accept
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if form.selectedAction != 0 {
		t.Errorf("Expected action to be 0 after pressing 1, got %d", form.selectedAction)
	}

	// Press 2 - select Revise
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if form.selectedAction != 1 {
		t.Errorf("Expected action to be 1 after pressing 2, got %d", form.selectedAction)
	}

	// Press 3 - select Reject
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if form.selectedAction != 2 {
		t.Errorf("Expected action to be 2 after pressing 3, got %d", form.selectedAction)
	}
}

// TestPlanReviewRevisionLimitBlocking tests that revision is blocked when limit reached
func TestPlanReviewRevisionLimitBlocking(t *testing.T) {
	testPlan := createTestPlan()
	// Create form with revision count at the limit
	form := NewPlanReviewForm(testPlan, maxGenerateRevisions)

	if form.CanRevise() {
		t.Error("Expected CanRevise to return false when at revision limit")
	}

	// Try to select Revise with number key - should not change selection
	form.selectedAction = 0
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if form.selectedAction != 0 {
		t.Errorf("Expected action to remain 0 when revision limit reached, got %d", form.selectedAction)
	}
}

// TestPlanReviewRevisionMode tests switching to revision prompt mode
func TestPlanReviewRevisionMode(t *testing.T) {
	testPlan := createTestPlan()
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 100
	m.windowHeight = 30

	// Create review form and set it on model
	form := NewPlanReviewForm(testPlan, 0)
	form.selectedAction = 1 // Select Revise
	m.planReviewForm = &form
	m.actionMode = ActionModePlanReview

	// Press enter to confirm Revise action
	m, _ = HandlePlanReviewKey(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.planReviewForm == nil {
		t.Fatal("Expected planReviewForm to still exist after switching to revision mode")
	}

	if m.planReviewForm.mode != ReviewModeRevisionPrompt {
		t.Errorf("Expected mode to be ReviewModeRevisionPrompt, got %v", m.planReviewForm.mode)
	}
}

// TestPlanReviewAccept tests accepting the plan
func TestPlanReviewAccept(t *testing.T) {
	testPlan := createTestPlan()
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 100
	m.windowHeight = 30

	// Create review form with Accept selected
	form := NewPlanReviewForm(testPlan, 0)
	form.selectedAction = 0 // Accept
	m.planReviewForm = &form
	m.actionMode = ActionModePlanReview

	// Press enter to confirm Accept action
	m, cmd := HandlePlanReviewKey(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.actionMode != ActionModeNone {
		t.Errorf("Expected actionMode to be ActionModeNone after accepting, got %v", m.actionMode)
	}

	if m.planReviewForm != nil {
		t.Error("Expected planReviewForm to be nil after accepting")
	}

	if cmd == nil {
		t.Error("Expected SavePlanCmd to be returned")
	}

	// Check that plan was updated
	if len(m.plan.Items) != len(testPlan.Items) {
		t.Errorf("Expected plan to have %d items, got %d", len(testPlan.Items), len(m.plan.Items))
	}
}

// TestPlanReviewReject tests rejecting the plan
func TestPlanReviewReject(t *testing.T) {
	originalPlan := createTestPlan()
	m := NewModel(originalPlan)
	m.windowWidth = 100
	m.windowHeight = 30

	newPlan := createTestPlanWithDifferentItems()

	// Create review form with Reject selected
	form := NewPlanReviewForm(newPlan, 0)
	form.selectedAction = 2 // Reject
	m.planReviewForm = &form
	m.actionMode = ActionModePlanReview

	// Press enter to confirm Reject action
	m, _ = HandlePlanReviewKey(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.actionMode != ActionModeNone {
		t.Errorf("Expected actionMode to be ActionModeNone after rejecting, got %v", m.actionMode)
	}

	if m.planReviewForm != nil {
		t.Error("Expected planReviewForm to be nil after rejecting")
	}

	// Check that plan was NOT updated (still has original)
	if len(m.plan.Items) != len(originalPlan.Items) {
		t.Errorf("Expected plan to still have %d items, got %d", len(originalPlan.Items), len(m.plan.Items))
	}
}

// TestPlanReviewRevisionPromptValidation tests that empty revision requests are not accepted
func TestPlanReviewRevisionPromptValidation(t *testing.T) {
	testPlan := createTestPlan()
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 100
	m.windowHeight = 30

	// Create review form in revision prompt mode
	form := NewPlanReviewForm(testPlan, 0)
	form.mode = ReviewModeRevisionPrompt
	m.planReviewForm = &form
	m.actionMode = ActionModePlanReview

	// Try to submit with empty textarea (ctrl+s)
	m, _ = HandlePlanReviewKey(m, tea.KeyMsg{Type: tea.KeyCtrlS})

	// Should remain in revision prompt mode
	if m.planReviewForm == nil || m.planReviewForm.mode != ReviewModeRevisionPrompt {
		t.Error("Expected to remain in revision prompt mode when textarea is empty")
	}
}

// TestPlanReviewEscapeFromRevisionPrompt tests going back from revision prompt to action selection
func TestPlanReviewEscapeFromRevisionPrompt(t *testing.T) {
	testPlan := createTestPlan()
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 100
	m.windowHeight = 30

	// Create review form in revision prompt mode
	form := NewPlanReviewForm(testPlan, 0)
	form.mode = ReviewModeRevisionPrompt
	m.planReviewForm = &form
	m.actionMode = ActionModePlanReview

	// Press escape to go back
	m, _ = HandlePlanReviewKey(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.planReviewForm == nil {
		t.Fatal("Expected planReviewForm to still exist after escape")
	}

	if m.planReviewForm.mode != ReviewModeChooseAction {
		t.Errorf("Expected mode to be ReviewModeChooseAction after escape, got %v", m.planReviewForm.mode)
	}
}

// Helper functions

func createTestPlan() plan.WorkGraph {
	now := time.Now().UTC()
	return plan.WorkGraph{
		SchemaVersion: 1,
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:                 "task-1",
				Title:              "Test Task 1",
				Description:        "Description 1",
				AcceptanceCriteria: []string{"Criterion 1"},
				Prompt:             "Prompt 1",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"task-2": {
				ID:                 "task-2",
				Title:              "Test Task 2",
				Description:        "Description 2",
				AcceptanceCriteria: []string{"Criterion 2"},
				Prompt:             "Prompt 2",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
}

func createTestPlanWithDifferentItems() plan.WorkGraph {
	now := time.Now().UTC()
	return plan.WorkGraph{
		SchemaVersion: 1,
		Items: map[string]plan.WorkItem{
			"task-3": {
				ID:                 "task-3",
				Title:              "Test Task 3",
				Description:        "Description 3",
				AcceptanceCriteria: []string{"Criterion 3"},
				Prompt:             "Prompt 3",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"task-4": {
				ID:                 "task-4",
				Title:              "Test Task 4",
				Description:        "Description 4",
				AcceptanceCriteria: []string{"Criterion 4"},
				Prompt:             "Prompt 4",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
}
