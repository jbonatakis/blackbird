package execution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type ExecuteStopReason string

const (
	ExecuteReasonCompleted            ExecuteStopReason = "completed"
	ExecuteReasonWaitingUser          ExecuteStopReason = "waiting_user"
	ExecuteReasonDecisionRequired     ExecuteStopReason = "decision_required"
	ExecuteReasonParentReviewRequired ExecuteStopReason = "parent_review_required"
	ExecuteReasonCanceled             ExecuteStopReason = "canceled"
	ExecuteReasonError                ExecuteStopReason = "error"
)

type ExecuteResult struct {
	Reason ExecuteStopReason
	TaskID string
	Run    *RunRecord
	Err    error
}

type ExecuteConfig struct {
	PlanPath            string
	Graph               *plan.WorkGraph
	Runtime             agent.Runtime
	StopAfterEachTask   bool
	ParentReviewEnabled bool
	StreamStdout        io.Writer
	StreamStderr        io.Writer
	OnStateChange       func(ExecutionStageState)
	OnParentReview      func(RunRecord)
	OnTaskStart         func(taskID string)
	OnTaskFinish        func(taskID string, record RunRecord, execErr error)
}

type ResumeConfig struct {
	PlanPath     string
	Graph        *plan.WorkGraph
	TaskID       string
	Answers      []agent.Answer
	Feedback     string
	Context      *ContextPack
	Runtime      agent.Runtime
	StreamStdout io.Writer
	StreamStderr io.Writer
	OnTaskStart  func(taskID string)
	OnTaskFinish func(taskID string, record RunRecord, execErr error)
}

type WaitingRunNotFoundError struct {
	TaskID string
}

func (e WaitingRunNotFoundError) Error() string {
	return fmt.Sprintf("no waiting runs for %s", e.TaskID)
}

type NoQuestionsFoundError struct {
	TaskID string
}

func (e NoQuestionsFoundError) Error() string {
	return fmt.Sprintf("no questions found in waiting run for %s", e.TaskID)
}

func RunExecute(ctx context.Context, cfg ExecuteConfig) (ExecuteResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.PlanPath == "" {
		return ExecuteResult{Reason: ExecuteReasonError}, fmt.Errorf("plan path required")
	}

	baseDir := filepath.Dir(cfg.PlanPath)
	preloaded := cfg.Graph != nil
	var latestParentReviewRun *RunRecord

	for {
		if ctx.Err() != nil {
			return ExecuteResult{Reason: ExecuteReasonCanceled, Err: ctx.Err()}, nil
		}

		g, err := loadValidatedPlan(cfg.PlanPath, cfg.Graph, &preloaded)
		if err != nil {
			return ExecuteResult{Reason: ExecuteReasonError, Err: err}, err
		}

		ready := ReadyTasks(g)
		if len(ready) == 0 {
			if latestParentReviewRun != nil {
				run := *latestParentReviewRun
				return ExecuteResult{
					Reason: ExecuteReasonCompleted,
					Run:    &run,
				}, nil
			}
			return ExecuteResult{Reason: ExecuteReasonCompleted}, nil
		}
		if ctx.Err() != nil {
			return ExecuteResult{Reason: ExecuteReasonCanceled, Err: ctx.Err()}, nil
		}

		taskID := ready[0]
		emitExecutionStageState(cfg.OnStateChange, ExecutionStageState{
			Stage:  ExecutionStageExecuting,
			TaskID: taskID,
		})
		if cfg.OnTaskStart != nil {
			cfg.OnTaskStart(taskID)
		}

		ctxPack, err := BuildContext(g, taskID)
		if err != nil {
			return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID, Err: err}, err
		}
		if err := UpdateTaskStatus(cfg.PlanPath, taskID, plan.StatusInProgress); err != nil {
			return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID, Err: err}, err
		}

		record, execErr := LaunchAgentWithStream(ctx, cfg.Runtime, ctxPack, StreamConfig{
			Stdout: cfg.StreamStdout,
			Stderr: cfg.StreamStderr,
		})
		maybeAttachReviewSummary(baseDir, &record)
		decisionGate := requiresDecisionGate(cfg.StopAfterEachTask, record.Status)
		if decisionGate {
			markDecisionRequired(&record)
		}
		if err := SaveRun(baseDir, record); err != nil {
			return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID, Err: err}, err
		}

		switch record.Status {
		case RunStatusSuccess:
			if err := UpdateTaskStatus(cfg.PlanPath, taskID, plan.StatusDone); err != nil {
				return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID, Err: err}, err
			}
		case RunStatusWaitingUser:
			if err := UpdateTaskStatus(cfg.PlanPath, taskID, plan.StatusWaitingUser); err != nil {
				return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID, Err: err}, err
			}
		case RunStatusFailed:
			if err := UpdateTaskStatus(cfg.PlanPath, taskID, plan.StatusFailed); err != nil {
				return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID, Err: err}, err
			}
		default:
			return ExecuteResult{Reason: ExecuteReasonError, TaskID: taskID}, fmt.Errorf("unexpected run status %q", record.Status)
		}

		if cfg.OnTaskFinish != nil {
			cfg.OnTaskFinish(taskID, record, execErr)
		}

		var currentParentReviewRun *RunRecord
		var pauseReviewRun *RunRecord
		if cfg.ParentReviewEnabled && record.Status == RunStatusSuccess && ctx.Err() == nil {
			currentParentReviewRun, pauseReviewRun, err = runParentReviewGateForCompletedTask(ctx, cfg, taskID)
			if err != nil {
				return ExecuteResult{
					Reason: ExecuteReasonError,
					TaskID: taskID,
					Run:    currentParentReviewRun,
					Err:    err,
				}, err
			}
		}
		latestParentReviewRun = currentParentReviewRun

		if record.Status == RunStatusWaitingUser {
			return ExecuteResult{Reason: ExecuteReasonWaitingUser, TaskID: taskID, Run: &record}, nil
		}
		if pauseReviewRun != nil {
			pauseTaskID := strings.TrimSpace(pauseReviewRun.TaskID)
			if pauseTaskID == "" {
				pauseTaskID = taskID
			}
			return ExecuteResult{
				Reason: ExecuteReasonParentReviewRequired,
				TaskID: pauseTaskID,
				Run:    pauseReviewRun,
			}, nil
		}
		if decisionGate {
			return ExecuteResult{Reason: ExecuteReasonDecisionRequired, TaskID: taskID, Run: &record}, nil
		}
		if ctx.Err() != nil {
			return ExecuteResult{Reason: ExecuteReasonCanceled, TaskID: taskID, Run: &record, Err: ctx.Err()}, nil
		}
	}
}

func RunResume(ctx context.Context, cfg ResumeConfig) (RunRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.PlanPath == "" {
		return RunRecord{}, fmt.Errorf("plan path required")
	}
	if cfg.TaskID == "" {
		return RunRecord{}, fmt.Errorf("task id required")
	}
	if ctx.Err() != nil {
		return RunRecord{}, ctx.Err()
	}

	baseDir := filepath.Dir(cfg.PlanPath)
	preloaded := cfg.Graph != nil

	g, err := loadValidatedPlan(cfg.PlanPath, cfg.Graph, &preloaded)
	if err != nil {
		return RunRecord{}, err
	}
	if _, ok := g.Items[cfg.TaskID]; !ok {
		return RunRecord{}, fmt.Errorf("unknown id %q", cfg.TaskID)
	}

	resolvedFeedback, err := ResolveResumeFeedbackSource(baseDir, cfg.TaskID, cfg.Feedback, cfg.Answers)
	if err != nil {
		return RunRecord{}, err
	}

	if resolvedFeedback.UsesFeedback() {
		latest, err := GetLatestRun(baseDir, cfg.TaskID)
		if err != nil {
			return RunRecord{}, err
		}
		if latest == nil {
			return RunRecord{}, fmt.Errorf("no runs found for %s", cfg.TaskID)
		}
		if latest.Status == RunStatusWaitingUser {
			return RunRecord{}, fmt.Errorf("latest run for %s is waiting for user input; answer questions to resume", cfg.TaskID)
		}
		if normalizeProvider(latest.Provider) == "" {
			return RunRecord{}, fmt.Errorf("resume with feedback requires provider on previous run")
		}
		if cfg.Runtime.Provider != "" && normalizeProvider(cfg.Runtime.Provider) != normalizeProvider(latest.Provider) {
			return RunRecord{}, fmt.Errorf("resume with feedback provider mismatch: run uses %q, runtime uses %q", latest.Provider, cfg.Runtime.Provider)
		}
		if !supportsResumeProvider(latest.Provider) {
			return RunRecord{}, fmt.Errorf("resume with feedback unsupported for provider %q", latest.Provider)
		}
		if strings.TrimSpace(latest.ProviderSessionRef) == "" {
			return RunRecord{}, fmt.Errorf("resume with feedback requires provider session ref for run %q", latest.ID)
		}

		previous := *latest
		if resolvedFeedback.Source == ResumeFeedbackSourcePendingParentReview {
			mergedContext, err := MergeParentReviewFeedbackContext(previous.Context, ParentReviewFeedbackContext{
				ParentTaskID: resolvedFeedback.ParentTaskID,
				ReviewRunID:  resolvedFeedback.ReviewRunID,
				Feedback:     resolvedFeedback.Feedback,
			})
			if err != nil {
				return RunRecord{}, err
			}
			previous.Context = mergedContext
		}

		if cfg.OnTaskStart != nil {
			cfg.OnTaskStart(cfg.TaskID)
		}
		if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusInProgress); err != nil {
			return RunRecord{}, err
		}

		record, execErr := ResumeWithFeedback(ctx, cfg.Runtime, previous, resolvedFeedback.Feedback, StreamConfig{
			Stdout: cfg.StreamStdout,
			Stderr: cfg.StreamStderr,
		})
		if record.ID == "" {
			return RunRecord{}, execErr
		}
		maybeAttachReviewSummary(baseDir, &record)
		if err := SaveRun(baseDir, record); err != nil {
			return RunRecord{}, err
		}
		if resolvedFeedback.Source == ResumeFeedbackSourcePendingParentReview {
			if err := ClearPendingParentReviewFeedback(baseDir, cfg.TaskID); err != nil {
				return record, err
			}
		}

		switch record.Status {
		case RunStatusSuccess:
			if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusDone); err != nil {
				return record, err
			}
		case RunStatusWaitingUser:
			if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusWaitingUser); err != nil {
				return record, err
			}
		case RunStatusFailed:
			if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusFailed); err != nil {
				return record, err
			}
		default:
			return record, fmt.Errorf("unexpected run status %q", record.Status)
		}

		if cfg.OnTaskFinish != nil {
			cfg.OnTaskFinish(cfg.TaskID, record, execErr)
		}

		if ctx.Err() != nil {
			return record, ctx.Err()
		}

		return record, nil
	}

	waiting, err := latestWaitingRun(baseDir, cfg.TaskID)
	if err != nil {
		return RunRecord{}, err
	}

	var ctxPack ContextPack
	if cfg.Context != nil {
		if cfg.Context.Task.ID != "" && cfg.Context.Task.ID != cfg.TaskID {
			return RunRecord{}, fmt.Errorf("context task id %q does not match %q", cfg.Context.Task.ID, cfg.TaskID)
		}
		ctxPack = *cfg.Context
	} else {
		questions, err := ParseQuestions(waiting.Stdout)
		if err != nil {
			return RunRecord{}, err
		}
		if len(questions) == 0 {
			return RunRecord{}, NoQuestionsFoundError{TaskID: cfg.TaskID}
		}

		ctxPack, err = ResumeWithAnswer(*waiting, cfg.Answers)
		if err != nil {
			return RunRecord{}, err
		}
	}

	if cfg.OnTaskStart != nil {
		cfg.OnTaskStart(cfg.TaskID)
	}
	if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusInProgress); err != nil {
		return RunRecord{}, err
	}

	record, execErr := LaunchAgentWithStream(ctx, cfg.Runtime, ctxPack, StreamConfig{
		Stdout: cfg.StreamStdout,
		Stderr: cfg.StreamStderr,
	})
	maybeAttachReviewSummary(baseDir, &record)
	if err := SaveRun(baseDir, record); err != nil {
		return RunRecord{}, err
	}

	switch record.Status {
	case RunStatusSuccess:
		if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusDone); err != nil {
			return record, err
		}
	case RunStatusWaitingUser:
		if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusWaitingUser); err != nil {
			return record, err
		}
	case RunStatusFailed:
		if err := UpdateTaskStatus(cfg.PlanPath, cfg.TaskID, plan.StatusFailed); err != nil {
			return record, err
		}
	default:
		return record, fmt.Errorf("unexpected run status %q", record.Status)
	}

	if cfg.OnTaskFinish != nil {
		cfg.OnTaskFinish(cfg.TaskID, record, execErr)
	}

	if ctx.Err() != nil {
		return record, ctx.Err()
	}

	return record, nil
}

func latestWaitingRun(baseDir, taskID string) (*RunRecord, error) {
	runs, err := ListRuns(baseDir, taskID)
	if err != nil {
		return nil, err
	}
	for i := len(runs) - 1; i >= 0; i-- {
		if runs[i].Status == RunStatusWaitingUser {
			waiting := runs[i]
			return &waiting, nil
		}
	}
	return nil, WaitingRunNotFoundError{TaskID: taskID}
}

func runParentReviewGateForCompletedTask(
	ctx context.Context,
	cfg ExecuteConfig,
	changedChildID string,
) (*RunRecord, *RunRecord, error) {
	if !cfg.ParentReviewEnabled {
		return nil, nil, nil
	}
	if strings.TrimSpace(changedChildID) == "" {
		return nil, nil, nil
	}

	g, err := loadValidatedPlan(cfg.PlanPath, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	var latestReviewRun *RunRecord
	var pauseRun *RunRecord
	result, err := RunParentReviewGate(ParentReviewGateInput{
		PlanPath:       cfg.PlanPath,
		Graph:          g,
		ChangedChildID: changedChildID,
	}, func(candidate ParentReviewGateCandidate) (ParentReviewGateExecutorResult, error) {
		emitExecutionStageState(cfg.OnStateChange, ExecutionStageState{
			Stage:          ExecutionStageReviewing,
			TaskID:         changedChildID,
			ReviewedTaskID: candidate.ParentTaskID,
		})
		reviewRecord, reviewErr := RunParentReview(ctx, ParentReviewRunConfig{
			PlanPath:            cfg.PlanPath,
			Graph:               g,
			ParentTaskID:        candidate.ParentTaskID,
			CompletionSignature: candidate.CompletionSignature,
			Runtime:             cfg.Runtime,
			StreamStdout:        cfg.StreamStdout,
			StreamStderr:        cfg.StreamStderr,
		})
		if strings.TrimSpace(reviewRecord.ID) != "" {
			recordCopy := reviewRecord
			latestReviewRun = &recordCopy
			if reviewErr == nil {
				emitParentReviewRun(cfg.OnParentReview, recordCopy)
			}
		}
		emitExecutionStageState(cfg.OnStateChange, ExecutionStageState{
			Stage:  ExecutionStagePostReview,
			TaskID: changedChildID,
		})
		if reviewErr != nil {
			return ParentReviewGateExecutorResult{}, reviewErr
		}
		if requiresParentReviewPause(reviewRecord) {
			if pauseRun == nil {
				recordCopy := reviewRecord
				pauseRun = &recordCopy
			}
			return ParentReviewGateExecutorResult{State: ParentReviewGateStatePauseRequired}, nil
		}
		return ParentReviewGateExecutorResult{State: ParentReviewGateStatePass}, nil
	})
	if err != nil {
		return latestReviewRun, nil, err
	}

	if result.State == ParentReviewGateStatePauseRequired {
		if pauseRun == nil {
			return latestReviewRun, nil, fmt.Errorf("parent review gate pause requested without pause run context")
		}
		return latestReviewRun, pauseRun, nil
	}
	return latestReviewRun, nil, nil
}

func requiresParentReviewPause(record RunRecord) bool {
	if record.ParentReviewPassed == nil || *record.ParentReviewPassed {
		return false
	}
	return len(ParentReviewFailedTaskIDs(record)) > 0
}

func emitParentReviewRun(emit func(RunRecord), run RunRecord) {
	if emit == nil {
		return
	}
	if strings.TrimSpace(run.ID) == "" {
		return
	}
	emit(run)
}

func loadValidatedPlan(planPath string, preloaded *plan.WorkGraph, usePreloaded *bool) (plan.WorkGraph, error) {
	if preloaded != nil && usePreloaded != nil && *usePreloaded {
		*usePreloaded = false
		if errs := plan.Validate(*preloaded); len(errs) != 0 {
			return plan.WorkGraph{}, fmt.Errorf("plan is invalid (run `blackbird validate`): %s", planPath)
		}
		return *preloaded, nil
	}

	g, err := plan.Load(planPath)
	if err != nil {
		return plan.WorkGraph{}, err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return plan.WorkGraph{}, fmt.Errorf("plan is invalid (run `blackbird validate`): %s", planPath)
	}
	return g, nil
}

func IsWaitingRunNotFound(err error) bool {
	var target WaitingRunNotFoundError
	return errors.As(err, &target)
}

func IsNoQuestionsFound(err error) bool {
	var target NoQuestionsFoundError
	return errors.As(err, &target)
}

func ParseQuestionsFromLatestWaitingRun(planPath string, g plan.WorkGraph, taskID string) ([]agent.Question, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id required")
	}
	if _, ok := g.Items[taskID]; !ok {
		return nil, fmt.Errorf("unknown id %q", taskID)
	}

	baseDir := filepath.Dir(planPath)
	waiting, err := latestWaitingRun(baseDir, taskID)
	if err != nil {
		return nil, err
	}
	return ParseQuestions(waiting.Stdout)
}
