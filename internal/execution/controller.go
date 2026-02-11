package execution

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// ExecutionController centralizes execution and decision checkpoint flows for CLI/TUI.
type ExecutionController struct {
	PlanPath            string
	Graph               *plan.WorkGraph
	Runtime             agent.Runtime
	StreamStdout        io.Writer
	StreamStderr        io.Writer
	StopAfterEachTask   bool
	ParentReviewEnabled bool
	OnStateChange       func(ExecutionStageState)
	OnParentReview      func(RunRecord)
	OnTaskStart         func(taskID string)
	OnTaskFinish        func(taskID string, record RunRecord, execErr error)
}

// DecisionRequest captures a user decision for a run checkpoint.
type DecisionRequest struct {
	TaskID   string
	RunID    string
	Action   DecisionState
	Feedback string
}

// DecisionResult captures the persisted decision plus any follow-on execution outcome.
type DecisionResult struct {
	Action   DecisionState
	Run      RunRecord
	Continue bool
	Next     *ExecuteResult
}

func (c ExecutionController) Execute(ctx context.Context) (ExecuteResult, error) {
	return RunExecute(ctx, ExecuteConfig{
		PlanPath:            c.PlanPath,
		Graph:               c.Graph,
		Runtime:             c.Runtime,
		StopAfterEachTask:   c.StopAfterEachTask,
		ParentReviewEnabled: c.ParentReviewEnabled,
		StreamStdout:        c.StreamStdout,
		StreamStderr:        c.StreamStderr,
		OnStateChange:       c.OnStateChange,
		OnParentReview:      c.OnParentReview,
		OnTaskStart:         c.OnTaskStart,
		OnTaskFinish:        c.OnTaskFinish,
	})
}

func (c ExecutionController) ResolveDecision(ctx context.Context, req DecisionRequest) (DecisionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.PlanPath == "" {
		return DecisionResult{}, fmt.Errorf("plan path required")
	}
	if req.TaskID == "" {
		return DecisionResult{}, fmt.Errorf("task id required")
	}
	if !isResolutionDecision(req.Action) {
		return DecisionResult{}, fmt.Errorf("invalid decision action %q", req.Action)
	}

	baseDir := filepath.Dir(c.PlanPath)

	record, err := loadDecisionRun(baseDir, req.TaskID, req.RunID)
	if err != nil {
		return DecisionResult{}, err
	}
	if !isDecisionPending(record) {
		return DecisionResult{}, fmt.Errorf("no pending decision for run %q", record.ID)
	}

	now := time.Now().UTC()
	record.DecisionRequired = true
	if record.DecisionRequestedAt == nil {
		record.DecisionRequestedAt = &now
	}
	record.DecisionState = req.Action
	record.DecisionResolvedAt = &now

	if req.Action == DecisionStateChangesRequested {
		feedback := strings.TrimSpace(req.Feedback)
		if feedback == "" {
			return DecisionResult{}, fmt.Errorf("decision feedback required")
		}
		record.DecisionFeedback = feedback
	} else {
		record.DecisionFeedback = ""
	}

	if err := SaveRun(baseDir, record); err != nil {
		return DecisionResult{}, err
	}

	result := DecisionResult{Action: req.Action, Run: record}

	switch req.Action {
	case DecisionStateApprovedContinue:
		result.Continue = true
		return result, nil
	case DecisionStateApprovedQuit:
		return result, nil
	case DecisionStateRejected:
		if err := UpdateTaskStatus(c.PlanPath, req.TaskID, plan.StatusFailed); err != nil {
			return result, err
		}
		return result, nil
	case DecisionStateChangesRequested:
		resumeRecord, execErr := RunResume(ctx, ResumeConfig{
			PlanPath:     c.PlanPath,
			Graph:        c.Graph,
			TaskID:       req.TaskID,
			Feedback:     record.DecisionFeedback,
			Runtime:      c.Runtime,
			StreamStdout: c.StreamStdout,
			StreamStderr: c.StreamStderr,
			OnTaskStart:  c.OnTaskStart,
			OnTaskFinish: c.OnTaskFinish,
		})
		if execErr != nil && resumeRecord.ID == "" {
			return result, execErr
		}
		next := ExecuteResult{TaskID: req.TaskID, Run: &resumeRecord}
		switch resumeRecord.Status {
		case RunStatusWaitingUser:
			next.Reason = ExecuteReasonWaitingUser
		case RunStatusSuccess, RunStatusFailed:
			markDecisionRequired(&resumeRecord)
			if err := SaveRun(baseDir, resumeRecord); err != nil {
				return result, err
			}
			next.Reason = ExecuteReasonDecisionRequired
		default:
			return result, fmt.Errorf("unexpected run status %q", resumeRecord.Status)
		}
		if execErr != nil {
			next.Err = execErr
		}
		result.Next = &next
		return result, nil
	default:
		return result, fmt.Errorf("unsupported decision action %q", req.Action)
	}
}

func isResolutionDecision(state DecisionState) bool {
	switch state {
	case DecisionStateApprovedContinue, DecisionStateApprovedQuit, DecisionStateChangesRequested, DecisionStateRejected:
		return true
	default:
		return false
	}
}

func loadDecisionRun(baseDir, taskID, runID string) (RunRecord, error) {
	if runID == "" {
		latest, err := GetLatestRun(baseDir, taskID)
		if err != nil {
			return RunRecord{}, err
		}
		if latest == nil {
			return RunRecord{}, fmt.Errorf("no runs found for %s", taskID)
		}
		return *latest, nil
	}
	return LoadRun(baseDir, taskID, runID)
}
