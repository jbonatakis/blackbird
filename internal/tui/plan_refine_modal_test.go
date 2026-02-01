package tui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestHandlePlanRefineKeySubmit(t *testing.T) {
	base := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {ID: "task-1", Title: "Task"},
		},
	}

	form := NewPlanRefineForm()
	form.changeRequest.SetValue("Update tasks")
	form.focusedField = RefineFieldSubmit // focus Submit button

	m := Model{
		plan:           base,
		planExists:     true,
		actionMode:     ActionModePlanRefine,
		planRefineForm: &form,
	}

	updated, _ := HandlePlanRefineKey(m, tea.KeyMsg{Type: tea.KeyEnter})

	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected action mode to reset after submit, got %v", updated.actionMode)
	}
	if !updated.actionInProgress {
		t.Fatalf("expected refine action to start")
	}
	if updated.pendingPlanRequest.kind != PendingPlanRefine {
		t.Fatalf("expected pending request kind to be refine")
	}
	if updated.pendingPlanRequest.changeRequest != "Update tasks" {
		t.Fatalf("unexpected change request: %q", updated.pendingPlanRequest.changeRequest)
	}
	if !reflect.DeepEqual(updated.pendingPlanRequest.basePlan, base) {
		t.Fatalf("expected base plan to be cloned from model plan")
	}
}

func TestHandlePlanRefineKeyEmptyRequest(t *testing.T) {
	form := NewPlanRefineForm()
	form.focusedField = RefineFieldSubmit // focus Submit button, but text is empty

	m := Model{
		planExists:     true,
		actionMode:     ActionModePlanRefine,
		planRefineForm: &form,
	}

	updated, _ := HandlePlanRefineKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if updated.actionMode != ActionModePlanRefine {
		t.Fatalf("expected refine modal to remain open on empty submit")
	}
	if updated.actionInProgress {
		t.Fatalf("expected refine action to stay idle on empty submit")
	}
}

func TestModel_PlanRefinePickerOpensOnAt(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanRefineForm()

	m.planRefineForm = &form
	m.actionMode = ActionModePlanRefine

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModePlanRefine {
		t.Fatalf("expected ActionModePlanRefine, got %v", m.actionMode)
	}
	if m.planRefineForm == nil {
		t.Fatalf("expected planRefineForm to remain open")
	}
	if !m.planRefineForm.filePicker.Open {
		t.Fatalf("expected file picker to be open after @")
	}
	if m.planRefineForm.filePicker.ActiveField != planRefinePickerChangeRequest {
		t.Fatalf("expected active field %q, got %q", planRefinePickerChangeRequest, m.planRefineForm.filePicker.ActiveField)
	}
	if m.planRefineForm.changeRequest.Value() != "@" {
		t.Fatalf("expected change request to contain @, got %q", m.planRefineForm.changeRequest.Value())
	}
}

func TestModel_PlanRefinePickerEnterInsertsChangeRequest(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanRefineForm()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}
	if !form.OpenFilePicker(anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planRefineForm = &form
	m.actionMode = ActionModePlanRefine

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModePlanRefine {
		t.Fatalf("expected ActionModePlanRefine, got %v", m.actionMode)
	}
	if m.planRefineForm == nil {
		t.Fatalf("expected planRefineForm to remain open")
	}
	if m.planRefineForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after insert")
	}
	expected := "See @src/main.go"
	if m.planRefineForm.changeRequest.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, m.planRefineForm.changeRequest.Value())
	}
}

func TestModel_PlanRefinePickerEscClosesPicker(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanRefineForm()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}
	if !form.OpenFilePicker(anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planRefineForm = &form
	m.actionMode = ActionModePlanRefine

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.actionMode != ActionModePlanRefine {
		t.Fatalf("expected ActionModePlanRefine, got %v", m.actionMode)
	}
	if m.planRefineForm == nil {
		t.Fatalf("expected planRefineForm to remain open")
	}
	if m.planRefineForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after esc")
	}
}

func TestModel_PlanRefinePickerTabClosesAndMovesFocus(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanRefineForm()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}
	if !form.OpenFilePicker(anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planRefineForm = &form
	m.actionMode = ActionModePlanRefine

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.planRefineForm == nil {
		t.Fatalf("expected planRefineForm to remain open")
	}
	if m.planRefineForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after tab")
	}
	if m.planRefineForm.focusedField != RefineFieldSubmit {
		t.Fatalf("expected focus to move to submit, got %v", m.planRefineForm.focusedField)
	}
}

func TestModel_PlanRefinePickerShiftTabClosesAndMovesFocus(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	form := NewPlanRefineForm()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}
	if !form.OpenFilePicker(anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	m.planRefineForm = &form
	m.actionMode = ActionModePlanRefine

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.planRefineForm == nil {
		t.Fatalf("expected planRefineForm to remain open")
	}
	if m.planRefineForm.filePicker.Open {
		t.Fatalf("expected file picker to be closed after shift+tab")
	}
	if m.planRefineForm.focusedField != RefineFieldSubmit {
		t.Fatalf("expected focus to move to submit, got %v", m.planRefineForm.focusedField)
	}
}
