package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRenderTreeView_EmptyPlan(t *testing.T) {
	model := Model{
		plan: plan.NewEmptyWorkGraph(),
	}

	result := RenderTreeView(model)
	if !strings.Contains(result, "No items") {
		t.Errorf("expected 'No items' message for empty plan, got %q", result)
	}
}

func TestRenderTreeView_SingleItem(t *testing.T) {
	now := time.Now()
	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: 1,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID:        "task-1",
					Title:     "Test Task",
					Status:    plan.StatusTodo,
					ParentID:  nil,
					ChildIDs:  []string{},
					Deps:      []string{},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		selectedID:    "task-1",
		filterMode:    FilterModeAll,
		expandedItems: map[string]bool{},
	}

	result := RenderTreeView(model)
	if !strings.Contains(result, "task-1") {
		t.Errorf("expected tree to contain task-1, got %q", result)
	}
	if !strings.Contains(result, "Test Task") {
		t.Errorf("expected tree to contain 'Test Task', got %q", result)
	}
}

func TestRenderTreeView_ParentChildHierarchy(t *testing.T) {
	now := time.Now()
	parentID := "parent"
	childID := "child"

	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: 1,
			Items: map[string]plan.WorkItem{
				parentID: {
					ID:        parentID,
					Title:     "Parent Task",
					Status:    plan.StatusTodo,
					ParentID:  nil,
					ChildIDs:  []string{childID},
					Deps:      []string{},
					CreatedAt: now,
					UpdatedAt: now,
				},
				childID: {
					ID:        childID,
					Title:     "Child Task",
					Status:    plan.StatusTodo,
					ParentID:  &parentID,
					ChildIDs:  []string{},
					Deps:      []string{},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		selectedID:    parentID,
		filterMode:    FilterModeAll,
		expandedItems: map[string]bool{},
	}

	result := RenderTreeView(model)
	if !strings.Contains(result, "parent") {
		t.Errorf("expected tree to contain parent, got %q", result)
	}
	if !strings.Contains(result, "child") {
		t.Errorf("expected tree to contain child, got %q", result)
	}

	// Parent should have expansion indicator
	if !strings.Contains(result, "▼") && !strings.Contains(result, "▶") {
		t.Errorf("expected tree to contain expansion indicator, got %q", result)
	}
}

func TestRenderTreeView_CollapsedParent(t *testing.T) {
	now := time.Now()
	parentID := "parent"
	childID := "child"

	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: 1,
			Items: map[string]plan.WorkItem{
				parentID: {
					ID:        parentID,
					Title:     "Parent Task",
					Status:    plan.StatusTodo,
					ParentID:  nil,
					ChildIDs:  []string{childID},
					Deps:      []string{},
					CreatedAt: now,
					UpdatedAt: now,
				},
				childID: {
					ID:        childID,
					Title:     "Child Task",
					Status:    plan.StatusTodo,
					ParentID:  &parentID,
					ChildIDs:  []string{},
					Deps:      []string{},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		selectedID: parentID,
		filterMode: FilterModeAll,
		expandedItems: map[string]bool{
			parentID: false,
		},
	}

	result := RenderTreeView(model)
	if !strings.Contains(result, "parent") {
		t.Errorf("expected tree to contain parent, got %q", result)
	}
	// Child should not be visible when parent is collapsed
	lines := strings.Split(result, "\n")
	childVisible := false
	for _, line := range lines {
		if strings.Contains(line, "child") && strings.Contains(line, "Child Task") {
			childVisible = true
			break
		}
	}
	if childVisible {
		t.Errorf("expected child to be hidden when parent is collapsed, got %q", result)
	}
}

func TestFilterMatch(t *testing.T) {
	tests := []struct {
		name      string
		mode      FilterMode
		readiness string
		wantMatch bool
	}{
		{"FilterModeAll matches READY", FilterModeAll, "READY", true},
		{"FilterModeAll matches BLOCKED", FilterModeAll, "BLOCKED", true},
		{"FilterModeAll matches DONE", FilterModeAll, "DONE", true},
		{"FilterModeReady matches READY", FilterModeReady, "READY", true},
		{"FilterModeReady rejects BLOCKED", FilterModeReady, "BLOCKED", false},
		{"FilterModeReady rejects DONE", FilterModeReady, "DONE", false},
		{"FilterModeBlocked matches BLOCKED", FilterModeBlocked, "BLOCKED", true},
		{"FilterModeBlocked rejects READY", FilterModeBlocked, "READY", false},
		{"FilterModeBlocked rejects DONE", FilterModeBlocked, "DONE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterMatch(tt.mode, tt.readiness)
			if got != tt.wantMatch {
				t.Errorf("filterMatch(%v, %q) = %v, want %v", tt.mode, tt.readiness, got, tt.wantMatch)
			}
		})
	}
}

func TestRootIDs(t *testing.T) {
	now := time.Now()
	parentID := "parent"
	childID := "child"
	orphanID := "orphan"

	tests := []struct {
		name     string
		graph    plan.WorkGraph
		wantRoot []string
	}{
		{
			name: "empty graph",
			graph: plan.WorkGraph{
				SchemaVersion: 1,
				Items:         map[string]plan.WorkItem{},
			},
			wantRoot: []string{},
		},
		{
			name: "single item with nil parent",
			graph: plan.WorkGraph{
				SchemaVersion: 1,
				Items: map[string]plan.WorkItem{
					parentID: {
						ID:        parentID,
						Title:     "Root",
						ParentID:  nil,
						ChildIDs:  []string{},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
			wantRoot: []string{parentID},
		},
		{
			name: "parent with child",
			graph: plan.WorkGraph{
				SchemaVersion: 1,
				Items: map[string]plan.WorkItem{
					parentID: {
						ID:        parentID,
						Title:     "Parent",
						ParentID:  nil,
						ChildIDs:  []string{childID},
						CreatedAt: now,
						UpdatedAt: now,
					},
					childID: {
						ID:        childID,
						Title:     "Child",
						ParentID:  &parentID,
						ChildIDs:  []string{},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
			wantRoot: []string{parentID},
		},
		{
			name: "orphan with missing parent",
			graph: plan.WorkGraph{
				SchemaVersion: 1,
				Items: map[string]plan.WorkItem{
					orphanID: {
						ID:        orphanID,
						Title:     "Orphan",
						ParentID:  strPtr("missing-parent"),
						ChildIDs:  []string{},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
			wantRoot: []string{orphanID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rootIDs(tt.graph)
			if len(got) != len(tt.wantRoot) {
				t.Errorf("rootIDs() = %v, want %v", got, tt.wantRoot)
				return
			}
			for i := range got {
				if got[i] != tt.wantRoot[i] {
					t.Errorf("rootIDs()[%d] = %v, want %v", i, got[i], tt.wantRoot[i])
				}
			}
		})
	}
}

func TestIsExpanded(t *testing.T) {
	tests := []struct {
		name          string
		expandedItems map[string]bool
		id            string
		wantExpanded  bool
	}{
		{
			name:          "nil map defaults to expanded",
			expandedItems: nil,
			id:            "task-1",
			wantExpanded:  true,
		},
		{
			name:          "empty map defaults to expanded",
			expandedItems: map[string]bool{},
			id:            "task-1",
			wantExpanded:  true,
		},
		{
			name: "explicitly expanded",
			expandedItems: map[string]bool{
				"task-1": true,
			},
			id:           "task-1",
			wantExpanded: true,
		},
		{
			name: "explicitly collapsed",
			expandedItems: map[string]bool{
				"task-1": false,
			},
			id:           "task-1",
			wantExpanded: false,
		},
		{
			name: "missing key defaults to expanded",
			expandedItems: map[string]bool{
				"other": false,
			},
			id:           "task-1",
			wantExpanded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				expandedItems: tt.expandedItems,
			}
			got := isExpanded(model, tt.id)
			if got != tt.wantExpanded {
				t.Errorf("isExpanded() = %v, want %v", got, tt.wantExpanded)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
