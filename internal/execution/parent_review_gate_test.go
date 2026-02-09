package execution

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunParentReviewGateNoOpWithoutCandidates(t *testing.T) {
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"child": parentReviewGateTestItem("child", plan.StatusDone, now, nil, nil),
		},
	}

	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       filepath.Join(t.TempDir(), "blackbird.plan.json"),
		Graph:          g,
		ChangedChildID: "child",
	}, nil)
	if err != nil {
		t.Fatalf("RunParentReviewGate: %v", err)
	}
	if result.State != ParentReviewGateStateNoOp {
		t.Fatalf("result.State = %q, want %q", result.State, ParentReviewGateStateNoOp)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected 0 candidate results, got %d", len(result.Candidates))
	}
}

func TestRunParentReviewGateDeterministicCallbackOrder(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g, changedChildID, midParentID, rootParentID := parentReviewGateTestGraph()
	var called []string

	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       planPath,
		Graph:          g,
		ChangedChildID: changedChildID,
	}, func(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error) {
		called = append(called, candidate.ParentTaskID)
		return ParentReviewGateExecutorResult{State: ParentReviewGateStatePass}, nil
	})
	if err != nil {
		t.Fatalf("RunParentReviewGate: %v", err)
	}
	if result.State != ParentReviewGateStatePass {
		t.Fatalf("result.State = %q, want %q", result.State, ParentReviewGateStatePass)
	}
	wantOrder := []string{midParentID, rootParentID}
	if !reflect.DeepEqual(called, wantOrder) {
		t.Fatalf("callback order = %v, want %v", called, wantOrder)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 candidate results, got %d", len(result.Candidates))
	}
	for i, candidate := range result.Candidates {
		if candidate.ParentTaskID != wantOrder[i] {
			t.Fatalf("candidate[%d].ParentTaskID = %q, want %q", i, candidate.ParentTaskID, wantOrder[i])
		}
		if candidate.State != ParentReviewGateStatePass {
			t.Fatalf("candidate[%d].State = %q, want %q", i, candidate.State, ParentReviewGateStatePass)
		}
		if !candidate.RanReview {
			t.Fatalf("candidate[%d].RanReview = false, want true", i)
		}
		if candidate.CompletionSignature == "" {
			t.Fatalf("candidate[%d] has empty completion signature", i)
		}
	}
}

func TestRunParentReviewGateSkipsIdempotentCandidates(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g, changedChildID, midParentID, rootParentID := parentReviewGateTestGraph()
	midSignature, err := parentReviewCompletionSignatureForTask(g, midParentID)
	if err != nil {
		t.Fatalf("parentReviewCompletionSignatureForTask(%s): %v", midParentID, err)
	}
	if err := SaveRun(tempDir, parentReviewGateTestReviewRun(midParentID, "review-mid-1", midSignature, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("SaveRun review-mid-1: %v", err)
	}

	var called []string
	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       planPath,
		Graph:          g,
		ChangedChildID: changedChildID,
	}, func(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error) {
		called = append(called, candidate.ParentTaskID)
		return ParentReviewGateExecutorResult{State: ParentReviewGateStatePass}, nil
	})
	if err != nil {
		t.Fatalf("RunParentReviewGate: %v", err)
	}

	if !reflect.DeepEqual(called, []string{rootParentID}) {
		t.Fatalf("callback parents = %v, want [%s]", called, rootParentID)
	}
	if result.State != ParentReviewGateStatePass {
		t.Fatalf("result.State = %q, want %q", result.State, ParentReviewGateStatePass)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 candidate results, got %d", len(result.Candidates))
	}
	first := result.Candidates[0]
	if first.ParentTaskID != midParentID || first.State != ParentReviewGateStateNoOp || first.RanReview {
		t.Fatalf("unexpected idempotent candidate result: %#v", first)
	}
	second := result.Candidates[1]
	if second.ParentTaskID != rootParentID || second.State != ParentReviewGateStatePass || !second.RanReview {
		t.Fatalf("unexpected executed candidate result: %#v", second)
	}
}

func TestRunParentReviewGateNoOpWhenAllCandidatesAreIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g, changedChildID, midParentID, rootParentID := parentReviewGateTestGraph()

	midSignature, err := parentReviewCompletionSignatureForTask(g, midParentID)
	if err != nil {
		t.Fatalf("parentReviewCompletionSignatureForTask(%s): %v", midParentID, err)
	}
	rootSignature, err := parentReviewCompletionSignatureForTask(g, rootParentID)
	if err != nil {
		t.Fatalf("parentReviewCompletionSignatureForTask(%s): %v", rootParentID, err)
	}

	fixtures := []RunRecord{
		parentReviewGateTestReviewRun(midParentID, "review-mid-1", midSignature, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)),
		parentReviewGateTestReviewRun(rootParentID, "review-root-1", rootSignature, time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC)),
	}
	for _, fixture := range fixtures {
		if err := SaveRun(tempDir, fixture); err != nil {
			t.Fatalf("SaveRun %s: %v", fixture.ID, err)
		}
	}

	calls := 0
	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       planPath,
		Graph:          g,
		ChangedChildID: changedChildID,
	}, func(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error) {
		calls++
		return ParentReviewGateExecutorResult{State: ParentReviewGateStatePass}, nil
	})
	if err != nil {
		t.Fatalf("RunParentReviewGate: %v", err)
	}
	if result.State != ParentReviewGateStateNoOp {
		t.Fatalf("result.State = %q, want %q", result.State, ParentReviewGateStateNoOp)
	}
	if calls != 0 {
		t.Fatalf("review callback calls = %d, want 0", calls)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 candidate results, got %d", len(result.Candidates))
	}
	for i, candidate := range result.Candidates {
		if candidate.State != ParentReviewGateStateNoOp {
			t.Fatalf("candidate[%d].State = %q, want %q", i, candidate.State, ParentReviewGateStateNoOp)
		}
		if candidate.RanReview {
			t.Fatalf("candidate[%d].RanReview = true, want false", i)
		}
	}
}

func TestRunParentReviewGatePauseRequiredAggregate(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g, changedChildID, midParentID, rootParentID := parentReviewGateTestGraph()
	var called []string

	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       planPath,
		Graph:          g,
		ChangedChildID: changedChildID,
	}, func(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error) {
		called = append(called, candidate.ParentTaskID)
		if candidate.ParentTaskID == midParentID {
			return ParentReviewGateExecutorResult{State: ParentReviewGateStatePass}, nil
		}
		return ParentReviewGateExecutorResult{State: ParentReviewGateStatePauseRequired}, nil
	})
	if err != nil {
		t.Fatalf("RunParentReviewGate: %v", err)
	}
	if result.State != ParentReviewGateStatePauseRequired {
		t.Fatalf("result.State = %q, want %q", result.State, ParentReviewGateStatePauseRequired)
	}
	wantOrder := []string{midParentID, rootParentID}
	if !reflect.DeepEqual(called, wantOrder) {
		t.Fatalf("callback order = %v, want %v", called, wantOrder)
	}
}

func parentReviewGateTestGraph() (plan.WorkGraph, string, string, string) {
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)
	rootParentID := "parent-root"
	midParentID := "parent-mid"
	childA := "child-a"
	childB := "child-b"

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			rootParentID: parentReviewGateTestItem(rootParentID, plan.StatusTodo, now, nil, []string{midParentID}),
			midParentID:  parentReviewGateTestItem(midParentID, plan.StatusDone, now, strPtr(rootParentID), []string{childA, childB}),
			childA:       parentReviewGateTestItem(childA, plan.StatusDone, now.Add(1*time.Minute), strPtr(midParentID), nil),
			childB:       parentReviewGateTestItem(childB, plan.StatusDone, now.Add(2*time.Minute), strPtr(midParentID), nil),
		},
	}

	return g, childB, midParentID, rootParentID
}

func parentReviewGateTestItem(id string, status plan.Status, updatedAt time.Time, parentID *string, childIDs []string) plan.WorkItem {
	return plan.WorkItem{
		ID:        id,
		Title:     "Task " + id,
		Prompt:    "do it",
		ParentID:  parentID,
		ChildIDs:  append([]string{}, childIDs...),
		Status:    status,
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
	}
}

func parentReviewGateTestReviewRun(taskID, runID, signature string, startedAt time.Time) RunRecord {
	completedAt := startedAt.Add(2 * time.Minute)
	return RunRecord{
		ID:          runID,
		TaskID:      taskID,
		Type:        RunTypeReview,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Status:      RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: taskID, Title: "Task " + taskID},
		},
		ParentReviewCompletionSignature: signature,
	}
}
