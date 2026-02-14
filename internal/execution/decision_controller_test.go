package execution

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunExecuteStopAfterEachTaskAddsDecisionGate(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": makeItem("a", plan.StatusTodo),
			"b": makeItem("b", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:          planPath,
		Runtime:           agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
		StopAfterEachTask: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonDecisionRequired {
		t.Fatalf("expected decision required, got %s", result.Reason)
	}
	if result.TaskID != "a" {
		t.Fatalf("expected task a, got %s", result.TaskID)
	}
	if result.Run == nil || !result.Run.DecisionRequired || result.Run.DecisionState != DecisionStatePending || result.Run.DecisionRequestedAt == nil {
		t.Fatalf("expected decision gate on run: %#v", result.Run)
	}
	if result.Run.DecisionResolvedAt != nil {
		t.Fatalf("expected unresolved decision, got %v", result.Run.DecisionResolvedAt)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusDone {
		t.Fatalf("expected a done, got %s", updated.Items["a"].Status)
	}
	if updated.Items["b"].Status != plan.StatusTodo {
		t.Fatalf("expected b todo, got %s", updated.Items["b"].Status)
	}

	latest, err := GetLatestRun(tempDir, "a")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil || !latest.DecisionRequired || latest.DecisionState != DecisionStatePending || latest.DecisionRequestedAt == nil {
		t.Fatalf("expected persisted decision gate, got %#v", latest)
	}
}

func TestResolveDecisionApproveContinue(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": makeItem("a", plan.StatusTodo),
			"b": makeItem("b", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	runtime := agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second}
	first, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:          planPath,
		Runtime:           runtime,
		StopAfterEachTask: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if first.Reason != ExecuteReasonDecisionRequired {
		t.Fatalf("expected decision required, got %s", first.Reason)
	}

	controller := ExecutionController{PlanPath: planPath, Runtime: runtime, StopAfterEachTask: true}
	decision, err := controller.ResolveDecision(context.Background(), DecisionRequest{
		TaskID: first.TaskID,
		RunID:  first.Run.ID,
		Action: DecisionStateApprovedContinue,
	})
	if err != nil {
		t.Fatalf("ResolveDecision: %v", err)
	}
	if !decision.Continue {
		t.Fatalf("expected continue true")
	}
	if decision.Run.DecisionState != DecisionStateApprovedContinue || decision.Run.DecisionResolvedAt == nil {
		t.Fatalf("expected approved decision, got %#v", decision.Run)
	}

	second, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:          planPath,
		Runtime:           runtime,
		StopAfterEachTask: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if second.Reason != ExecuteReasonDecisionRequired || second.TaskID != "b" {
		t.Fatalf("expected decision required for b, got reason=%s task=%s", second.Reason, second.TaskID)
	}
}

func TestResolveDecisionApproveContinueRunsDeferredParentReview(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	parentID := "parent-1"
	childID := "child-1"

	parent := makeItem(parentID, plan.StatusTodo)
	parent.ChildIDs = []string{childID}
	parent.AcceptanceCriteria = []string{"Parent criteria."}

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

	runtime := agent.Runtime{
		Provider: "codex",
		Command:  `printf '{"passed":true}'`,
		UseShell: true,
		Timeout:  2 * time.Second,
	}
	first, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		Runtime:             runtime,
		StopAfterEachTask:   true,
		ParentReviewEnabled: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if first.Reason != ExecuteReasonDecisionRequired {
		t.Fatalf("expected decision required, got %s", first.Reason)
	}

	controller := ExecutionController{
		PlanPath:            planPath,
		Runtime:             runtime,
		StopAfterEachTask:   true,
		ParentReviewEnabled: true,
	}
	decision, err := controller.ResolveDecision(context.Background(), DecisionRequest{
		TaskID: first.TaskID,
		RunID:  first.Run.ID,
		Action: DecisionStateApprovedContinue,
	})
	if err != nil {
		t.Fatalf("ResolveDecision: %v", err)
	}
	if !decision.Continue {
		t.Fatalf("expected continue true for passing deferred parent review")
	}
	if decision.Next == nil {
		t.Fatalf("expected deferred parent review execute result")
	}
	if decision.Next.Reason != ExecuteReasonCompleted {
		t.Fatalf("decision.Next.Reason = %q, want %q", decision.Next.Reason, ExecuteReasonCompleted)
	}
	if decision.Next.Run == nil || decision.Next.Run.Type != RunTypeReview {
		t.Fatalf("expected decision.Next.Run to be a parent review run, got %#v", decision.Next.Run)
	}
	if decision.Next.Run.TaskID != parentID {
		t.Fatalf("decision.Next.Run.TaskID = %q, want %q", decision.Next.Run.TaskID, parentID)
	}
}

func TestResolveDecisionApproveQuit(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": makeItem("a", plan.StatusTodo),
			"b": makeItem("b", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	runtime := agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second}
	first, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:          planPath,
		Runtime:           runtime,
		StopAfterEachTask: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if first.Reason != ExecuteReasonDecisionRequired {
		t.Fatalf("expected decision required, got %s", first.Reason)
	}

	controller := ExecutionController{PlanPath: planPath, Runtime: runtime, StopAfterEachTask: true}
	decision, err := controller.ResolveDecision(context.Background(), DecisionRequest{
		TaskID: first.TaskID,
		RunID:  first.Run.ID,
		Action: DecisionStateApprovedQuit,
	})
	if err != nil {
		t.Fatalf("ResolveDecision approve quit: %v", err)
	}
	if decision.Continue {
		t.Fatalf("expected continue false")
	}
	if decision.Next != nil {
		t.Fatalf("expected no follow-on execution result")
	}
	if decision.Run.DecisionState != DecisionStateApprovedQuit || decision.Run.DecisionResolvedAt == nil {
		t.Fatalf("expected approved quit decision, got %#v", decision.Run)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusDone {
		t.Fatalf("expected a done, got %s", updated.Items["a"].Status)
	}
	if updated.Items["b"].Status != plan.StatusTodo {
		t.Fatalf("expected b todo, got %s", updated.Items["b"].Status)
	}

	latest, err := GetLatestRun(tempDir, "a")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil || latest.DecisionState != DecisionStateApprovedQuit || latest.DecisionResolvedAt == nil {
		t.Fatalf("expected persisted approved quit decision, got %#v", latest)
	}
}

func TestResolveDecisionRejectUpdatesPlan(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": makeItem("a", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	runtime := agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second}
	first, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:          planPath,
		Runtime:           runtime,
		StopAfterEachTask: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	controller := ExecutionController{PlanPath: planPath, Runtime: runtime, StopAfterEachTask: true}
	if _, err := controller.ResolveDecision(context.Background(), DecisionRequest{
		TaskID: first.TaskID,
		RunID:  first.Run.ID,
		Action: DecisionStateRejected,
	}); err != nil {
		t.Fatalf("ResolveDecision reject: %v", err)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusFailed {
		t.Fatalf("expected failed after reject, got %s", updated.Items["a"].Status)
	}
}

func TestResolveDecisionRequestChanges(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": makeItem("a", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	runtime := agent.Runtime{Provider: "codex", Command: "true", Timeout: 2 * time.Second}
	first, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:          planPath,
		Runtime:           runtime,
		StopAfterEachTask: true,
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if first.Run == nil || first.Run.ProviderSessionRef == "" {
		t.Fatalf("expected provider session ref on run")
	}

	controller := ExecutionController{PlanPath: planPath, Runtime: runtime, StopAfterEachTask: true}
	decision, err := controller.ResolveDecision(context.Background(), DecisionRequest{
		TaskID:   first.TaskID,
		RunID:    first.Run.ID,
		Action:   DecisionStateChangesRequested,
		Feedback: "please adjust",
	})
	if err != nil {
		t.Fatalf("ResolveDecision changes: %v", err)
	}
	if decision.Run.DecisionState != DecisionStateChangesRequested || decision.Run.DecisionFeedback == "" {
		t.Fatalf("expected changes requested decision, got %#v", decision.Run)
	}
	if decision.Next == nil {
		t.Fatalf("expected follow-on execution result")
	}
	if decision.Next.Reason != ExecuteReasonDecisionRequired {
		t.Fatalf("expected decision required after resume, got %s", decision.Next.Reason)
	}
	if decision.Next.Run == nil || !decision.Next.Run.DecisionRequired || decision.Next.Run.DecisionState != DecisionStatePending {
		t.Fatalf("expected pending decision on resumed run, got %#v", decision.Next.Run)
	}
	if decision.Next.Run.ID == first.Run.ID {
		t.Fatalf("expected new run id for resumed run")
	}
}
