package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func testWorkGraph() plan.WorkGraph {
	return plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"test": {ID: "test", Status: plan.StatusTodo},
		},
	}
}

func TestTabModeToggle(t *testing.T) {
	model := NewModel(testWorkGraph())
	model.viewMode = ViewModeMain // tab toggle only applies in main view

	// Initial state should be TabDetails
	if model.tabMode != TabDetails {
		t.Fatalf("expected initial tab mode to be TabDetails, got %v", model.tabMode)
	}

	// Press 't' to switch to execution tab
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updated.(Model)

	if model.tabMode != TabExecution {
		t.Fatalf("expected tab mode to be TabExecution after 't' key, got %v", model.tabMode)
	}

	// Press 't' again to switch back to details tab
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updated.(Model)

	if model.tabMode != TabDetails {
		t.Fatalf("expected tab mode to be TabDetails after second 't' key, got %v", model.tabMode)
	}
}

func TestTabModeResetsDetailOffset(t *testing.T) {
	model := NewModel(testWorkGraph())
	model.viewMode = ViewModeMain
	model.detailOffset = 10

	// Press 't' to switch tabs
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updated.(Model)

	if model.detailOffset != 0 {
		t.Fatalf("expected detail offset to be reset to 0 after tab switch, got %d", model.detailOffset)
	}
}

func TestTabModeIgnoredDuringAction(t *testing.T) {
	model := NewModel(testWorkGraph())
	model.actionInProgress = true

	// Press 't' while action is in progress
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updated.(Model)

	// Tab mode should remain as TabDetails
	if model.tabMode != TabDetails {
		t.Fatalf("expected tab mode to remain TabDetails when action is in progress, got %v", model.tabMode)
	}
}
