package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestListenExecutionStageCmdReturnsStageMessages(t *testing.T) {
	stageCh := make(chan execution.ExecutionStageState, 1)
	stageCh <- execution.ExecutionStageState{
		Stage:          execution.ExecutionStageReviewing,
		TaskID:         "child-1",
		ReviewedTaskID: "parent-1",
	}

	cmd := listenExecutionStageCmd(stageCh)
	msg := cmd()
	typed, ok := msg.(executionStageMsg)
	if !ok {
		t.Fatalf("expected executionStageMsg, got %T", msg)
	}
	if typed.state.Stage != execution.ExecutionStageReviewing {
		t.Fatalf("typed.state.Stage = %q, want %q", typed.state.Stage, execution.ExecutionStageReviewing)
	}
	if typed.state.ReviewedTaskID != "parent-1" {
		t.Fatalf("typed.state.ReviewedTaskID = %q, want %q", typed.state.ReviewedTaskID, "parent-1")
	}
}

func TestModelUpdateExecutionStageMsgStoresState(t *testing.T) {
	model := Model{}

	updatedModel, _ := model.Update(executionStageMsg{
		state: execution.ExecutionStageState{
			Stage:          execution.ExecutionStageReviewing,
			TaskID:         "child-1",
			ReviewedTaskID: "parent-1",
		},
	})
	updated := updatedModel.(Model)

	if updated.executionState.Stage != execution.ExecutionStageReviewing {
		t.Fatalf("executionState.Stage = %q, want %q", updated.executionState.Stage, execution.ExecutionStageReviewing)
	}
	if updated.executionState.ReviewedTaskID != "parent-1" {
		t.Fatalf("executionState.ReviewedTaskID = %q, want %q", updated.executionState.ReviewedTaskID, "parent-1")
	}
}

func TestExecutionStageReviewingStateShowsBottomBarAndTreeReviewIndicators(t *testing.T) {
	model := executionStageRenderTestModel()

	updatedModel, _ := model.Update(executionStageMsg{
		state: execution.ExecutionStageState{
			Stage:          execution.ExecutionStageReviewing,
			TaskID:         "child-1",
			ReviewedTaskID: "parent-1",
		},
	})
	updated := updatedModel.(Model)

	bottomBar := stripANSI(RenderBottomBar(updated))
	if !strings.Contains(bottomBar, "Reviewing...") {
		t.Fatalf("expected reviewing status text in bottom bar, got %q", bottomBar)
	}
	if strings.Contains(bottomBar, "Executing...") {
		t.Fatalf("expected reviewing text to replace executing text, got %q", bottomBar)
	}

	tree := stripANSI(RenderTreeView(updated))
	if !strings.Contains(tree, reviewingRowMarker) {
		t.Fatalf("expected reviewing marker in tree view, got %q", tree)
	}
	parentLine := executionStageLineContaining(tree, "Parent Task")
	if !strings.Contains(parentLine, reviewingRowMarker) {
		t.Fatalf("expected reviewed parent row to include marker, got %q", parentLine)
	}
	childLine := executionStageLineContaining(tree, "Child Task")
	if strings.Contains(childLine, reviewingRowMarker) {
		t.Fatalf("expected non-reviewed child row to omit marker, got %q", childLine)
	}
}

func TestExecutionStageExecutingStatesDoNotShowReviewIndicatorsWhenParentReviewDisabled(t *testing.T) {
	model := executionStageRenderTestModel()

	// Parent review disabled emits executing stages only.
	states := []execution.ExecutionStageState{
		{Stage: execution.ExecutionStageExecuting, TaskID: "child-1"},
		{Stage: execution.ExecutionStageExecuting, TaskID: "other-1"},
	}

	for i, state := range states {
		updatedModel, _ := model.Update(executionStageMsg{state: state})
		model = updatedModel.(Model)

		bottomBar := stripANSI(RenderBottomBar(model))
		if strings.Contains(bottomBar, "Reviewing...") {
			t.Fatalf("state[%d]: expected no reviewing status text, got %q", i, bottomBar)
		}
		if !strings.Contains(bottomBar, "Executing...") {
			t.Fatalf("state[%d]: expected executing status text, got %q", i, bottomBar)
		}

		tree := stripANSI(RenderTreeView(model))
		if strings.Contains(tree, reviewingRowMarker) {
			t.Fatalf("state[%d]: expected no reviewing marker in tree view, got %q", i, tree)
		}
	}
}

func executionStageRenderTestModel() Model {
	now := time.Date(2026, 2, 11, 14, 0, 0, 0, time.UTC)
	parentID := "parent-1"
	childID := "child-1"
	otherID := "other-1"

	childParent := parentID
	return Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				parentID: {
					ID:        parentID,
					Title:     "Parent Task",
					Status:    plan.StatusTodo,
					ChildIDs:  []string{childID},
					CreatedAt: now,
					UpdatedAt: now,
				},
				childID: {
					ID:        childID,
					Title:     "Child Task",
					Status:    plan.StatusTodo,
					ParentID:  &childParent,
					CreatedAt: now,
					UpdatedAt: now,
				},
				otherID: {
					ID:        otherID,
					Title:     "Other Task",
					Status:    plan.StatusTodo,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		viewMode:         ViewModeMain,
		planExists:       true,
		actionInProgress: true,
		actionName:       "Executing...",
		selectedID:       parentID,
		filterMode:       FilterModeAll,
		expandedItems: map[string]bool{
			parentID: true,
		},
		windowWidth:  160,
		windowHeight: 24,
	}
}

func executionStageLineContaining(rendered string, needle string) string {
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}
