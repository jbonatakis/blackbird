package execution

import (
	"fmt"
	"path/filepath"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// ParentReviewGateState captures the aggregate or per-parent review-gate outcome.
type ParentReviewGateState string

const (
	ParentReviewGateStatePass          ParentReviewGateState = "pass"
	ParentReviewGateStatePauseRequired ParentReviewGateState = "pause_required"
	ParentReviewGateStateNoOp          ParentReviewGateState = "no_op"
)

// ParentReviewGateInput is the execution context for parent-review gate orchestration.
type ParentReviewGateInput struct {
	PlanPath       string
	Graph          plan.WorkGraph
	ChangedChildID string
}

// ParentReviewGateCandidate is a single parent task that is eligible for review execution.
type ParentReviewGateCandidate struct {
	ParentTaskID        string
	CompletionSignature string
}

// ParentReviewGateExecutorResult captures callback output for a single parent review run.
type ParentReviewGateExecutorResult struct {
	State ParentReviewGateState
}

// ParentReviewGateExecutor executes a single parent review run when idempotence allows it.
type ParentReviewGateExecutor func(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error)

// ParentReviewGateCandidateResult captures per-parent orchestration output.
type ParentReviewGateCandidateResult struct {
	ParentTaskID        string
	CompletionSignature string
	State               ParentReviewGateState
	RanReview           bool
}

// ParentReviewGateResult captures aggregate orchestration output and per-parent details.
type ParentReviewGateResult struct {
	State      ParentReviewGateState
	Candidates []ParentReviewGateCandidateResult
}

// RunParentReviewGate discovers parent-review candidates after a child completion, applies
// idempotence checks, and invokes execute for each eligible parent in deterministic order.
func RunParentReviewGate(input ParentReviewGateInput, execute ParentReviewGateExecutor) (ParentReviewGateResult, error) {
	if input.PlanPath == "" {
		return ParentReviewGateResult{}, fmt.Errorf("plan path required")
	}
	if input.ChangedChildID == "" {
		return ParentReviewGateResult{State: ParentReviewGateStateNoOp}, nil
	}

	candidateIDs := ParentReviewCandidateIDs(input.Graph, input.ChangedChildID)
	if len(candidateIDs) == 0 {
		return ParentReviewGateResult{State: ParentReviewGateStateNoOp}, nil
	}

	baseDir := filepath.Dir(input.PlanPath)
	results := make([]ParentReviewGateCandidateResult, 0, len(candidateIDs))
	ranAny := false
	pauseRequired := false

	for _, parentTaskID := range candidateIDs {
		signature, err := parentReviewCompletionSignatureForTask(input.Graph, parentTaskID)
		if err != nil {
			return ParentReviewGateResult{}, err
		}

		shouldRun, err := ShouldRunParentReviewForSignature(baseDir, parentTaskID, signature)
		if err != nil {
			return ParentReviewGateResult{}, fmt.Errorf("check parent review idempotence for %q: %w", parentTaskID, err)
		}

		candidateResult := ParentReviewGateCandidateResult{
			ParentTaskID:        parentTaskID,
			CompletionSignature: signature,
			State:               ParentReviewGateStateNoOp,
			RanReview:           false,
		}

		if !shouldRun {
			results = append(results, candidateResult)
			continue
		}
		if execute == nil {
			return ParentReviewGateResult{}, fmt.Errorf("parent review executor required")
		}

		execResult, err := execute(ParentReviewGateCandidate{
			ParentTaskID:        parentTaskID,
			CompletionSignature: signature,
		})
		if err != nil {
			return ParentReviewGateResult{}, fmt.Errorf("execute parent review for %q: %w", parentTaskID, err)
		}
		if err := validateParentReviewGateExecutorState(execResult.State); err != nil {
			return ParentReviewGateResult{}, fmt.Errorf("parent review executor result for %q: %w", parentTaskID, err)
		}

		candidateResult.State = execResult.State
		candidateResult.RanReview = true
		results = append(results, candidateResult)

		ranAny = true
		if execResult.State == ParentReviewGateStatePauseRequired {
			pauseRequired = true
		}
	}

	aggregate := ParentReviewGateStateNoOp
	if ranAny {
		if pauseRequired {
			aggregate = ParentReviewGateStatePauseRequired
		} else {
			aggregate = ParentReviewGateStatePass
		}
	}

	return ParentReviewGateResult{
		State:      aggregate,
		Candidates: results,
	}, nil
}

func parentReviewCompletionSignatureForTask(g plan.WorkGraph, parentTaskID string) (string, error) {
	parent, ok := g.Items[parentTaskID]
	if !ok {
		return "", fmt.Errorf("unknown id %q", parentTaskID)
	}
	if len(parent.ChildIDs) == 0 {
		return "", fmt.Errorf("parent task %q has no child ids", parentTaskID)
	}

	completions := make([]ChildCompletion, 0, len(parent.ChildIDs))
	for _, childID := range parent.ChildIDs {
		child, ok := g.Items[childID]
		if !ok {
			return "", fmt.Errorf("parent task %q references unknown child %q", parentTaskID, childID)
		}
		if child.Status != plan.StatusDone {
			return "", fmt.Errorf("parent task %q child %q is not done (%s)", parentTaskID, childID, child.Status)
		}

		completions = append(completions, ChildCompletion{
			ChildID:     childID,
			CompletedAt: child.UpdatedAt,
		})
	}

	signature, err := ParentReviewCompletionSignature(parentTaskID, completions)
	if err != nil {
		return "", fmt.Errorf("build parent review completion signature for %q: %w", parentTaskID, err)
	}
	return signature, nil
}

func validateParentReviewGateExecutorState(state ParentReviewGateState) error {
	switch state {
	case ParentReviewGateStatePass, ParentReviewGateStatePauseRequired:
		return nil
	default:
		return fmt.Errorf("unsupported state %q", state)
	}
}
