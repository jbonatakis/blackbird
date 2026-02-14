package execution

import (
	"sort"
	"strings"
)

// NormalizeParentReviewTaskResults builds a deterministic task-indexed result map for all
// parent child tasks from normalized parent-review response fields and optional reviewer
// task-level overrides.
func NormalizeParentReviewTaskResults(
	parentChildIDs []string,
	passed bool,
	resumeTaskIDs []string,
	feedbackForResume string,
	taskResults ParentReviewTaskResults,
) ParentReviewTaskResults {
	feedbackForResume = strings.TrimSpace(feedbackForResume)

	allowed := make(map[string]struct{}, len(parentChildIDs))
	childIDs := make([]string, 0, len(parentChildIDs))
	for _, childID := range parentChildIDs {
		childID = strings.TrimSpace(childID)
		if childID == "" {
			continue
		}
		if _, ok := allowed[childID]; ok {
			continue
		}
		allowed[childID] = struct{}{}
		childIDs = append(childIDs, childID)
	}
	sort.Strings(childIDs)

	failedSet := make(map[string]struct{}, len(resumeTaskIDs))
	for _, taskID := range resumeTaskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		failedSet[taskID] = struct{}{}
	}

	out := make(ParentReviewTaskResults, len(childIDs))
	for _, childID := range childIDs {
		result := ParentReviewTaskResult{
			TaskID: childID,
			Status: ParentReviewTaskStatusPassed,
		}
		if !passed {
			if _, failed := failedSet[childID]; failed {
				result.Status = ParentReviewTaskStatusFailed
				result.Feedback = feedbackForResume
			}
		}
		out[childID] = result
	}

	for key, candidate := range taskResults {
		taskID := strings.TrimSpace(candidate.TaskID)
		if taskID == "" {
			taskID = strings.TrimSpace(key)
		}
		if taskID == "" {
			continue
		}
		if _, ok := allowed[taskID]; !ok {
			continue
		}

		base := out[taskID]
		status := normalizeParentReviewTaskStatus(candidate.Status, base.Status)
		taskFeedback := strings.TrimSpace(candidate.Feedback)
		if status == ParentReviewTaskStatusFailed {
			if taskFeedback == "" {
				taskFeedback = base.Feedback
			}
		} else {
			taskFeedback = ""
		}

		out[taskID] = ParentReviewTaskResult{
			TaskID:   taskID,
			Status:   status,
			Feedback: taskFeedback,
		}
	}

	if passed {
		for _, childID := range childIDs {
			out[childID] = ParentReviewTaskResult{
				TaskID: childID,
				Status: ParentReviewTaskStatusPassed,
			}
		}
		return out
	}

	for _, taskID := range resumeTaskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		existing, ok := out[taskID]
		if !ok {
			continue
		}
		existing.Status = ParentReviewTaskStatusFailed
		if strings.TrimSpace(existing.Feedback) == "" {
			existing.Feedback = feedbackForResume
		}
		out[taskID] = existing
	}

	for _, childID := range childIDs {
		existing := out[childID]
		switch existing.Status {
		case ParentReviewTaskStatusPassed:
			existing.Feedback = ""
		case ParentReviewTaskStatusFailed:
			if strings.TrimSpace(existing.Feedback) == "" {
				existing.Feedback = feedbackForResume
			}
		default:
			existing.Status = ParentReviewTaskStatusPassed
			existing.Feedback = ""
		}
		out[childID] = existing
	}

	return out
}

// ParentReviewTaskResultsForRecord returns the structured per-task review results for the run.
// It falls back to legacy top-level fields when the task-indexed payload is missing.
func ParentReviewTaskResultsForRecord(record RunRecord) ParentReviewTaskResults {
	if len(record.ParentReviewResults) > 0 {
		out := make(ParentReviewTaskResults, len(record.ParentReviewResults))
		for key, result := range record.ParentReviewResults {
			taskID := strings.TrimSpace(result.TaskID)
			if taskID == "" {
				taskID = strings.TrimSpace(key)
			}
			if taskID == "" {
				continue
			}
			status := normalizeParentReviewTaskStatus(result.Status, ParentReviewTaskStatusPassed)
			feedback := strings.TrimSpace(result.Feedback)
			if status != ParentReviewTaskStatusFailed {
				feedback = ""
			}
			out[taskID] = ParentReviewTaskResult{
				TaskID:   taskID,
				Status:   status,
				Feedback: feedback,
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	feedback := strings.TrimSpace(record.ParentReviewFeedback)
	out := make(ParentReviewTaskResults)
	for _, taskID := range record.ParentReviewResumeTaskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		out[taskID] = ParentReviewTaskResult{
			TaskID:   taskID,
			Status:   ParentReviewTaskStatusFailed,
			Feedback: feedback,
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParentReviewFailedTaskIDs returns deterministic failed task IDs from the review result model.
func ParentReviewFailedTaskIDs(record RunRecord) []string {
	results := ParentReviewTaskResultsForRecord(record)
	if len(results) == 0 {
		return nil
	}

	ids := make([]string, 0, len(results))
	for _, result := range results {
		taskID := strings.TrimSpace(result.TaskID)
		if taskID == "" {
			continue
		}
		if result.Status != ParentReviewTaskStatusFailed {
			continue
		}
		ids = append(ids, taskID)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)
	return ids
}

// ParentReviewFeedbackForTask resolves per-task feedback from the structured result model.
func ParentReviewFeedbackForTask(record RunRecord, taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ""
	}

	if len(record.ParentReviewResults) > 0 {
		results := ParentReviewTaskResultsForRecord(record)
		if result, ok := results[taskID]; ok && result.Status == ParentReviewTaskStatusFailed {
			return strings.TrimSpace(result.Feedback)
		}
		return ""
	}

	results := ParentReviewTaskResultsForRecord(record)
	if len(results) != 0 {
		if result, ok := results[taskID]; ok && result.Status == ParentReviewTaskStatusFailed {
			return strings.TrimSpace(result.Feedback)
		}
	}

	for _, candidate := range record.ParentReviewResumeTaskIDs {
		if strings.TrimSpace(candidate) == taskID {
			return strings.TrimSpace(record.ParentReviewFeedback)
		}
	}
	return ""
}

// ParentReviewPrimaryFeedback returns the first available failed-task feedback excerpt source.
func ParentReviewPrimaryFeedback(record RunRecord) string {
	feedback := strings.TrimSpace(record.ParentReviewFeedback)
	if feedback != "" {
		return feedback
	}

	for _, taskID := range ParentReviewFailedTaskIDs(record) {
		feedback = strings.TrimSpace(ParentReviewFeedbackForTask(record, taskID))
		if feedback != "" {
			return feedback
		}
	}

	return ""
}

func cloneParentReviewTaskResults(in ParentReviewTaskResults) ParentReviewTaskResults {
	if len(in) == 0 {
		return nil
	}
	out := make(ParentReviewTaskResults, len(in))
	for taskID, result := range in {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			taskID = strings.TrimSpace(result.TaskID)
		}
		if taskID == "" {
			continue
		}
		out[taskID] = ParentReviewTaskResult{
			TaskID:   taskID,
			Status:   normalizeParentReviewTaskStatus(result.Status, ParentReviewTaskStatusPassed),
			Feedback: strings.TrimSpace(result.Feedback),
		}
		if out[taskID].Status != ParentReviewTaskStatusFailed {
			clean := out[taskID]
			clean.Feedback = ""
			out[taskID] = clean
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeParentReviewTaskStatus(status ParentReviewTaskStatus, fallback ParentReviewTaskStatus) ParentReviewTaskStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "pass", "passed":
		return ParentReviewTaskStatusPassed
	case "fail", "failed":
		return ParentReviewTaskStatusFailed
	default:
		return fallback
	}
}
