package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestReviewCheckpointFormCreation(t *testing.T) {
	run := testDecisionRun()
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())

	if form.mode != ReviewCheckpointChooseAction {
		t.Fatalf("expected mode ReviewCheckpointChooseAction, got %v", form.mode)
	}
	if form.selectedAction != 0 {
		t.Fatalf("expected selected action 0, got %d", form.selectedAction)
	}
	if form.task.ID != "task-1" {
		t.Fatalf("expected task id task-1, got %q", form.task.ID)
	}
	if form.task.Title != "Review task" {
		t.Fatalf("expected task title from context, got %q", form.task.Title)
	}
}

func TestReviewCheckpointNavigation(t *testing.T) {
	run := testDecisionRun()
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())

	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != 1 {
		t.Fatalf("expected action 1 after down, got %d", form.selectedAction)
	}
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if form.selectedAction != 2 {
		t.Fatalf("expected action 2 after down, got %d", form.selectedAction)
	}
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if form.selectedAction != 1 {
		t.Fatalf("expected action 1 after up, got %d", form.selectedAction)
	}
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if form.selectedAction != 3 {
		t.Fatalf("expected action 3 after quick select, got %d", form.selectedAction)
	}
}

func TestReviewCheckpointSwitchToRequestChanges(t *testing.T) {
	run := testDecisionRun()
	m := Model{}
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())
	form.selectedAction = 2
	m.reviewCheckpointForm = &form
	m.actionMode = ActionModeReviewCheckpoint

	m, _ = HandleReviewCheckpointKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.reviewCheckpointForm == nil {
		t.Fatal("expected reviewCheckpointForm to remain")
	}
	if m.reviewCheckpointForm.mode != ReviewCheckpointRequestChanges {
		t.Fatalf("expected request changes mode, got %v", m.reviewCheckpointForm.mode)
	}
}

func TestReviewCheckpointForm_FilePickerOpenAndQueryUpdates(t *testing.T) {
	run := testDecisionRun()
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())
	form.mode = ReviewCheckpointRequestChanges
	form.changeRequest.Focus()

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	form = updated

	if !form.filePicker.Open {
		t.Fatalf("expected file picker to open on @")
	}
	if form.filePicker.ActiveField != reviewCheckpointPickerChangeRequest {
		t.Fatalf("expected active field %q, got %q", reviewCheckpointPickerChangeRequest, form.filePicker.ActiveField)
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

func TestReviewCheckpointForm_FilePickerEnterInsertsSelection(t *testing.T) {
	run := testDecisionRun()
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())
	form.mode = ReviewCheckpointRequestChanges
	form.changeRequest.Focus()
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}

	if !form.OpenFilePicker(anchor) {
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

func TestRenderReviewCheckpointModalIncludesSummary(t *testing.T) {
	run := testDecisionRun()
	m := Model{windowWidth: 100, windowHeight: 40}
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())

	out := RenderReviewCheckpointModal(m, form)
	if !strings.Contains(out, "Task review checkpoint") {
		t.Fatalf("expected title in modal output, got %q", out)
	}
	if !strings.Contains(out, "Review summary") {
		t.Fatalf("expected review summary label, got %q", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("expected file name in summary, got %q", out)
	}
	if !strings.Contains(out, "Diffstat") {
		t.Fatalf("expected diffstat label in summary, got %q", out)
	}
}

func TestRenderReviewCheckpointModal_IncludesFilePickerWhenOpen(t *testing.T) {
	run := testDecisionRun()
	m := Model{windowWidth: 90, windowHeight: 30}
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())
	form.mode = ReviewCheckpointRequestChanges
	form.SetSize(90, 30)
	form.changeRequest.SetValue("See @src/")
	anchor := FilePickerAnchor{Start: runeIndexInString(form.changeRequest.Value(), "@")}
	if !form.OpenFilePicker(anchor) {
		t.Fatalf("expected OpenFilePicker to return true")
	}
	form.filePicker.Query = "src/"
	form.filePicker.Matches = []string{"src/main.go", "src/other.go"}
	form.filePicker.Selected = 0

	out := RenderReviewCheckpointModal(m, form)
	if !strings.Contains(out, "src/main.go") {
		t.Fatalf("expected picker output to include match, got:\n%s", out)
	}

	labelIndex := strings.Index(out, "Change request:")
	pickerIndex := strings.Index(out, "src/main.go")
	helpIndex := strings.Index(out, "ctrl+s")
	if labelIndex == -1 || pickerIndex == -1 || helpIndex == -1 {
		t.Fatalf("expected label, picker output, and help text to be present")
	}
	if !(labelIndex < pickerIndex && pickerIndex < helpIndex) {
		t.Fatalf("expected picker output between change request and help text")
	}
}

func TestReviewCheckpointEscBackPreservesChangeRequest(t *testing.T) {
	run := testDecisionRun()
	form := NewReviewCheckpointForm(run, plan.NewEmptyWorkGraph())
	form.mode = ReviewCheckpointRequestChanges
	form.changeRequest.Focus()
	form.changeRequest.SetValue("Please update tests")

	m := Model{
		actionMode:           ActionModeReviewCheckpoint,
		reviewCheckpointForm: &form,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.reviewCheckpointForm == nil {
		t.Fatal("expected reviewCheckpointForm to remain open")
	}
	if m.reviewCheckpointForm.mode != ReviewCheckpointChooseAction {
		t.Fatalf("expected mode ReviewCheckpointChooseAction, got %v", m.reviewCheckpointForm.mode)
	}
	if got := m.reviewCheckpointForm.changeRequest.Value(); got != "Please update tests" {
		t.Fatalf("expected change request to be preserved, got %q", got)
	}
}

func TestRenderActionRequiredBanner(t *testing.T) {
	run := testDecisionRun()
	m := Model{
		windowWidth: 80,
		runData: map[string]execution.RunRecord{
			"task-1": run,
		},
		plan: plan.NewEmptyWorkGraph(),
	}

	out := RenderActionRequiredBanner(m)
	if !strings.Contains(out, "ACTION REQUIRED") {
		t.Fatalf("expected action required banner, got %q", out)
	}
	if !strings.Contains(out, "task-1") {
		t.Fatalf("expected task id in banner, got %q", out)
	}
}

func testDecisionRun() execution.RunRecord {
	now := time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC)
	return execution.RunRecord{
		ID:               "run-1",
		TaskID:           "task-1",
		StartedAt:        now.Add(-2 * time.Minute),
		CompletedAt:      ptrTime(now),
		Status:           execution.RunStatusSuccess,
		DecisionRequired: true,
		DecisionState:    execution.DecisionStatePending,
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "task-1",
				Title: "Review task",
			},
		},
		ReviewSummary: &execution.ReviewSummary{
			Files:    []string{"main.go"},
			DiffStat: "main.go | 2 + -",
			Snippets: []execution.ReviewSnippet{{File: "main.go", Snippet: "+foo\n-bar"}},
		},
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
