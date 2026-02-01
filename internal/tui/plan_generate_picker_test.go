package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPlanGenerateForm_OpenFilePickerTracksField(t *testing.T) {
	form := NewPlanGenerateForm()
	anchor := FilePickerAnchor{Start: 4}

	opened := form.OpenFilePicker(FieldConstraints, anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	if !form.filePicker.Open {
		t.Fatalf("expected file picker to be open")
	}
	if form.filePicker.ActiveField != planGeneratePickerConstraints {
		t.Fatalf("expected active field %q, got %q", planGeneratePickerConstraints, form.filePicker.ActiveField)
	}
	if form.constraintsAnchor != anchor {
		t.Fatalf("expected constraints anchor to be stored")
	}
}

func TestPlanGenerateForm_ApplyFilePickerSelection(t *testing.T) {
	form := NewPlanGenerateForm()
	form.description.SetValue("See @src/ for details")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}

	opened := form.OpenFilePicker(FieldDescription, anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"

	applied := form.ApplyFilePickerSelection("src/main.go")
	if !applied {
		t.Fatalf("expected ApplyFilePickerSelection to return true")
	}

	expected := "See @src/main.go for details"
	if form.description.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, form.description.Value())
	}
	if form.filePicker.Open {
		t.Fatalf("expected file picker to be closed after apply")
	}
}

func TestPlanGenerateForm_FilePickerOpenAndQueryUpdates(t *testing.T) {
	form := NewPlanGenerateForm()

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	form = updated

	if !form.filePicker.Open {
		t.Fatalf("expected file picker to open on @")
	}
	if form.filePicker.ActiveField != planGeneratePickerDescription {
		t.Fatalf("expected active field %q, got %q", planGeneratePickerDescription, form.filePicker.ActiveField)
	}
	if form.description.Value() != "@" {
		t.Fatalf("expected description to contain @, got %q", form.description.Value())
	}

	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	form = updated
	if form.filePicker.Query != "s" {
		t.Fatalf("expected query %q, got %q", "s", form.filePicker.Query)
	}
	if form.description.Value() != "@s" {
		t.Fatalf("expected description %q, got %q", "@s", form.description.Value())
	}

	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	form = updated
	if form.filePicker.Query != "" {
		t.Fatalf("expected query to be cleared, got %q", form.filePicker.Query)
	}
	if form.description.Value() != "@" {
		t.Fatalf("expected description %q, got %q", "@", form.description.Value())
	}
}

func TestPlanGenerateForm_FilePickerTabClosesAndMovesFocus(t *testing.T) {
	form := NewPlanGenerateForm()
	form.description.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}

	opened := form.OpenFilePicker(FieldDescription, anchor)
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
	if form.focusedField != FieldConstraints {
		t.Fatalf("expected focus to move to constraints, got %v", form.focusedField)
	}
	if !form.constraints.Focused() {
		t.Fatalf("expected constraints to be focused")
	}
}

func TestPlanGenerateForm_FilePickerShiftTabClosesAndMovesFocus(t *testing.T) {
	form := NewPlanGenerateForm()
	form = form.focusNext() // Constraints
	form.constraints.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.constraints.Value(), "@")}

	opened := form.OpenFilePicker(FieldConstraints, anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	form = updated

	if form.filePicker.Open {
		t.Fatalf("expected file picker to close on shift+tab")
	}
	if form.focusedField != FieldDescription {
		t.Fatalf("expected focus to move to description, got %v", form.focusedField)
	}
	if !form.description.Focused() {
		t.Fatalf("expected description to be focused")
	}
}

func TestPlanGenerateForm_FilePickerEnterInsertsSelection(t *testing.T) {
	form := NewPlanGenerateForm()
	form.description.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.description.Value(), "@")}

	opened := form.OpenFilePicker(FieldDescription, anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updated

	expected := "See @src/main.go"
	if form.description.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, form.description.Value())
	}
	if form.filePicker.Open {
		t.Fatalf("expected file picker to be closed after insert")
	}
}

func TestPlanGenerateForm_FilePickerOpenAndQueryUpdatesGranularity(t *testing.T) {
	form := NewPlanGenerateForm()
	form = form.focusNext() // Constraints
	form = form.focusNext() // Granularity

	if form.focusedField != FieldGranularity {
		t.Fatalf("expected focus to be on granularity, got %v", form.focusedField)
	}

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	form = updated

	if !form.filePicker.Open {
		t.Fatalf("expected file picker to open on @")
	}
	if form.filePicker.ActiveField != planGeneratePickerGranularity {
		t.Fatalf("expected active field %q, got %q", planGeneratePickerGranularity, form.filePicker.ActiveField)
	}
	if form.granularity.Value() != "@" {
		t.Fatalf("expected granularity to contain @, got %q", form.granularity.Value())
	}

	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	form = updated
	if form.filePicker.Query != "s" {
		t.Fatalf("expected query %q, got %q", "s", form.filePicker.Query)
	}
	if form.granularity.Value() != "@s" {
		t.Fatalf("expected granularity %q, got %q", "@s", form.granularity.Value())
	}

	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	form = updated
	if form.filePicker.Query != "" {
		t.Fatalf("expected query to be cleared, got %q", form.filePicker.Query)
	}
	if form.granularity.Value() != "@" {
		t.Fatalf("expected granularity %q, got %q", "@", form.granularity.Value())
	}
}

func TestPlanGenerateForm_FilePickerTabClosesGranularityMovesFocus(t *testing.T) {
	form := NewPlanGenerateForm()
	form = form.focusNext() // Constraints
	form = form.focusNext() // Granularity
	form.granularity.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.granularity.Value(), "@")}

	opened := form.OpenFilePicker(FieldGranularity, anchor)
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
	if form.focusedField != FieldSubmit {
		t.Fatalf("expected focus to move to submit, got %v", form.focusedField)
	}
}

func TestPlanGenerateForm_FilePickerEnterInsertsGranularitySelection(t *testing.T) {
	form := NewPlanGenerateForm()
	form = form.focusNext() // Constraints
	form = form.focusNext() // Granularity
	form.granularity.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.granularity.Value(), "@")}

	opened := form.OpenFilePicker(FieldGranularity, anchor)
	if !opened {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go"}
	form.filePicker.Selected = 0

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updated

	expected := "See @src/main.go"
	if form.granularity.Value() != expected {
		t.Fatalf("expected %q, got %q", expected, form.granularity.Value())
	}
	if form.filePicker.Open {
		t.Fatalf("expected file picker to be closed after insert")
	}
}

func runeIndexInString(value string, substr string) int {
	idx := strings.Index(value, substr)
	if idx < 0 {
		return -1
	}
	return utf8.RuneCountInString(value[:idx])
}
