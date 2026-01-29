package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestConfirmOverwriteEmptyPlan(t *testing.T) {
	// Empty plan should skip confirmation and go straight to generation modal
	m := NewModel(plan.NewEmptyWorkGraph())
	m.windowWidth = 80
	m.windowHeight = 24

	// Press 'g' to trigger plan generation
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = result.(Model)

	// Should go directly to plan generation modal (not confirmation modal)
	if m.actionMode != ActionModeGeneratePlan {
		t.Errorf("Expected ActionModeGeneratePlan, got %v", m.actionMode)
	}
	if m.planGenerateForm == nil {
		t.Error("Expected planGenerateForm to be initialized")
	}
}

func TestConfirmOverwriteWithExistingPlan(t *testing.T) {
	// Create a plan with items
	g := plan.NewEmptyWorkGraph()
	g.Items = map[string]plan.WorkItem{
		"task-1": {ID: "task-1", Title: "Task 1", Status: plan.StatusTodo},
		"task-2": {ID: "task-2", Title: "Task 2", Status: plan.StatusTodo},
	}

	m := NewModel(g)
	m.windowWidth = 80
	m.windowHeight = 24

	// Press 'g' to trigger plan generation
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = result.(Model)

	// Should show confirmation modal first
	if m.actionMode != ActionModeConfirmOverwrite {
		t.Errorf("Expected ActionModeConfirmOverwrite, got %v", m.actionMode)
	}
}

func TestConfirmOverwriteDecline(t *testing.T) {
	// Create a plan with items
	g := plan.NewEmptyWorkGraph()
	g.Items = map[string]plan.WorkItem{
		"task-1": {ID: "task-1", Title: "Task 1", Status: plan.StatusTodo},
	}

	m := NewModel(g)
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionMode = ActionModeConfirmOverwrite

	// Press 'n' to decline
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = result.(Model)

	// Should return to normal mode without showing generation modal
	if m.actionMode != ActionModeNone {
		t.Errorf("Expected ActionModeNone, got %v", m.actionMode)
	}
	if m.planGenerateForm != nil {
		t.Error("Expected planGenerateForm to be nil after declining")
	}
}

func TestConfirmOverwriteAccept(t *testing.T) {
	// Create a plan with items
	g := plan.NewEmptyWorkGraph()
	g.Items = map[string]plan.WorkItem{
		"task-1": {ID: "task-1", Title: "Task 1", Status: plan.StatusTodo},
	}

	m := NewModel(g)
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionMode = ActionModeConfirmOverwrite

	// Press 'y' to accept
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = result.(Model)

	// Should proceed to generation modal
	if m.actionMode != ActionModeGeneratePlan {
		t.Errorf("Expected ActionModeGeneratePlan, got %v", m.actionMode)
	}
	if m.planGenerateForm == nil {
		t.Error("Expected planGenerateForm to be initialized after accepting")
	}
}

func TestConfirmOverwriteEscape(t *testing.T) {
	// Create a plan with items
	g := plan.NewEmptyWorkGraph()
	g.Items = map[string]plan.WorkItem{
		"task-1": {ID: "task-1", Title: "Task 1", Status: plan.StatusTodo},
	}

	m := NewModel(g)
	m.windowWidth = 80
	m.windowHeight = 24
	m.actionMode = ActionModeConfirmOverwrite

	// Press ESC to cancel
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = result.(Model)

	// Should return to normal mode
	if m.actionMode != ActionModeNone {
		t.Errorf("Expected ActionModeNone, got %v", m.actionMode)
	}
	if m.planGenerateForm != nil {
		t.Error("Expected planGenerateForm to be nil after escape")
	}
}
