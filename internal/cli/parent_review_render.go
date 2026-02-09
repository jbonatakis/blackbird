package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jbonatakis/blackbird/internal/execution"
)

const parentReviewFeedbackExcerptMaxLen = 200

func printParentReviewRequired(w io.Writer, fallbackTaskID string, run *execution.RunRecord) {
	for _, line := range formatParentReviewRequiredLines(fallbackTaskID, run) {
		fmt.Fprintln(w, line)
	}
}

func formatParentReviewRequiredLines(fallbackTaskID string, run *execution.RunRecord) []string {
	parentTaskID := parentReviewTaskID(fallbackTaskID, run)
	resumeTaskIDs := parentReviewResumeTaskIDs(run)
	feedback := parentReviewFeedbackExcerpt(run)

	lines := []string{
		fmt.Sprintf("running parent review for %s", parentTaskID),
		fmt.Sprintf("parent review failed for %s", parentTaskID),
		fmt.Sprintf("resume tasks: %s", strings.Join(resumeTaskIDs, ", ")),
		fmt.Sprintf("feedback: %s", feedback),
	}

	if len(resumeTaskIDs) == 1 && resumeTaskIDs[0] == "(none)" {
		lines = append(lines, "next step: no resume targets returned by parent review")
		return lines
	}

	for _, taskID := range resumeTaskIDs {
		lines = append(lines, fmt.Sprintf("next step: blackbird resume %s", taskID))
	}
	return lines
}

func parentReviewTaskID(fallbackTaskID string, run *execution.RunRecord) string {
	taskID := strings.TrimSpace(fallbackTaskID)
	if run != nil {
		if candidate := strings.TrimSpace(run.TaskID); candidate != "" {
			taskID = candidate
		}
	}
	if taskID == "" {
		return "unknown"
	}
	return taskID
}

func parentReviewResumeTaskIDs(run *execution.RunRecord) []string {
	if run == nil || len(run.ParentReviewResumeTaskIDs) == 0 {
		return []string{"(none)"}
	}

	seen := make(map[string]struct{}, len(run.ParentReviewResumeTaskIDs))
	ids := make([]string, 0, len(run.ParentReviewResumeTaskIDs))
	for _, id := range run.ParentReviewResumeTaskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return []string{"(none)"}
	}
	return ids
}

func parentReviewFeedbackExcerpt(run *execution.RunRecord) string {
	if run == nil {
		return "(none)"
	}

	feedback := strings.Join(strings.Fields(run.ParentReviewFeedback), " ")
	if feedback == "" {
		return "(none)"
	}
	if len(feedback) <= parentReviewFeedbackExcerptMaxLen {
		return feedback
	}

	trimTo := parentReviewFeedbackExcerptMaxLen - 3
	if trimTo < 0 {
		trimTo = 0
	}
	return feedback[:trimTo] + "..."
}
