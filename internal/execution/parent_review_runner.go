package execution

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// ParentReviewRunConfig configures a single parent-review execution run.
type ParentReviewRunConfig struct {
	PlanPath            string
	Graph               plan.WorkGraph
	ParentTaskID        string
	CompletionSignature string
	Runtime             agent.Runtime
	StreamStdout        io.Writer
	StreamStderr        io.Writer
	ContextOptions      ParentReviewContextOptions
}

// RunParentReview builds parent-review context, launches a review run, and persists it.
func RunParentReview(ctx context.Context, cfg ParentReviewRunConfig) (RunRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.PlanPath == "" {
		return RunRecord{}, fmt.Errorf("plan path required")
	}
	if cfg.ParentTaskID == "" {
		return RunRecord{}, fmt.Errorf("parent task id required")
	}

	baseDir := filepath.Dir(cfg.PlanPath)
	ctxPack, err := BuildParentReviewContext(cfg.Graph, baseDir, cfg.ParentTaskID, cfg.ContextOptions)
	if err != nil {
		return RunRecord{}, err
	}

	record, execErr := LaunchAgentWithStream(ctx, cfg.Runtime, ctxPack, StreamConfig{
		Stdout: cfg.StreamStdout,
		Stderr: cfg.StreamStderr,
	})
	if record.ID == "" {
		if execErr != nil {
			return RunRecord{}, execErr
		}
		return RunRecord{}, fmt.Errorf("parent review run missing id")
	}

	record.Type = RunTypeReview
	record.ParentReviewCompletionSignature = strings.TrimSpace(cfg.CompletionSignature)

	response, err := parseParentReviewOutcome(cfg.Graph, cfg.ParentTaskID, record)
	if err != nil {
		if saveErr := SaveRun(baseDir, record); saveErr != nil {
			return RunRecord{}, saveErr
		}
		return record, err
	}
	if response != nil {
		applyParentReviewOutcome(&record, *response)
	}

	if err := SaveRun(baseDir, record); err != nil {
		return RunRecord{}, err
	}
	if response != nil && !response.Passed {
		if err := persistPendingParentReviewFeedbackLinks(baseDir, cfg.ParentTaskID, record.ID, *response); err != nil {
			return record, err
		}
	}

	return record, execErr
}

func parseParentReviewOutcome(g plan.WorkGraph, parentTaskID string, record RunRecord) (*ParentReviewResponse, error) {
	if record.Status != RunStatusSuccess {
		return nil, nil
	}

	parent, ok := g.Items[parentTaskID]
	if !ok {
		return nil, fmt.Errorf("unknown id %q", parentTaskID)
	}

	response, err := ParseParentReviewResponse(record.Stdout, parentTaskID, parent.ChildIDs)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func applyParentReviewOutcome(record *RunRecord, response ParentReviewResponse) {
	if record == nil {
		return
	}

	passed := response.Passed
	record.ParentReviewPassed = &passed
	record.ParentReviewResults = cloneParentReviewTaskResults(response.TaskResults)
	record.ParentReviewResumeTaskIDs = ParentReviewFailedTaskIDs(*record)
	record.ParentReviewFeedback = ParentReviewPrimaryFeedback(*record)
}

func persistPendingParentReviewFeedbackLinks(
	baseDir string,
	parentTaskID string,
	reviewRunID string,
	response ParentReviewResponse,
) error {
	failedTaskIDs := ParentReviewFailedTaskIDs(RunRecord{
		ParentReviewResults:       response.TaskResults,
		ParentReviewResumeTaskIDs: response.ResumeTaskIDs,
		ParentReviewFeedback:      response.FeedbackForResume,
	})
	if len(failedTaskIDs) == 0 {
		failedTaskIDs = append([]string{}, response.ResumeTaskIDs...)
	}
	for _, childTaskID := range failedTaskIDs {
		feedback := strings.TrimSpace(response.FeedbackForResume)
		if result, ok := response.TaskResults[childTaskID]; ok {
			if taskFeedback := strings.TrimSpace(result.Feedback); taskFeedback != "" {
				feedback = taskFeedback
			}
		}
		if _, err := UpsertPendingParentReviewFeedback(
			baseDir,
			childTaskID,
			parentTaskID,
			reviewRunID,
			feedback,
		); err != nil {
			return fmt.Errorf("persist pending parent review feedback for child %q: %w", childTaskID, err)
		}
	}
	return nil
}
