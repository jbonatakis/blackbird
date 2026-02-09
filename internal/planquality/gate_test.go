package planquality

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunQualityGateNoBlocking(t *testing.T) {
	initial := validExecutablePlan("leaf")
	callbackCalls := 0

	result, err := RunQualityGate(initial, 3, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		return input.Plan, nil
	})
	if err != nil {
		t.Fatalf("RunQualityGate() error = %v", err)
	}
	if callbackCalls != 0 {
		t.Fatalf("refine callback calls = %d, want 0", callbackCalls)
	}
	if result.AutoRefinePassesRun != 0 {
		t.Fatalf("AutoRefinePassesRun = %d, want 0", result.AutoRefinePassesRun)
	}
	if !reflect.DeepEqual(result.FinalPlan, initial) {
		t.Fatalf("FinalPlan changed unexpectedly")
	}
	if len(result.InitialFindings) != 0 {
		t.Fatalf("InitialFindings = %v, want empty", result.InitialFindings)
	}
	if len(result.FinalFindings) != 0 {
		t.Fatalf("FinalFindings = %v, want empty", result.FinalFindings)
	}
}

func TestRunQualityGateMaxPassesZeroSkipsRefine(t *testing.T) {
	initial := planWithBlockingDescription("leaf")
	callbackCalls := 0

	result, err := RunQualityGate(initial, 0, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		return input.Plan, nil
	})
	if err != nil {
		t.Fatalf("RunQualityGate() error = %v", err)
	}
	if callbackCalls != 0 {
		t.Fatalf("refine callback calls = %d, want 0", callbackCalls)
	}
	if result.AutoRefinePassesRun != 0 {
		t.Fatalf("AutoRefinePassesRun = %d, want 0", result.AutoRefinePassesRun)
	}
	if !HasBlocking(result.InitialFindings) {
		t.Fatalf("expected blocking findings in InitialFindings")
	}
	if !HasBlocking(result.FinalFindings) {
		t.Fatalf("expected blocking findings in FinalFindings")
	}
}

func TestRunQualityGateNegativeMaxPassesClampsToZero(t *testing.T) {
	initial := planWithBlockingDescription("leaf")
	callbackCalls := 0

	result, err := RunQualityGate(initial, -4, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		return input.Plan, nil
	})
	if err != nil {
		t.Fatalf("RunQualityGate() error = %v", err)
	}
	if callbackCalls != 0 {
		t.Fatalf("refine callback calls = %d, want 0", callbackCalls)
	}
	if result.AutoRefinePassesRun != 0 {
		t.Fatalf("AutoRefinePassesRun = %d, want 0", result.AutoRefinePassesRun)
	}
	if !HasBlocking(result.InitialFindings) {
		t.Fatalf("expected blocking findings in InitialFindings")
	}
	if !HasBlocking(result.FinalFindings) {
		t.Fatalf("expected blocking findings in FinalFindings")
	}
}

func TestRunQualityGateBlockingWithEnabledPassesRequiresRefineCallback(t *testing.T) {
	initial := planWithBlockingDescription("leaf")

	result, err := RunQualityGate(initial, 1, nil)
	if !errors.Is(err, ErrRefineCallbackRequired) {
		t.Fatalf("RunQualityGate() error = %v, want %v", err, ErrRefineCallbackRequired)
	}
	if result.AutoRefinePassesRun != 0 {
		t.Fatalf("AutoRefinePassesRun = %d, want 0", result.AutoRefinePassesRun)
	}
	if !HasBlocking(result.InitialFindings) {
		t.Fatalf("expected blocking findings in InitialFindings")
	}
	if !HasBlocking(result.FinalFindings) {
		t.Fatalf("expected blocking findings in FinalFindings")
	}
}

func TestRunQualityGateClearsBlockingAfterOnePass(t *testing.T) {
	initial := planWithBlockingDescription("leaf")
	refined := validExecutablePlan("leaf")
	callbackCalls := 0

	result, err := RunQualityGate(initial, 3, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		if input.Pass != 1 {
			t.Fatalf("Pass = %d, want 1", input.Pass)
		}
		if input.MaxPasses != 3 {
			t.Fatalf("MaxPasses = %d, want 3", input.MaxPasses)
		}
		if !HasBlocking(input.Findings) {
			t.Fatalf("expected blocking findings passed into callback")
		}
		if strings.TrimSpace(input.ChangeRequest) == "" {
			t.Fatalf("expected non-empty ChangeRequest")
		}
		return refined, nil
	})
	if err != nil {
		t.Fatalf("RunQualityGate() error = %v", err)
	}
	if callbackCalls != 1 {
		t.Fatalf("refine callback calls = %d, want 1", callbackCalls)
	}
	if result.AutoRefinePassesRun != 1 {
		t.Fatalf("AutoRefinePassesRun = %d, want 1", result.AutoRefinePassesRun)
	}
	if !HasBlocking(result.InitialFindings) {
		t.Fatalf("expected blocking findings in InitialFindings")
	}
	if HasBlocking(result.FinalFindings) {
		t.Fatalf("expected no blocking findings in FinalFindings, got %v", result.FinalFindings)
	}
	if !reflect.DeepEqual(result.FinalPlan, refined) {
		t.Fatalf("FinalPlan mismatch after refine")
	}
}

func TestRunQualityGateRemainsBlockingAfterMaxPasses(t *testing.T) {
	initial := planWithBlockingDescription("leaf")
	callbackCalls := 0

	result, err := RunQualityGate(initial, 2, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		return planWithBlockingDescription("leaf"), nil
	})
	if err != nil {
		t.Fatalf("RunQualityGate() error = %v", err)
	}
	if callbackCalls != 2 {
		t.Fatalf("refine callback calls = %d, want 2", callbackCalls)
	}
	if result.AutoRefinePassesRun != 2 {
		t.Fatalf("AutoRefinePassesRun = %d, want 2", result.AutoRefinePassesRun)
	}
	if !HasBlocking(result.FinalFindings) {
		t.Fatalf("expected blocking findings to remain in FinalFindings")
	}
}

func TestRunQualityGateRefineCallbackError(t *testing.T) {
	initial := planWithBlockingDescription("leaf")
	callbackCalls := 0
	wantErr := errors.New("refine failed")

	result, err := RunQualityGate(initial, 2, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		return plan.WorkGraph{}, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunQualityGate() error = %v, want %v", err, wantErr)
	}
	if callbackCalls != 1 {
		t.Fatalf("refine callback calls = %d, want 1", callbackCalls)
	}
	if result.AutoRefinePassesRun != 0 {
		t.Fatalf("AutoRefinePassesRun = %d, want 0", result.AutoRefinePassesRun)
	}
	if !HasBlocking(result.FinalFindings) {
		t.Fatalf("expected blocking findings to remain after callback error")
	}
}

func TestRunQualityGateRefinedPlanMustValidate(t *testing.T) {
	initial := planWithBlockingDescription("leaf")
	callbackCalls := 0

	_, err := RunQualityGate(initial, 2, func(input AutoRefineInput) (plan.WorkGraph, error) {
		callbackCalls++
		return plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"leaf": {
					ID: "leaf",
				},
			},
		}, nil
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "returned invalid plan") {
		t.Fatalf("error = %q, want invalid plan message", err.Error())
	}
	if callbackCalls != 1 {
		t.Fatalf("refine callback calls = %d, want 1", callbackCalls)
	}
}

func validExecutablePlan(id string) plan.WorkGraph {
	now := time.Date(2026, 2, 6, 12, 0, 0, 0, time.UTC)
	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			id: {
				ID:          id,
				Title:       "Task " + id,
				Description: "Implement deterministic bounded quality-gate orchestration for generated plans.",
				AcceptanceCriteria: []string{
					"`RunQualityGate` returns deterministic findings for identical input.",
					"`go test ./internal/planquality/...` passes.",
				},
				Prompt:    "Implement `RunQualityGate`, include loop validation, and verify behavior with package tests.",
				ChildIDs:  []string{},
				Deps:      []string{},
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

func planWithBlockingDescription(id string) plan.WorkGraph {
	g := validExecutablePlan(id)
	item := g.Items[id]
	item.Description = "TODO"
	g.Items[id] = item
	return g
}
