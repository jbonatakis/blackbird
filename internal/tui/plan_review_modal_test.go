package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
	"github.com/jbonatakis/blackbird/internal/planquality"
)

// TestPlanReviewFormCreation tests that a new plan review form is created correctly
func TestPlanReviewFormCreation(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)

	if form.mode != ReviewModeChooseAction {
		t.Errorf("Expected initial mode to be ReviewModeChooseAction, got %v", form.mode)
	}

	if form.HasBlockingFindings() && form.selectedAction == planReviewActionAccept {
		t.Errorf("Expected non-accept default when blocking findings remain, got %d", form.selectedAction)
	}

	if !form.HasBlockingFindings() && form.selectedAction != planReviewActionAccept {
		t.Errorf("Expected accept default when no blocking findings remain, got %d", form.selectedAction)
	}

	if form.revisionCount != 0 {
		t.Errorf("Expected revision count to be 0, got %d", form.revisionCount)
	}

	if len(form.plan.Items) != len(testPlan.Items) {
		t.Errorf("Expected plan to have %d items, got %d", len(testPlan.Items), len(form.plan.Items))
	}
}

func TestRenderPlanReviewModalQualitySummaryNoFindings(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 0,
		InitialWarningCount:  0,
		BlockingCount:        0,
		WarningCount:         0,
		KeyFindings:          nil,
		AutoRefinePassesRun:  0,
	})

	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 88
	m.windowHeight = 26

	rendered := stripANSI(RenderPlanReviewModal(m, form))

	for _, want := range []string{
		"Quality summary:",
		"Initial: blocking=0 warning=0",
		"Final: blocking=0 warning=0",
		"Key findings: none",
		"1. Accept",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected modal to include %q, got:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "Auto-refine:") {
		t.Fatalf("did not expect auto-refine line when passes run is zero, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "Accept anyway") {
		t.Fatalf("did not expect override label without blocking findings, got:\n%s", rendered)
	}
}

func TestRenderPlanReviewModalQualitySummaryBlockingAfterAutoRefine(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 2,
		InitialWarningCount:  1,
		BlockingCount:        1,
		WarningCount:         2,
		KeyFindings: []string{
			"task-1.description [blocking] Description must name explicit implementation scope.",
			"task-2.prompt [warning] Prompt should include a concrete verification command.",
		},
		AutoRefinePassesRun: 1,
	})

	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 120
	m.windowHeight = 30

	rendered := stripANSI(RenderPlanReviewModal(m, form))
	for _, want := range []string{
		"Quality summary:",
		"Initial: blocking=2 warning=1",
		"Final: blocking=1 warning=2",
		"Auto-refine: 1 pass run, blocking findings remain",
		"Blocking findings remain: explicit override required to accept",
		"task-1.description [blocking] Description must name explicit implementation scope.",
		"1. Accept anyway",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected modal to include %q, got:\n%s", want, rendered)
		}
	}
}

func TestBuildPlanReviewQualitySummaryDeterministicCountsAndKeyFindings(t *testing.T) {
	result := planquality.QualityGateResult{
		InitialFindings: []planquality.PlanQualityFinding{
			{
				Severity: planquality.SeverityWarning,
				Code:     planquality.RuleVagueLanguageDetected,
				TaskID:   "task-c",
				Field:    "task",
				Message:  "Language should describe explicit outputs.",
			},
			{
				Severity: planquality.SeverityBlocking,
				Code:     planquality.RuleLeafDescriptionMissingOrPlaceholder,
				TaskID:   "task-a",
				Field:    "description",
				Message:  "Description must include explicit scope.",
			},
		},
		FinalFindings: []planquality.PlanQualityFinding{
			{
				Severity: planquality.SeverityWarning,
				Code:     planquality.RuleLeafPromptMissingVerificationHint,
				TaskID:   "task-b",
				Field:    "prompt",
				Message:  "Prompt should include verification command.",
			},
			{
				Severity: planquality.SeverityBlocking,
				Code:     planquality.RuleLeafPromptMissing,
				TaskID:   "task-b",
				Field:    "prompt",
				Message:  "Prompt cannot be empty.",
			},
			{
				Severity: planquality.SeverityWarning,
				Code:     planquality.RuleLeafAcceptanceCriteriaLowCount,
				TaskID:   "task-a",
				Field:    "acceptanceCriteria",
				Message:  "Add one more acceptance criterion.",
			},
			{
				Severity: planquality.SeverityBlocking,
				Code:     planquality.RuleLeafDescriptionMissingOrPlaceholder,
				TaskID:   "task-a",
				Field:    "description",
				Message:  "Description must include explicit scope.",
			},
		},
		AutoRefinePassesRun: 2,
	}

	summary := buildPlanReviewQualitySummary(result)
	if summary.InitialBlockingCount != 1 || summary.InitialWarningCount != 1 {
		t.Fatalf("unexpected initial counts: blocking=%d warning=%d", summary.InitialBlockingCount, summary.InitialWarningCount)
	}
	if summary.BlockingCount != 2 || summary.WarningCount != 2 {
		t.Fatalf("unexpected final counts: blocking=%d warning=%d", summary.BlockingCount, summary.WarningCount)
	}
	if summary.AutoRefinePassesRun != 2 {
		t.Fatalf("AutoRefinePassesRun = %d, want 2", summary.AutoRefinePassesRun)
	}

	wantKeyFindings := []string{
		"task-a.description [blocking] Description must include explicit scope.",
		"task-a.acceptanceCriteria [warning] Add one more acceptance criterion.",
		"task-b.prompt [blocking] Prompt cannot be empty.",
	}
	if len(summary.KeyFindings) != len(wantKeyFindings) {
		t.Fatalf("KeyFindings length = %d, want %d (%v)", len(summary.KeyFindings), len(wantKeyFindings), summary.KeyFindings)
	}
	for i := range wantKeyFindings {
		if summary.KeyFindings[i] != wantKeyFindings[i] {
			t.Fatalf("KeyFindings[%d] = %q, want %q", i, summary.KeyFindings[i], wantKeyFindings[i])
		}
	}
}

func TestRenderPlanReviewModalQualitySummaryPluralPassesNoBlockingRemain(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 2,
		InitialWarningCount:  1,
		BlockingCount:        0,
		WarningCount:         2,
		KeyFindings: []string{
			"task-2.prompt [warning] Prompt should include a concrete verification command.",
		},
		AutoRefinePassesRun: 2,
	})

	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 110
	m.windowHeight = 28

	rendered := stripANSI(RenderPlanReviewModal(m, form))
	for _, want := range []string{
		"Quality summary:",
		"Initial: blocking=2 warning=1",
		"Final: blocking=0 warning=2",
		"Auto-refine: 2 passes run, no blocking findings remain",
		"1. Accept",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected modal to include %q, got:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "Accept anyway") {
		t.Fatalf("did not expect override label without blocking findings, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "Blocking findings remain: explicit override required to accept") {
		t.Fatalf("did not expect blocking warning when final blocking count is zero, got:\n%s", rendered)
	}
}

// TestPlanReviewNavigationUpDown tests navigation between actions
func TestPlanReviewNavigationUpDown(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 0,
		InitialWarningCount:  0,
		BlockingCount:        0,
		WarningCount:         0,
	})

	// Start at action 0 (Accept)
	if form.selectedAction != planReviewActionAccept {
		t.Errorf("Expected initial action to be 0, got %d", form.selectedAction)
	}

	// Press down - should move to action 1 (Revise)
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != planReviewActionRevise {
		t.Errorf("Expected action to be 1 after down, got %d", form.selectedAction)
	}

	// Press down again - should move to action 2 (Reject)
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != planReviewActionReject {
		t.Errorf("Expected action to be 2 after second down, got %d", form.selectedAction)
	}

	// Press down at bottom - should stay at 2
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != planReviewActionReject {
		t.Errorf("Expected action to stay at 2 at bottom, got %d", form.selectedAction)
	}

	// Press up - should move to action 1
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != planReviewActionRevise {
		t.Errorf("Expected action to be 1 after up, got %d", form.selectedAction)
	}

	// Press up again - should move to action 0
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != planReviewActionAccept {
		t.Errorf("Expected action to be 0 after second up, got %d", form.selectedAction)
	}

	// Press up at top - should stay at 0
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != planReviewActionAccept {
		t.Errorf("Expected action to stay at 0 at top, got %d", form.selectedAction)
	}
}

// TestPlanReviewQuickSelect tests number key quick selection
func TestPlanReviewQuickSelect(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 0,
		InitialWarningCount:  0,
		BlockingCount:        0,
		WarningCount:         0,
	})

	// Press 1 - select Accept
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if form.selectedAction != planReviewActionAccept {
		t.Errorf("Expected action to be 0 after pressing 1, got %d", form.selectedAction)
	}

	// Press 2 - select Revise
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if form.selectedAction != planReviewActionRevise {
		t.Errorf("Expected action to be 1 after pressing 2, got %d", form.selectedAction)
	}

	// Press 3 - select Reject
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if form.selectedAction != planReviewActionReject {
		t.Errorf("Expected action to be 2 after pressing 3, got %d", form.selectedAction)
	}
}

func TestPlanReviewBlockingDefaultAction(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 1,
		InitialWarningCount:  0,
		BlockingCount:        1,
		WarningCount:         0,
	})

	if form.selectedAction != planReviewActionRevise {
		t.Fatalf("expected default action to be Revise when blocking findings remain, got %d", form.selectedAction)
	}

	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if form.selectedAction != planReviewActionAccept {
		t.Fatalf("expected quick-select 1 to target Accept anyway, got %d", form.selectedAction)
	}
}

// TestPlanReviewRevisionLimitBlocking tests that revision is blocked when limit reached
func TestPlanReviewRevisionLimitBlocking(t *testing.T) {
	testPlan := createTestPlan()
	// Create form with revision count at the limit
	form := NewPlanReviewForm(testPlan, agent.MaxPlanGenerateRevisions)

	if form.CanRevise() {
		t.Error("Expected CanRevise to return false when at revision limit")
	}

	// Try to select Revise with number key - should not change selection
	form.selectedAction = planReviewActionAccept
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if form.selectedAction != planReviewActionAccept {
		t.Errorf("Expected action to remain 0 when revision limit reached, got %d", form.selectedAction)
	}
}

func TestPlanReviewBlockingDefaultsToRejectWhenRevisionLimitReached(t *testing.T) {
	testPlan := createTestPlan()
	form := NewPlanReviewForm(testPlan, agent.MaxPlanGenerateRevisions)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 1,
		InitialWarningCount:  0,
		BlockingCount:        1,
		WarningCount:         0,
	})

	if form.selectedAction != planReviewActionReject {
		t.Fatalf("expected default action to be Reject when blocking findings remain and revisions are exhausted, got %d", form.selectedAction)
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
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 1,
		InitialWarningCount:  0,
		BlockingCount:        1,
		WarningCount:         0,
	})
	form.selectedAction = planReviewActionRevise
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
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 0,
		InitialWarningCount:  0,
		BlockingCount:        0,
		WarningCount:         0,
	})
	form.selectedAction = planReviewActionAccept
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
	form.selectedAction = planReviewActionReject
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

func TestPlanReviewAcceptAnywayWithBlocking(t *testing.T) {
	testPlan := createTestPlan()
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 100
	m.windowHeight = 30
	m.planExists = false
	m.viewMode = ViewModeHome

	form := NewPlanReviewForm(testPlan, 0)
	form.SetQualitySummary(PlanReviewQualitySummary{
		InitialBlockingCount: 2,
		InitialWarningCount:  1,
		BlockingCount:        1,
		WarningCount:         2,
	})
	form.selectedAction = planReviewActionAccept
	m.planReviewForm = &form
	m.actionMode = ActionModePlanReview

	m, cmd := HandlePlanReviewKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save command for accept-anyway")
	}
	if m.actionOutput == nil || !strings.Contains(m.actionOutput.Message, "blocking findings were overridden") {
		t.Fatalf("expected immediate override warning, got %#v", m.actionOutput)
	}

	msg := cmd()
	updatedModel, _ := m.Update(msg)
	updated, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected Model from update, got %T", updatedModel)
	}
	if updated.actionOutput == nil || !strings.Contains(updated.actionOutput.Message, "blocking findings were overridden") {
		t.Fatalf("expected persisted override warning in action output, got %#v", updated.actionOutput)
	}
	if !updated.planExists {
		t.Fatal("expected overridden save to mark plan as existing")
	}
	if updated.viewMode != ViewModeMain {
		t.Fatalf("expected overridden save to return to main view, got %v", updated.viewMode)
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
