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
