package execution

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunExecuteStageTransitionsSingleTaskParentReviewFailure(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	parentID := "parent-1"
	childID := "child-1"

	parent := makeItem(parentID, plan.StatusTodo)
	parent.ChildIDs = []string{childID}
	parent.AcceptanceCriteria = []string{"Parent criteria must hold for child output."}

	child := makeItem(childID, plan.StatusTodo)
	childParent := parentID
	child.ParentID = &childParent

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	var states []ExecutionStageState
	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		ParentReviewEnabled: true,
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":false,"resumeTaskIds":["child-1"],"feedbackForResume":"Fix child output before continuing."}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
		OnStateChange: func(state ExecutionStageState) {
			states = append(states, state)
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonParentReviewRequired {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonParentReviewRequired)
	}

	if len(states) != 3 {
		t.Fatalf("state transition count = %d, want 3 (%#v)", len(states), states)
	}

	if states[0].Stage != ExecutionStageExecuting {
		t.Fatalf("states[0].Stage = %q, want %q", states[0].Stage, ExecutionStageExecuting)
	}
	if states[0].TaskID != childID {
		t.Fatalf("states[0].TaskID = %q, want %q", states[0].TaskID, childID)
	}
	if states[0].ReviewedTaskID != "" {
		t.Fatalf("states[0].ReviewedTaskID = %q, want empty", states[0].ReviewedTaskID)
	}

	if states[1].Stage != ExecutionStageReviewing {
		t.Fatalf("states[1].Stage = %q, want %q", states[1].Stage, ExecutionStageReviewing)
	}
	if states[1].ReviewedTaskID != parentID {
		t.Fatalf("states[1].ReviewedTaskID = %q, want %q", states[1].ReviewedTaskID, parentID)
	}

	if states[2].Stage != ExecutionStagePostReview {
		t.Fatalf("states[2].Stage = %q, want %q", states[2].Stage, ExecutionStagePostReview)
	}
	if states[2].ReviewedTaskID != "" {
		t.Fatalf("states[2].ReviewedTaskID = %q, want empty", states[2].ReviewedTaskID)
	}
}

func TestRunExecuteStageTransitionsMultiTaskParentReviewPass(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	rootID := "root-parent"
	midID := "mid-parent"
	childID := "a-child"
	otherID := "z-other"

	root := makeItem(rootID, plan.StatusTodo)
	root.ChildIDs = []string{midID}
	root.AcceptanceCriteria = []string{"Root parent criteria."}

	mid := makeItem(midID, plan.StatusTodo)
	midParent := rootID
	mid.ParentID = &midParent
	mid.ChildIDs = []string{childID}
	mid.AcceptanceCriteria = []string{"Mid parent criteria."}

	child := makeItem(childID, plan.StatusTodo)
	childParent := midID
	child.ParentID = &childParent

	other := makeItem(otherID, plan.StatusTodo)

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			rootID:  root,
			midID:   mid,
			childID: child,
			otherID: other,
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	var states []ExecutionStageState
	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		ParentReviewEnabled: true,
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":true}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
		OnStateChange: func(state ExecutionStageState) {
			states = append(states, state)
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonCompleted {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonCompleted)
	}

	if len(states) != 6 {
		t.Fatalf("state transition count = %d, want 6 (%#v)", len(states), states)
	}

	wantStages := []ExecutionStage{
		ExecutionStageExecuting,
		ExecutionStageReviewing,
		ExecutionStagePostReview,
		ExecutionStageReviewing,
		ExecutionStagePostReview,
		ExecutionStageExecuting,
	}
	for i, want := range wantStages {
		if states[i].Stage != want {
			t.Fatalf("states[%d].Stage = %q, want %q", i, states[i].Stage, want)
		}
	}

	if states[0].TaskID != childID {
		t.Fatalf("states[0].TaskID = %q, want %q", states[0].TaskID, childID)
	}
	if states[1].ReviewedTaskID != midID {
		t.Fatalf("states[1].ReviewedTaskID = %q, want %q", states[1].ReviewedTaskID, midID)
	}
	if states[2].ReviewedTaskID != "" {
		t.Fatalf("states[2].ReviewedTaskID = %q, want empty", states[2].ReviewedTaskID)
	}
	if states[3].ReviewedTaskID != rootID {
		t.Fatalf("states[3].ReviewedTaskID = %q, want %q", states[3].ReviewedTaskID, rootID)
	}
	if states[4].ReviewedTaskID != "" {
		t.Fatalf("states[4].ReviewedTaskID = %q, want empty", states[4].ReviewedTaskID)
	}
	if states[5].TaskID != otherID {
		t.Fatalf("states[5].TaskID = %q, want %q", states[5].TaskID, otherID)
	}
}

func TestRunExecuteStageTransitionsNoReviewingWhenParentReviewDisabled(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	parentID := "parent-1"
	childID := "a-child"
	otherID := "b-other"

	parent := makeItem(parentID, plan.StatusTodo)
	parent.ChildIDs = []string{childID}
	parent.AcceptanceCriteria = []string{"Parent criteria."}

	child := makeItem(childID, plan.StatusTodo)
	childParent := parentID
	child.ParentID = &childParent

	other := makeItem(otherID, plan.StatusTodo)

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
			otherID:  other,
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	var states []ExecutionStageState
	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		ParentReviewEnabled: false,
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":false,"resumeTaskIds":["a-child"],"feedbackForResume":"Fix child output before continuing."}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
		OnStateChange: func(state ExecutionStageState) {
			states = append(states, state)
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonCompleted {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonCompleted)
	}

	if len(states) != 2 {
		t.Fatalf("state transition count = %d, want 2 (%#v)", len(states), states)
	}
	for i, state := range states {
		if state.Stage != ExecutionStageExecuting {
			t.Fatalf("states[%d].Stage = %q, want %q", i, state.Stage, ExecutionStageExecuting)
		}
		if state.ReviewedTaskID != "" {
			t.Fatalf("states[%d].ReviewedTaskID = %q, want empty", i, state.ReviewedTaskID)
		}
	}
}
