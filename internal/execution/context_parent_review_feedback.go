package execution

import (
	"fmt"
	"strings"

	"github.com/jbonatakis/blackbird/internal/agent"
)

// MergeParentReviewFeedbackContext returns a cloned context with parent-review feedback attached.
func MergeParentReviewFeedbackContext(
	base ContextPack,
	feedback ParentReviewFeedbackContext,
) (ContextPack, error) {
	feedback = normalizeParentReviewFeedbackContext(feedback)
	if feedback.ParentTaskID == "" && feedback.ReviewRunID == "" && feedback.Feedback == "" {
		return cloneContextPack(base), nil
	}
	if feedback.ParentTaskID == "" {
		return ContextPack{}, fmt.Errorf("parent task id required")
	}
	if feedback.ReviewRunID == "" {
		return ContextPack{}, fmt.Errorf("review run id required")
	}
	if feedback.Feedback == "" {
		return ContextPack{}, fmt.Errorf("feedback required")
	}

	merged := cloneContextPack(base)
	merged.ParentReviewFeedback = &feedback
	return merged, nil
}

// MergePendingParentReviewFeedbackContext maps pending parent-review feedback into ContextPack.
func MergePendingParentReviewFeedbackContext(
	base ContextPack,
	pending PendingParentReviewFeedback,
) (ContextPack, error) {
	return MergeParentReviewFeedbackContext(base, ParentReviewFeedbackContext{
		ParentTaskID: pending.ParentTaskID,
		ReviewRunID:  pending.ReviewRunID,
		Feedback:     pending.Feedback,
	})
}

func normalizeParentReviewFeedbackContext(
	feedback ParentReviewFeedbackContext,
) ParentReviewFeedbackContext {
	feedback.ParentTaskID = strings.TrimSpace(feedback.ParentTaskID)
	feedback.ReviewRunID = strings.TrimSpace(feedback.ReviewRunID)
	feedback.Feedback = strings.TrimSpace(feedback.Feedback)
	return feedback
}

func cloneContextPack(src ContextPack) ContextPack {
	cloned := src
	cloned.Task.AcceptanceCriteria = cloneStringSlice(src.Task.AcceptanceCriteria)
	cloned.Dependencies = cloneDependencyContextSlice(src.Dependencies)
	cloned.ParentReview = cloneParentReviewContext(src.ParentReview)
	cloned.Questions = cloneQuestionSlice(src.Questions)
	cloned.Answers = cloneAnswerSlice(src.Answers)
	if src.ParentReviewFeedback != nil {
		feedback := *src.ParentReviewFeedback
		cloned.ParentReviewFeedback = &feedback
	}
	return cloned
}

func cloneDependencyContextSlice(src []DependencyContext) []DependencyContext {
	if src == nil {
		return nil
	}
	cloned := make([]DependencyContext, len(src))
	for idx, dep := range src {
		cloned[idx] = dep
		cloned[idx].Artifacts = cloneStringSlice(dep.Artifacts)
	}
	return cloned
}

func cloneParentReviewContext(src *ParentReviewContext) *ParentReviewContext {
	if src == nil {
		return nil
	}

	cloned := *src
	cloned.AcceptanceCriteria = cloneStringSlice(src.AcceptanceCriteria)

	if src.Children != nil {
		cloned.Children = make([]ParentReviewChildContext, len(src.Children))
		for idx, child := range src.Children {
			cloned.Children[idx] = child
			cloned.Children[idx].ArtifactRefs = cloneStringSlice(child.ArtifactRefs)
		}
	}

	return &cloned
}

func cloneQuestionSlice(src []agent.Question) []agent.Question {
	if src == nil {
		return nil
	}
	cloned := make([]agent.Question, len(src))
	for idx, q := range src {
		cloned[idx] = q
		cloned[idx].Options = cloneStringSlice(q.Options)
	}
	return cloned
}

func cloneAnswerSlice(src []agent.Answer) []agent.Answer {
	if src == nil {
		return nil
	}
	cloned := make([]agent.Answer, len(src))
	copy(cloned, src)
	return cloned
}

func cloneStringSlice(src []string) []string {
	if src == nil {
		return nil
	}
	cloned := make([]string, len(src))
	copy(cloned, src)
	return cloned
}
