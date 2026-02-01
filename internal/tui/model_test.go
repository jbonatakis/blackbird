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

func TestNextVisibleItem(t *testing.T) {
	g := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:       "task-1",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
			"task-2": {
				ID:       "task-2",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
			"task-3": {
				ID:       "task-3",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
		},
	}

	model := Model{
		plan:          g,
		selectedID:    "task-1",
		filterMode:    FilterModeAll,
		expandedItems: map[string]bool{},
	}

	// From first to second
	next := model.nextVisibleItem()
	if next != "task-2" {
		t.Errorf("nextVisibleItem() from task-1 = %q, want task-2", next)
	}

	// From second to third
	model.selectedID = "task-2"
	next = model.nextVisibleItem()
	if next != "task-3" {
		t.Errorf("nextVisibleItem() from task-2 = %q, want task-3", next)
	}

	// From last item stays at last
	model.selectedID = "task-3"
	next = model.nextVisibleItem()
	if next != "task-3" {
		t.Errorf("nextVisibleItem() from task-3 = %q, want task-3 (stay at end)", next)
	}
}

func TestPrevVisibleItem(t *testing.T) {
	g := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:       "task-1",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
			"task-2": {
				ID:       "task-2",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
			"task-3": {
				ID:       "task-3",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
		},
	}

	model := Model{
		plan:          g,
		selectedID:    "task-3",
		filterMode:    FilterModeAll,
		expandedItems: map[string]bool{},
	}

	// From third to second
	prev := model.prevVisibleItem()
	if prev != "task-2" {
		t.Errorf("prevVisibleItem() from task-3 = %q, want task-2", prev)
	}

	// From second to first
	model.selectedID = "task-2"
	prev = model.prevVisibleItem()
	if prev != "task-1" {
		t.Errorf("prevVisibleItem() from task-2 = %q, want task-1", prev)
	}

	// From first item stays at first
	model.selectedID = "task-1"
	prev = model.prevVisibleItem()
	if prev != "task-1" {
		t.Errorf("prevVisibleItem() from task-1 = %q, want task-1 (stay at start)", prev)
	}
}

func TestToggleExpanded(t *testing.T) {
	model := Model{
		expandedItems: map[string]bool{},
	}

	// Initially expanded (default)
	if !isExpanded(model, "task-1") {
		t.Errorf("isExpanded() initially = false, want true")
	}

	// Toggle to collapsed
	model.toggleExpanded("task-1")
	if isExpanded(model, "task-1") {
		t.Errorf("isExpanded() after first toggle = true, want false")
	}

	// Toggle back to expanded
	model.toggleExpanded("task-1")
	if !isExpanded(model, "task-1") {
		t.Errorf("isExpanded() after second toggle = false, want true")
	}
}

func TestIsParent(t *testing.T) {
	parentID := "parent"
	g := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			parentID: {
				ID:       parentID,
				Status:   plan.StatusTodo,
				ChildIDs: []string{"child"},
			},
			"child": {
				ID:       "child",
				Status:   plan.StatusTodo,
				ParentID: &parentID,
				ChildIDs: []string{},
			},
			"leaf": {
				ID:       "leaf",
				Status:   plan.StatusTodo,
				ChildIDs: []string{},
			},
		},
	}

	model := Model{
		plan: g,
	}

	if !model.isParent("parent") {
		t.Errorf("isParent(parent) = false, want true")
	}

	if model.isParent("leaf") {
		t.Errorf("isParent(leaf) = true, want false")
	}

	if model.isParent("nonexistent") {
		t.Errorf("isParent(nonexistent) = true, want false")
	}
}

func TestNextFilterMode(t *testing.T) {
	tests := []struct {
		current FilterMode
		want    FilterMode
	}{
		{FilterModeAll, FilterModeReady},
		{FilterModeReady, FilterModeBlocked},
		{FilterModeBlocked, FilterModeAll},
	}

	for _, tt := range tests {
		got := nextFilterMode(tt.current)
		if got != tt.want {
			t.Errorf("nextFilterMode(%v) = %v, want %v", tt.current, got, tt.want)
		}
	}
}

func TestSplitPaneWidths(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		wantLeft  int
		wantRight int
	}{
		{"zero width", 0, 0, 0},
		{"small width", 60, 24, 32},
		{"medium width", 120, 38, 78},
		{"large width", 180, 58, 118},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, right := splitPaneWidths(tt.total)
			if left != tt.wantLeft || right != tt.wantRight {
				t.Errorf("splitPaneWidths(%d) = (%d, %d), want (%d, %d)", tt.total, left, right, tt.wantLeft, tt.wantRight)
			}
			// Rendered width is (left+2)+(right+2) = total
			if tt.total > 0 && left+right+4 != tt.total {
				t.Errorf("splitPaneWidths(%d) left+right+4 = %d, want %d", tt.total, left+right+4, tt.total)
			}
		})
	}
}

func TestDetailPageSize(t *testing.T) {
	tests := []struct {
		name         string
		windowHeight int
		wantPageSize int
	}{
		{"zero height", 0, 0},
		{"single line", 1, 0},
		{"small window", 10, 5},
		{"medium window", 30, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				windowHeight: tt.windowHeight,
			}
			got := model.detailPageSize()
			if got != tt.wantPageSize {
				t.Errorf("detailPageSize() with height %d = %d, want %d", tt.windowHeight, got, tt.wantPageSize)
			}
		})
	}
}
