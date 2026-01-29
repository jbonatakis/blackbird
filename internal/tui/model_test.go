package tui

import (
	"testing"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestVisibleNavigationRespectsExpanded(t *testing.T) {
	g := plan.WorkGraph{
		Items: map[string]plan.WorkItem{},
	}

	rootID := "A"
	childB := "B"
	childC := "C"
	childD := "D"

	g.Items[rootID] = plan.WorkItem{
		ID:       rootID,
		ChildIDs: []string{childB, childC},
		Status:   plan.StatusTodo,
	}
	g.Items[childB] = plan.WorkItem{
		ID:       childB,
		ParentID: &rootID,
		ChildIDs: []string{childD},
		Status:   plan.StatusTodo,
	}
	g.Items[childC] = plan.WorkItem{
		ID:       childC,
		ParentID: &rootID,
		Status:   plan.StatusTodo,
	}
	g.Items[childD] = plan.WorkItem{
		ID:       childD,
		ParentID: &childB,
		Status:   plan.StatusTodo,
	}

	model := Model{
		plan:          g,
		filterMode:    FilterModeAll,
		expandedItems: map[string]bool{},
		selectedID:    childB,
	}

	if next := model.nextVisibleItem(); next != childD {
		t.Fatalf("expected next to be %q, got %q", childD, next)
	}
	if prev := model.prevVisibleItem(); prev != rootID {
		t.Fatalf("expected prev to be %q, got %q", rootID, prev)
	}

	model.expandedItems[rootID] = false
	model.selectedID = childD
	model.ensureSelectionVisible()
	if model.selectedID != rootID {
		t.Fatalf("expected selection to snap to %q, got %q", rootID, model.selectedID)
	}
}

func TestFilterModeBlockedIncludesParents(t *testing.T) {
	g := plan.WorkGraph{
		Items: map[string]plan.WorkItem{},
	}

	rootID := "R"
	childA := "A"
	childB := "B"
	depID := "X"

	g.Items[rootID] = plan.WorkItem{
		ID:       rootID,
		ChildIDs: []string{childA, childB},
		Status:   plan.StatusTodo,
	}
	g.Items[childA] = plan.WorkItem{
		ID:       childA,
		ParentID: &rootID,
		Deps:     []string{depID},
		Status:   plan.StatusTodo,
	}
	g.Items[childB] = plan.WorkItem{
		ID:       childB,
		ParentID: &rootID,
		Status:   plan.StatusTodo,
	}
	g.Items[depID] = plan.WorkItem{
		ID:     depID,
		Status: plan.StatusTodo,
	}

	model := Model{
		plan:          g,
		filterMode:    FilterModeBlocked,
		expandedItems: map[string]bool{},
		selectedID:    childB,
	}

	model.ensureSelectionVisible()
	if model.selectedID != rootID {
		t.Fatalf("expected selection to move to %q, got %q", rootID, model.selectedID)
	}

	visible := model.visibleItemIDs()
	if len(visible) != 2 || visible[0] != rootID || visible[1] != childA {
		t.Fatalf("expected visible items [ %s %s ], got %#v", rootID, childA, visible)
	}
}
