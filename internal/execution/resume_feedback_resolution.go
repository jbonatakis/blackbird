package execution

import (
	"fmt"
	"strings"

	"github.com/jbonatakis/blackbird/internal/agent"
)

type ResumeFeedbackSource string

const (
	ResumeFeedbackSourceNone                ResumeFeedbackSource = "none"
	ResumeFeedbackSourceExplicit            ResumeFeedbackSource = "explicit"
	ResumeFeedbackSourcePendingParentReview ResumeFeedbackSource = "pending_parent_review"
)

// ResolvedResumeFeedback captures the feedback text and where it came from.
type ResolvedResumeFeedback struct {
	Source       ResumeFeedbackSource
	Feedback     string
	ParentTaskID string
	ReviewRunID  string
}

func (r ResolvedResumeFeedback) UsesFeedback() bool {
	return r.Source != ResumeFeedbackSourceNone
}

// ResolveResumeFeedbackSource applies deterministic precedence for resume feedback:
// explicit feedback > pending parent-review feedback > none.
func ResolveResumeFeedbackSource(
	baseDir string,
	taskID string,
	explicitFeedback string,
	answers []agent.Answer,
) (ResolvedResumeFeedback, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return ResolvedResumeFeedback{}, fmt.Errorf("baseDir required")
	}

	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ResolvedResumeFeedback{}, fmt.Errorf("task id required")
	}

	feedback := strings.TrimSpace(explicitFeedback)
	if feedback != "" {
		if len(answers) > 0 {
			return ResolvedResumeFeedback{}, fmt.Errorf(
				"resume answers cannot be combined with feedback-based resume; provide either answers or feedback",
			)
		}
		return ResolvedResumeFeedback{
			Source:   ResumeFeedbackSourceExplicit,
			Feedback: feedback,
		}, nil
	}

	pending, err := LoadPendingParentReviewFeedback(baseDir, taskID)
	if err != nil {
		return ResolvedResumeFeedback{}, fmt.Errorf("resolve resume feedback for %q: %w", taskID, err)
	}
	if pending == nil {
		return ResolvedResumeFeedback{Source: ResumeFeedbackSourceNone}, nil
	}

	parentTaskID := strings.TrimSpace(pending.ParentTaskID)
	reviewRunID := strings.TrimSpace(pending.ReviewRunID)
	pendingFeedback := strings.TrimSpace(pending.Feedback)
	if parentTaskID == "" {
		return ResolvedResumeFeedback{}, fmt.Errorf(
			"pending parent-review feedback for %q is invalid: parent task id required",
			taskID,
		)
	}
	if reviewRunID == "" {
		return ResolvedResumeFeedback{}, fmt.Errorf(
			"pending parent-review feedback for %q is invalid: review run id required",
			taskID,
		)
	}
	if pendingFeedback == "" {
		return ResolvedResumeFeedback{}, fmt.Errorf(
			"pending parent-review feedback for %q is invalid: feedback required",
			taskID,
		)
	}
	if len(answers) > 0 {
		return ResolvedResumeFeedback{}, fmt.Errorf(
			"resume answers cannot be combined with pending parent-review feedback from %q (review run %q); retry with answers omitted",
			parentTaskID,
			reviewRunID,
		)
	}

	return ResolvedResumeFeedback{
		Source:       ResumeFeedbackSourcePendingParentReview,
		Feedback:     pendingFeedback,
		ParentTaskID: parentTaskID,
		ReviewRunID:  reviewRunID,
	}, nil
}
