package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestPlanRefineForm_OpenFilePickerTracksField(t *testing.T) {
	form := NewPlanRefineForm()
	anchor := FilePickerAnchor{Start: 3}

	opened := form.OpenFilePicker(anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	if !form.filePicker.Open {
		t.Fatalf("expected file picker to be open")
	}
	if form.filePicker.ActiveField != planRefinePickerChangeRequest {
		t.Fatalf("expected active field %q, got %q", planRefinePickerChangeRequest, form.filePicker.ActiveField)
	}
	if form.requestAnchor != anchor {
		t.Fatalf("expected request anchor to be stored")
	}
}

func TestPlanRefineForm_FilePickerOpenAndQueryUpdates(t *testing.T) {
	form := NewPlanRefineForm()

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	form = updated

	if !form.filePicker.Open {
		t.Fatalf("expected file picker to open on @")
	}
	if form.filePicker.ActiveField != planRefinePickerChangeRequest {
		t.Fatalf("expected active field %q, got %q", planRefinePickerChangeRequest, form.filePicker.ActiveField)
	}
	if form.changeRequest.Value() != "@" {
		t.Fatalf("expected change request to contain @, got %q", form.changeRequest.Value())
	}

	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	form = updated
	if form.filePicker.Query != "s" {
		t.Fatalf("expected query %q, got %q", "s", form.filePicker.Query)
	}
	if form.changeRequest.Value() != "@s" {
		t.Fatalf("expected change request %q, got %q", "@s", form.changeRequest.Value())
	}

	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	form = updated
	if form.filePicker.Query != "" {
		t.Fatalf("expected query to be cleared, got %q", form.filePicker.Query)
	}
	if form.changeRequest.Value() != "@" {
		t.Fatalf("expected change request %q, got %q", "@", form.changeRequest.Value())
	}
}

func TestPlanRefineForm_FilePickerTabClosesAndMovesFocus(t *testing.T) {
	form := NewPlanRefineForm()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}

	opened := form.OpenFilePicker(anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyTab})
	form = updated

	if form.filePicker.Open {
		t.Fatalf("expected file picker to close on tab")
	}
	if form.focusedField != RefineFieldSubmit {
		t.Fatalf("expected focus to move to submit, got %v", form.focusedField)
	}
}

func TestPlanRefineForm_FilePickerEnterInsertsSelection(t *testing.T) {
	form := NewPlanRefineForm()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}

	opened := form.OpenFilePicker(anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updated

	expected := "See @src/main.go"
	if form.changeRequest.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, form.changeRequest.Value())
	}
	if form.filePicker.Open {
		t.Fatalf("expected file picker to be closed after insert")
	}
}

func TestRenderPlanRefineModal_IncludesFilePickerWhenOpen(t *testing.T) {
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	form := NewPlanRefineForm()
	form.SetSize(80, 24)
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}
	if !form.OpenFilePicker(anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go", "src/other.go"}
	form.filePicker.Selected = 0

	out := RenderPlanRefineModal(m, form)
	if !strings.Contains(out, "src/main.go") {
		t.Fatalf("expected picker output to include match, got:\n%s", out)
	}

	labelIndex := strings.Index(out, "Describe the changes you want:")
	pickerIndex := strings.Index(out, "src/main.go")
	submitIndex := strings.Index(out, "Submit")
	if labelIndex == -1 || pickerIndex == -1 || submitIndex == -1 {
		t.Fatalf("expected label, picker output, and submit button to be present")
	}
	if !(labelIndex < pickerIndex && pickerIndex < submitIndex) {
		t.Fatalf("expected picker output between change request and submit sections")
	}
}

func TestModel_EscClosesFilePickerInPlanRefineModal(t *testing.T) {
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
