package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// TestPlanGenerateForm_EmptyDescription tests that empty description fails validation
func TestPlanGenerateForm_EmptyDescription(t *testing.T) {
	form := NewPlanGenerateForm()
	form.description.SetValue("   ") // Whitespace only
	form.constraints.SetValue("constraint1")
	form.granularity.SetValue("detailed")

	err := form.Validate()
	if err == nil {
		t.Error("Expected validation error for whitespace-only description")
	}
}

// TestPlanGenerateForm_VeryLongDescription tests character limit enforcement
func TestPlanGenerateForm_VeryLongDescription(t *testing.T) {
	form := NewPlanGenerateForm()

	// Create a string longer than the 5000 character limit
	longString := strings.Repeat("a", 6000)
	form.description.SetValue(longString)

	// The textarea should enforce the limit
	actualValue := form.description.Value()
	if len(actualValue) > 5000 {
		t.Errorf("Expected description to be limited to 5000 chars, got %d", len(actualValue))
	}
}

// TestPlanGenerateForm_SpecialCharactersInDescription tests special character handling
func TestPlanGenerateForm_SpecialCharactersInDescription(t *testing.T) {
	form := NewPlanGenerateForm()

	specialChars := `<>&"'\n\t\r`
	form.description.SetValue(specialChars)

	desc, _, _ := form.GetValues()
	if desc != specialChars {
		t.Errorf("Expected special characters to be preserved, got '%s'", desc)
	}
}

// TestPlanGenerateForm_ConstraintsParsing tests comma-separated constraint parsing
func TestPlanGenerateForm_ConstraintsParsing(t *testing.T) {
	form := NewPlanGenerateForm()

	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "Go, React, PostgreSQL",
			expected: []string{"Go", "React", "PostgreSQL"},
		},
		{
			input:    "Go,React,PostgreSQL",
			expected: []string{"Go", "React", "PostgreSQL"},
		},
		{
			input:    "  Go  ,  React  ,  PostgreSQL  ",
			expected: []string{"Go", "React", "PostgreSQL"},
		},
		{
			input:    "Single",
			expected: []string{"Single"},
		},
		{
			input:    "",
			expected: nil,
		},
		{
			input:    ",,",
			expected: nil, // Empty parts should be filtered
		},
		{
			input:    "Valid,,Another",
			expected: []string{"Valid", "Another"},
		},
	}

	for _, tc := range testCases {
		form.constraints.SetValue(tc.input)
		_, constraints, _ := form.GetValues()

		if len(constraints) != len(tc.expected) {
			t.Errorf("For input '%s', expected %d constraints, got %d", tc.input, len(tc.expected), len(constraints))
			continue
		}

		for i, expected := range tc.expected {
			if constraints[i] != expected {
				t.Errorf("For input '%s', expected constraint[%d]='%s', got '%s'", tc.input, i, expected, constraints[i])
			}
		}
	}
}

// TestPlanGenerateForm_AllFieldsOptionalExceptDescription tests that only description is required
func TestPlanGenerateForm_AllFieldsOptionalExceptDescription(t *testing.T) {
	form := NewPlanGenerateForm()

	// Only description filled
	form.description.SetValue("Project description")
	// constraints and granularity left empty

	err := form.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass with only description, got error: %v", err)
	}

	desc, constraints, gran := form.GetValues()
	if desc != "Project description" {
		t.Errorf("Expected description 'Project description', got '%s'", desc)
	}
	if len(constraints) != 0 {
		t.Errorf("Expected no constraints, got %d", len(constraints))
	}
	if gran != "" {
		t.Errorf("Expected empty granularity, got '%s'", gran)
	}
}

// TestPlanGenerateForm_SubmitWithInvalidData tests that form stays open when validation fails
func TestPlanGenerateForm_SubmitWithInvalidData(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Open modal
	form := NewPlanGenerateForm()
	// Leave description empty (invalid)
	form.focusedField = FieldSubmit
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Try to submit with Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Modal should still be open
	if m.actionMode != ActionModeGeneratePlan {
		t.Errorf("Expected modal to stay open on validation failure, got mode %v", m.actionMode)
	}

	if m.planGenerateForm == nil {
		t.Error("Expected form to still exist after validation failure")
	}

	// Should not start action
	if m.actionInProgress {
		t.Error("Expected actionInProgress to be false after validation failure")
	}
}

// TestPlanGenerateForm_FocusWrapsAround tests that focus wraps around at boundaries
func TestPlanGenerateForm_FocusWrapsAround(t *testing.T) {
	form := NewPlanGenerateForm()

	// Start at description
	if form.focusedField != FieldDescription {
		t.Fatalf("Expected initial focus on FieldDescription")
	}

	// Move forward through all fields back to description
	form = form.focusNext() // Constraints
	form = form.focusNext() // Granularity
	form = form.focusNext() // Submit
	form = form.focusNext() // Should wrap to Description

	if form.focusedField != FieldDescription {
		t.Errorf("Expected focus to wrap to FieldDescription, got %v", form.focusedField)
	}

	// Move backward from description to submit
	form = form.focusPrev()
	if form.focusedField != FieldSubmit {
		t.Errorf("Expected focus to wrap to FieldSubmit when going back from first, got %v", form.focusedField)
	}
}

// TestModel_RapidKeyPressesWhileGenerating tests that input is ignored during generation
func TestModel_RapidKeyPressesWhileGenerating(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionInProgress = true
	m.actionName = "Generating plan..."

	// Try to press 'g' while generation is in progress
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Should not open modal
	if m.actionMode == ActionModeGeneratePlan {
		t.Error("Expected modal not to open while action is in progress")
	}
	if m.planGenerateForm != nil {
		t.Error("Expected planGenerateForm to remain nil during action")
	}
}

// TestModel_WindowResizeDuringModal tests that modal handles window resize
func TestModel_WindowResizeDuringModal(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Open modal
	form := NewPlanGenerateForm()
	form.SetSize(80, 24)
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Check that model dimensions updated
	if m.windowWidth != 120 {
		t.Errorf("Expected windowWidth to be 120, got %d", m.windowWidth)
	}
	if m.windowHeight != 40 {
		t.Errorf("Expected windowHeight to be 40, got %d", m.windowHeight)
	}

	// Note: In real implementation, modal should call SetSize on resize
	// This test verifies dimensions are tracked; actual resize handling
	// should be tested manually or with a more sophisticated test
}

// TestModel_EscFromModalClearsState tests that ESC properly cleans up modal state
func TestModel_EscFromModalClearsState(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Set up pending request
	m.pendingPlanRequest = PendingPlanRequest{
		description:   "Test",
		constraints:   []string{"c1"},
		granularity:   "detailed",
		questionRound: 0,
	}

	// Open modal
	form := NewPlanGenerateForm()
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Press ESC
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Check cleanup
	if m.actionMode != ActionModeNone {
		t.Errorf("Expected ActionModeNone, got %v", m.actionMode)
	}
	if m.planGenerateForm != nil {
		t.Error("Expected planGenerateForm to be nil")
	}

	// Note: pendingPlanRequest is intentionally kept for potential resume
	// but that's an implementation detail that could change
}

// TestModel_MultipleModalsPreventedConcurrently tests that only one modal can be open
func TestModel_MultipleModalsPreventedConcurrently(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Open plan generate modal
	form := NewPlanGenerateForm()
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Try to open set-status modal (press 's')
	m.selectedID = "task-1"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	// Should still be in generate plan mode
	if m.actionMode != ActionModeGeneratePlan {
		t.Errorf("Expected to remain in ActionModeGeneratePlan, got %v", m.actionMode)
	}

	// Should not have switched to set-status mode
	if m.pendingStatusID != "" {
		t.Error("Expected pendingStatusID to remain empty")
	}
}

// TestPlanGenerateForm_TabAndEnterBehavior tests tab vs enter on form fields
func TestPlanGenerateForm_TabAndEnterBehavior(t *testing.T) {
	form := NewPlanGenerateForm()

	// On description field (textarea), Enter inserts newline and stays; Tab moves to next
	if form.focusedField != FieldDescription {
		t.Fatalf("Expected initial focus on FieldDescription")
	}

	// Enter on description inserts newline (stays on description)
	updatedForm, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm
	if form.focusedField != FieldDescription {
		t.Errorf("Expected Enter on textarea to stay on FieldDescription (newline), got %v", form.focusedField)
	}

	// Tab moves to next field
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyTab})
	form = updatedForm
	if form.focusedField != FieldConstraints {
		t.Errorf("Expected Tab to move to FieldConstraints, got %v", form.focusedField)
	}
}

// TestModel_QuitDuringModal tests that Ctrl+C still works
func TestModel_QuitDuringModal(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Open modal
	form := NewPlanGenerateForm()
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan

	// Press Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	// Should return quit command
	if cmd == nil {
		t.Error("Expected quit command, got nil")
	}
	// In real Bubble Tea, this would be tea.Quit, but we can't easily test that
}

// TestModel_ValidationErrorMessage tests that validation errors are shown
func TestModel_ValidationErrorMessage(t *testing.T) {
	form := NewPlanGenerateForm()
	form.focusedField = FieldSubmit

	// Validation should fail
	err := form.Validate()
	if err == nil {
		t.Fatal("Expected validation error")
	}

	// Error message should contain useful info
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Should mention the required field
	if !strings.Contains(strings.ToLower(errMsg), "description") {
		t.Errorf("Expected error message to mention 'description', got: %s", errMsg)
	}
}
