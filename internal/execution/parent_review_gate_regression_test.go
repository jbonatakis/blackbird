package execution

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestParentReviewGateRegressionFinalChildDoneTriggersSingleReview(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, plan.DefaultPlanFilename)

	parentID := "parent"
	childAID := "child-a"
	childBID := "child-b"
	base := time.Date(2026, 2, 9, 15, 0, 0, 0, time.UTC)

	graph := parentReviewRegressionGraph(
		parentID,
		childAID,
		childBID,
		plan.StatusDone,
		base.Add(10*time.Minute),
		plan.StatusInProgress,
		base.Add(20*time.Minute),
	)
	parentReviewRegressionSaveGraph(t, planPath, graph)

	executor := newParentReviewRegressionExecutor(t, tempDir, base.Add(1*time.Hour))

	beforeFinalChildDone := runParentReviewGateFromDisk(t, planPath, childAID, executor.Execute)
	if beforeFinalChildDone.State != ParentReviewGateStateNoOp {
		t.Fatalf("before final child done state = %q, want %q", beforeFinalChildDone.State, ParentReviewGateStateNoOp)
	}
	if len(beforeFinalChildDone.Candidates) != 0 {
		t.Fatalf("expected no candidates before final child done, got %d", len(beforeFinalChildDone.Candidates))
	}
	if executor.CallCount != 0 {
		t.Fatalf("executor calls before final child done = %d, want 0", executor.CallCount)
	}

	graph = parentReviewRegressionLoadGraph(t, planPath)
	childB := graph.Items[childBID]
	childB.Status = plan.StatusDone
	childB.UpdatedAt = base.Add(30 * time.Minute)
	graph.Items[childBID] = childB
	parentReviewRegressionSaveGraph(t, planPath, graph)

	result := runParentReviewGateFromDisk(t, planPath, childBID, executor.Execute)
	if result.State != ParentReviewGateStatePass {
		t.Fatalf("result.State = %q, want %q", result.State, ParentReviewGateStatePass)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]
	if candidate.ParentTaskID != parentID {
		t.Fatalf("candidate.ParentTaskID = %q, want %q", candidate.ParentTaskID, parentID)
	}
	if candidate.State != ParentReviewGateStatePass {
		t.Fatalf("candidate.State = %q, want %q", candidate.State, ParentReviewGateStatePass)
	}
	if !candidate.RanReview {
		t.Fatalf("candidate.RanReview = false, want true")
	}
	if candidate.CompletionSignature == "" {
		t.Fatalf("candidate.CompletionSignature should be non-empty")
	}
	if executor.CallCount != 1 {
		t.Fatalf("executor call count = %d, want 1", executor.CallCount)
	}
	if len(executor.Candidates) != 1 {
		t.Fatalf("executor candidate count = %d, want 1", len(executor.Candidates))
	}
	if executor.Candidates[0].CompletionSignature != candidate.CompletionSignature {
		t.Fatalf("executor signature = %q, want %q", executor.Candidates[0].CompletionSignature, candidate.CompletionSignature)
	}

	reviewRuns, err := ListReviewRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns: %v", err)
	}
	if len(reviewRuns) != 1 {
		t.Fatalf("review run count = %d, want 1", len(reviewRuns))
	}
	if reviewRuns[0].ParentReviewCompletionSignature != candidate.CompletionSignature {
		t.Fatalf("stored signature = %q, want %q", reviewRuns[0].ParentReviewCompletionSignature, candidate.CompletionSignature)
	}
}

func TestParentReviewGateRegressionIdempotentAcrossRepeatedReloadLoops(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, plan.DefaultPlanFilename)

	parentID := "parent"
	childAID := "child-a"
	childBID := "child-b"
	base := time.Date(2026, 2, 9, 16, 0, 0, 0, time.UTC)

	graph := parentReviewRegressionGraph(
		parentID,
		childAID,
		childBID,
		plan.StatusDone,
		base.Add(5*time.Minute),
		plan.StatusDone,
		base.Add(10*time.Minute),
	)
	parentReviewRegressionSaveGraph(t, planPath, graph)

	executor := newParentReviewRegressionExecutor(t, tempDir, base.Add(2*time.Hour))
	signatures := make([]string, 0, 4)

	for i := 0; i < 4; i++ {
		result := runParentReviewGateFromDisk(t, planPath, childBID, executor.Execute)
		if len(result.Candidates) != 1 {
			t.Fatalf("loop %d candidate count = %d, want 1", i, len(result.Candidates))
		}

		candidate := result.Candidates[0]
		signatures = append(signatures, candidate.CompletionSignature)
		if candidate.CompletionSignature == "" {
			t.Fatalf("loop %d completion signature should be non-empty", i)
		}

		if i == 0 {
			if result.State != ParentReviewGateStatePass {
				t.Fatalf("loop %d result.State = %q, want %q", i, result.State, ParentReviewGateStatePass)
			}
			if !candidate.RanReview || candidate.State != ParentReviewGateStatePass {
				t.Fatalf("loop %d first candidate = %#v, want ran pass review", i, candidate)
			}
			continue
		}

		if result.State != ParentReviewGateStateNoOp {
			t.Fatalf("loop %d result.State = %q, want %q", i, result.State, ParentReviewGateStateNoOp)
		}
		if candidate.RanReview || candidate.State != ParentReviewGateStateNoOp {
			t.Fatalf("loop %d idempotent candidate = %#v, want no-op skip", i, candidate)
		}
	}

	if executor.CallCount != 1 {
		t.Fatalf("executor call count = %d, want 1", executor.CallCount)
	}
	for i := 1; i < len(signatures); i++ {
		if signatures[i] != signatures[0] {
			t.Fatalf("signature mismatch at loop %d: got %q want %q", i, signatures[i], signatures[0])
		}
	}

	reviewRuns, err := ListReviewRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns: %v", err)
	}
	if len(reviewRuns) != 1 {
		t.Fatalf("review run count = %d, want 1", len(reviewRuns))
	}
}

func TestParentReviewGateRegressionRetriggersAfterChildLeavesAndReturnsDone(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, plan.DefaultPlanFilename)

	parentID := "parent"
	childAID := "child-a"
	childBID := "child-b"
	base := time.Date(2026, 2, 9, 17, 0, 0, 0, time.UTC)

	graph := parentReviewRegressionGraph(
		parentID,
		childAID,
		childBID,
		plan.StatusDone,
		base.Add(5*time.Minute),
		plan.StatusDone,
		base.Add(10*time.Minute),
	)
	parentReviewRegressionSaveGraph(t, planPath, graph)

	executor := newParentReviewRegressionExecutor(t, tempDir, base.Add(3*time.Hour))

	initial := runParentReviewGateFromDisk(t, planPath, childBID, executor.Execute)
	if initial.State != ParentReviewGateStatePass || len(initial.Candidates) != 1 || !initial.Candidates[0].RanReview {
		t.Fatalf("initial gate result = %#v, want first review pass", initial)
	}

	unchanged := runParentReviewGateFromDisk(t, planPath, childBID, executor.Execute)
	if unchanged.State != ParentReviewGateStateNoOp || len(unchanged.Candidates) != 1 || unchanged.Candidates[0].RanReview {
		t.Fatalf("unchanged gate result = %#v, want idempotent no-op", unchanged)
	}

	graph = parentReviewRegressionLoadGraph(t, planPath)
	childB := graph.Items[childBID]
	childB.Status = plan.StatusInProgress
	childB.UpdatedAt = base.Add(20 * time.Minute)
	graph.Items[childBID] = childB
	parentReviewRegressionSaveGraph(t, planPath, graph)

	leftDone := runParentReviewGateFromDisk(t, planPath, childBID, executor.Execute)
	if leftDone.State != ParentReviewGateStateNoOp {
		t.Fatalf("left-done state = %q, want %q", leftDone.State, ParentReviewGateStateNoOp)
	}
	if len(leftDone.Candidates) != 0 {
		t.Fatalf("left-done candidate count = %d, want 0", len(leftDone.Candidates))
	}
	if executor.CallCount != 1 {
		t.Fatalf("executor call count after leaving done = %d, want 1", executor.CallCount)
	}

	graph = parentReviewRegressionLoadGraph(t, planPath)
	childB = graph.Items[childBID]
	childB.Status = plan.StatusDone
	childB.UpdatedAt = base.Add(30 * time.Minute)
	graph.Items[childBID] = childB
	parentReviewRegressionSaveGraph(t, planPath, graph)

	afterRework := runParentReviewGateFromDisk(t, planPath, childBID, executor.Execute)
	if afterRework.State != ParentReviewGateStatePass || len(afterRework.Candidates) != 1 || !afterRework.Candidates[0].RanReview {
		t.Fatalf("after rework gate result = %#v, want second review pass", afterRework)
	}
	if executor.CallCount != 2 {
		t.Fatalf("executor call count after rework = %d, want 2", executor.CallCount)
	}

	firstSignature := initial.Candidates[0].CompletionSignature
	secondSignature := afterRework.Candidates[0].CompletionSignature
	if firstSignature == secondSignature {
		t.Fatalf("expected signature change after rework cycle, got unchanged %q", firstSignature)
	}

	reviewRuns, err := ListReviewRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns: %v", err)
	}
	if len(reviewRuns) != 2 {
		t.Fatalf("review run count = %d, want 2", len(reviewRuns))
	}
	if reviewRuns[0].ParentReviewCompletionSignature == reviewRuns[1].ParentReviewCompletionSignature {
		t.Fatalf("persisted signatures should differ after rework cycle")
	}
}

func runParentReviewGateFromDisk(t *testing.T, planPath, changedChildID string, execute ParentReviewGateExecutor) ParentReviewGateResult {
	t.Helper()

	graph := parentReviewRegressionLoadGraph(t, planPath)
	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       planPath,
		Graph:          graph,
		ChangedChildID: changedChildID,
	}, execute)
	if err != nil {
		t.Fatalf("RunParentReviewGate: %v", err)
	}
	return result
}

func parentReviewRegressionLoadGraph(t *testing.T, planPath string) plan.WorkGraph {
	t.Helper()

	graph, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	return graph
}

func parentReviewRegressionSaveGraph(t *testing.T, planPath string, graph plan.WorkGraph) {
	t.Helper()

	if errs := plan.Validate(graph); len(errs) != 0 {
		t.Fatalf("invalid graph fixture: %v", errs)
	}
	if err := plan.SaveAtomic(planPath, graph); err != nil {
		t.Fatalf("plan.SaveAtomic: %v", err)
	}
}

func parentReviewRegressionGraph(
	parentID string,
	childAID string,
	childBID string,
	childAStatus plan.Status,
	childAUpdatedAt time.Time,
	childBStatus plan.Status,
	childBUpdatedAt time.Time,
) plan.WorkGraph {
	createdAt := childAUpdatedAt
	if childBUpdatedAt.Before(createdAt) {
		createdAt = childBUpdatedAt
	}
	createdAt = createdAt.Add(-1 * time.Hour)

	parentUpdatedAt := childAUpdatedAt
	if childBUpdatedAt.After(parentUpdatedAt) {
		parentUpdatedAt = childBUpdatedAt
	}

	parentStatus := plan.StatusInProgress
	if childAStatus == plan.StatusDone && childBStatus == plan.StatusDone {
		parentStatus = plan.StatusDone
	}

	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: {
				ID:                 parentID,
				Title:              "Task " + parentID,
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "review",
				ParentID:           nil,
				ChildIDs:           []string{childAID, childBID},
				Deps:               []string{},
				Status:             parentStatus,
				CreatedAt:          createdAt,
				UpdatedAt:          parentUpdatedAt,
			},
			childAID: {
				ID:                 childAID,
				Title:              "Task " + childAID,
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "implement",
				ParentID:           &parentID,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             childAStatus,
				CreatedAt:          createdAt,
				UpdatedAt:          childAUpdatedAt,
			},
			childBID: {
				ID:                 childBID,
				Title:              "Task " + childBID,
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "implement",
				ParentID:           &parentID,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             childBStatus,
				CreatedAt:          createdAt,
				UpdatedAt:          childBUpdatedAt,
			},
		},
	}
}

type parentReviewRegressionExecutor struct {
	baseDir    string
	startedAt  time.Time
	CallCount  int
	Candidates []ParentReviewGateCandidate
}

func newParentReviewRegressionExecutor(t *testing.T, baseDir string, startedAt time.Time) *parentReviewRegressionExecutor {
	t.Helper()
	return &parentReviewRegressionExecutor{
		baseDir:   baseDir,
		startedAt: startedAt,
	}
}

func (e *parentReviewRegressionExecutor) Execute(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error) {
	e.CallCount++
	e.Candidates = append(e.Candidates, candidate)

	startedAt := e.startedAt.Add(time.Duration(e.CallCount-1) * time.Minute)
	run := parentReviewRegressionReviewRun(
		candidate.ParentTaskID,
		fmt.Sprintf("review-%03d", e.CallCount),
		candidate.CompletionSignature,
		startedAt,
	)
	if err := SaveRun(e.baseDir, run); err != nil {
		return ParentReviewGateExecutorResult{}, err
	}

	return ParentReviewGateExecutorResult{State: ParentReviewGateStatePass}, nil
}

func parentReviewRegressionReviewRun(taskID, runID, signature string, startedAt time.Time) RunRecord {
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
