package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestPlanGenerateForm_Validation(t *testing.T) {
	form := NewPlanGenerateForm()

	// Empty form should fail validation
	if err := form.Validate(); err == nil {
		t.Error("Expected validation error for empty description, got nil")
	}

	// Set description
	form.description.SetValue("Test project description")

	// Should pass validation now
	if err := form.Validate(); err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}

func TestPlanGenerateForm_GetValues(t *testing.T) {
	form := NewPlanGenerateForm()

	form.description.SetValue("Test description")
	form.constraints.SetValue("constraint1, constraint2, constraint3")
	form.granularity.SetValue("detailed")

	desc, constraints, gran := form.GetValues()

	if desc != "Test description" {
		t.Errorf("Expected description 'Test description', got '%s'", desc)
	}

	expectedConstraints := []string{"constraint1", "constraint2", "constraint3"}
	if len(constraints) != len(expectedConstraints) {
		t.Errorf("Expected %d constraints, got %d", len(expectedConstraints), len(constraints))
	}

	for i, c := range constraints {
		if c != expectedConstraints[i] {
			t.Errorf("Expected constraint[%d] = '%s', got '%s'", i, expectedConstraints[i], c)
		}
	}

	if gran != "detailed" {
		t.Errorf("Expected granularity 'detailed', got '%s'", gran)
	}
}

func TestPlanGenerateForm_FocusNavigation(t *testing.T) {
	form := NewPlanGenerateForm()

	// Initial focus should be on description
	if form.focusedField != FieldDescription {
		t.Errorf("Expected initial focus on FieldDescription, got %v", form.focusedField)
	}

	// Move to next field
	form = form.focusNext()
	if form.focusedField != FieldConstraints {
		t.Errorf("Expected focus on FieldConstraints, got %v", form.focusedField)
	}

	// Move to next field
	form = form.focusNext()
	if form.focusedField != FieldGranularity {
		t.Errorf("Expected focus on FieldGranularity, got %v", form.focusedField)
	}

	// Move to submit
	form = form.focusNext()
	if form.focusedField != FieldSubmit {
		t.Errorf("Expected focus on FieldSubmit, got %v", form.focusedField)
	}

	// Cycle back to description
	form = form.focusNext()
	if form.focusedField != FieldDescription {
		t.Errorf("Expected focus to cycle back to FieldDescription, got %v", form.focusedField)
	}

	// Test backward navigation
	form = form.focusPrev()
	if form.focusedField != FieldSubmit {
		t.Errorf("Expected focus on FieldSubmit, got %v", form.focusedField)
	}

	// Test backward from Submit to Granularity
	form = form.focusPrev()
	if form.focusedField != FieldGranularity {
		t.Errorf("Expected focus on FieldGranularity, got %v", form.focusedField)
	}

	// Verify granularity is focused
	if !form.granularity.Focused() {
		t.Error("Expected granularity textinput to be focused")
	}
}

func TestModel_OpenPlanGenerateModal(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Press 'g' to open modal
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Check that modal is open
	if m.actionMode != ActionModeGeneratePlan {
		t.Errorf("Expected ActionModeGeneratePlan, got %v", m.actionMode)
	}

	if m.planGenerateForm == nil {
		t.Error("Expected planGenerateForm to be initialized, got nil")
	}
}

func TestModel_ClosePlanGenerateModal(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Open modal
	form := NewPlanGenerateForm()
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Press ESC to close
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Check that modal is closed
	if m.actionMode != ActionModeNone {
		t.Errorf("Expected ActionModeNone, got %v", m.actionMode)
	}

	if m.planGenerateForm != nil {
		t.Error("Expected planGenerateForm to be nil after close, got non-nil")
	}
}

func TestModel_EscClosesFilePickerInPlanGenerateModal(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanGenerateForm()
	form.description.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}
	if !form.OpenFilePicker(FieldDescription, anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModeGeneratePlan {
		t.Fatalf("expected ActionModeGeneratePlan, got %v", m.actionMode)
	}
	if m.planGenerateForm == nil {
		t.Fatalf("expected planGenerateForm to remain open")
	}
	if m.planGenerateForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after esc")
	}
}

func TestModel_PlanGeneratePickerOpensOnAt(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanGenerateForm()

	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModeGeneratePlan {
		t.Fatalf("expected ActionModeGeneratePlan, got %v", m.actionMode)
	}
	if m.planGenerateForm == nil {
		t.Fatalf("expected planGenerateForm to remain open")
	}
	if !m.planGenerateForm.filePicker.Open {
		t.Fatalf("expected file picker to be open after @")
	}
	if m.planGenerateForm.filePicker.ActiveField != planGeneratePickerDescription {
		t.Fatalf("expected active field %q, got %q", planGeneratePickerDescription, m.planGenerateForm.filePicker.ActiveField)
	}
	if m.planGenerateForm.description.Value() != "@" {
		t.Fatalf("expected description to contain @, got %q", m.planGenerateForm.description.Value())
	}
}

func TestModel_PlanGeneratePickerEnterInsertsDescription(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanGenerateForm()
	form.description.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}
	if !form.OpenFilePicker(FieldDescription, anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModeGeneratePlan {
		t.Fatalf("expected ActionModeGeneratePlan, got %v", m.actionMode)
	}
	if m.planGenerateForm == nil {
		t.Fatalf("expected planGenerateForm to remain open")
	}
	if m.planGenerateForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after insert")
	}
	expected := "See @src/main.go"
	if m.planGenerateForm.description.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, m.planGenerateForm.description.Value())
	}
}

func TestModel_PlanGeneratePickerEnterInsertsConstraints(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanGenerateForm()
	form = form.focusNext() // Constraints
	form.constraints.SetValue("Use @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.constraints.Value(), "@")}
	if !form.OpenFilePicker(FieldConstraints, anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/config.yaml"}
	form.filePicker.Selected = 0

	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModeGeneratePlan {
		t.Fatalf("expected ActionModeGeneratePlan, got %v", m.actionMode)
	}
	if m.planGenerateForm == nil {
		t.Fatalf("expected planGenerateForm to remain open")
	}
	if m.planGenerateForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after insert")
	}
	expected := "Use @src/config.yaml"
	if m.planGenerateForm.constraints.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, m.planGenerateForm.constraints.Value())
	}
}

func TestModel_PlanGeneratePickerTabClosesAndMovesFocus(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanGenerateForm()
	form.description.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}
	if !form.OpenFilePicker(FieldDescription, anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.planGenerateForm == nil {
		t.Fatalf("expected planGenerateForm to remain open")
	}
	if m.planGenerateForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after tab")
	}
	if m.planGenerateForm.focusedField != FieldConstraints {
		t.Fatalf("expected focus to move to constraints, got %v", m.planGenerateForm.focusedField)
	}
	if !m.planGenerateForm.constraints.Focused() {
		t.Fatalf("expected constraints to be focused")
	}
}

func TestModel_PlanGeneratePickerShiftTabClosesAndMovesFocus(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanGenerateForm()
	form = form.focusNext() // Constraints
	form.constraints.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.constraints.Value(), "@")}
	if !form.OpenFilePicker(FieldConstraints, anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.planGenerateForm == nil {
		t.Fatalf("expected planGenerateForm to remain open")
	}
	if m.planGenerateForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after shift+tab")
	}
	if m.planGenerateForm.focusedField != FieldDescription {
		t.Fatalf("expected focus to move to description, got %v", m.planGenerateForm.focusedField)
	}
	if !m.planGenerateForm.description.Focused() {
		t.Fatalf("expected description to be focused")
	}
}

func TestModel_SpinnerDuringPlanGeneration(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Open modal and set form values
	form := NewPlanGenerateForm()
	form.description.SetValue("Test project description")
	form.focusedField = FieldSubmit
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Submit form with Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	m = updated.(Model)

	// Check that spinner is activated
	if !m.actionInProgress {
		t.Error("Expected actionInProgress to be true after form submission")
	}

	if m.actionName != "Generating plan..." {
		t.Errorf("Expected actionName to be 'Generating plan...', got '%s'", m.actionName)
	}

	// Check that modal is closed
	if m.actionMode != ActionModeNone {
		t.Errorf("Expected ActionModeNone after submission, got %v", m.actionMode)
	}

	if m.planGenerateForm != nil {
		t.Error("Expected planGenerateForm to be nil after submission, got non-nil")
	}

	// Verify that command was returned (for async execution)
	if cmd == nil {
		t.Error("Expected non-nil command to be returned for plan generation")
	}
}

func TestRenderPlanGenerateModal_IncludesFilePickerWhenOpen(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	form := NewPlanGenerateForm()
	form.SetSize(80, 24)
	form.description.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}
	if !form.OpenFilePicker(FieldDescription, anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go", "src/other.go"}
	form.filePicker.Selected = 0

	out := RenderPlanGenerateModal(m, form)
	if !strings.Contains(out, "src/main.go") {
		t.Fatalf("expected picker output to include match, got:\n%s", out)
	}

	descIndex := strings.Index(out, "Project Description (required):")
	pickerIndex := strings.Index(out, "src/main.go")
	constraintsIndex := strings.Index(out, "Constraints (optional, comma-separated):")
	if descIndex == -1 || pickerIndex == -1 || constraintsIndex == -1 {
		t.Fatalf("expected labels and picker output to be present")
	}
	if !(descIndex < pickerIndex && pickerIndex < constraintsIndex) {
		t.Fatalf("expected picker output between description and constraints sections")
	}
}

func TestModel_HandlePlanGenerationSuccess(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionInProgress = true
	m.actionName = "Generating plan..."

	// Simulate successful plan generation
	newPlan := plan.NewEmptyWorkGraph()
	newPlan.Items["test-id"] = plan.WorkItem{
		ID:                 "test-id",
		Title:              "Test Task",
		Description:        "Test Description",
		AcceptanceCriteria: []string{"Test AC"},
		Prompt:             "Test Prompt",
		Status:             plan.StatusTodo,
	}

	msg := PlanGenerateInMemoryResult{
		Success: true,
		Plan:    &newPlan,
		Quality: &PlanReviewQualitySummary{
			InitialBlockingCount: 0,
			InitialWarningCount:  0,
			BlockingCount:        0,
			WarningCount:         0,
			KeyFindings:          nil,
			AutoRefinePassesRun:  0,
		},
		Err: nil,
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Check that spinner is cleared
	if m.actionInProgress {
		t.Error("Expected actionInProgress to be false after success")
	}

	if m.actionName != "" {
		t.Errorf("Expected actionName to be empty, got '%s'", m.actionName)
	}

	// Check that plan review modal is shown (plan is not yet applied)
	if m.actionMode != ActionModePlanReview {
		t.Errorf("Expected actionMode to be ActionModePlanReview, got %v", m.actionMode)
	}

	if m.planReviewForm == nil {
		t.Fatal("Expected planReviewForm to be set")
	}

	// Check that the review form contains the generated plan
	if len(m.planReviewForm.plan.Items) != 1 {
		t.Errorf("Expected review form plan to have 1 item, got %d", len(m.planReviewForm.plan.Items))
	}
	if m.planReviewForm.qualitySummary.WarningCount != 0 {
		t.Errorf("Expected warning count 0, got %d", m.planReviewForm.qualitySummary.WarningCount)
	}
	if m.planReviewForm.qualitySummary.BlockingCount != 0 {
		t.Errorf("Expected blocking count 0, got %d", m.planReviewForm.qualitySummary.BlockingCount)
	}
	if len(m.planReviewForm.qualitySummary.KeyFindings) != 0 {
		t.Errorf("Expected no key findings, got %d", len(m.planReviewForm.qualitySummary.KeyFindings))
	}

	// Plan should NOT be applied to the model yet (only after accepting)
	if len(m.plan.Items) != 0 {
		t.Errorf("Expected model plan to still be empty until accepted, got %d items", len(m.plan.Items))
	}
}

func TestModel_HandlePlanGenerationSuccess_WithBlockingQualitySummary(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionInProgress = true
	m.actionName = "Generating plan..."

	newPlan := plan.NewEmptyWorkGraph()
	newPlan.Items["task-block"] = plan.WorkItem{
		ID:                 "task-block",
		Title:              "Task with findings",
		Description:        "Description",
		AcceptanceCriteria: []string{"AC"},
		Prompt:             "Prompt",
		Status:             plan.StatusTodo,
	}

	msg := PlanGenerateInMemoryResult{
		Success: true,
		Plan:    &newPlan,
		Quality: &PlanReviewQualitySummary{
			InitialBlockingCount: 2,
			InitialWarningCount:  1,
			BlockingCount:        1,
			WarningCount:         2,
			KeyFindings: []string{
				"task-block.description [blocking] Description must include concrete implementation scope.",
			},
			AutoRefinePassesRun: 1,
		},
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.planReviewForm == nil {
		t.Fatalf("expected plan review form to be set")
	}
	if m.planReviewForm.qualitySummary.BlockingCount != 1 {
		t.Fatalf("expected blocking count 1, got %d", m.planReviewForm.qualitySummary.BlockingCount)
	}
	if m.planReviewForm.qualitySummary.AutoRefinePassesRun != 1 {
		t.Fatalf("expected auto-refine pass count 1, got %d", m.planReviewForm.qualitySummary.AutoRefinePassesRun)
	}
	if len(m.planReviewForm.qualitySummary.KeyFindings) == 0 {
		t.Fatalf("expected key findings to be populated")
	}
}

func TestModel_HandlePlanGenerationError(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionInProgress = true
	m.actionName = "Generating plan..."

	// Simulate error during plan generation
	msg := PlanGenerateInMemoryResult{
		Success: false,
		Err:     &ValidationError{Field: "description", Message: "Invalid description"},
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Check that spinner is cleared
	if m.actionInProgress {
		t.Error("Expected actionInProgress to be false after error")
	}

	if m.actionName != "" {
		t.Errorf("Expected actionName to be empty, got '%s'", m.actionName)
	}

	// Check that error message is shown
	if m.actionOutput == nil {
		t.Error("Expected actionOutput to be set")
	} else if !m.actionOutput.IsError {
		t.Error("Expected actionOutput to be an error")
	}
}
