package execution

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ParentReviewResponse is the structured outcome returned by a parent review run.
type ParentReviewResponse struct {
	Passed            bool
	ResumeTaskIDs     []string
	FeedbackForResume string
	TaskResults       ParentReviewTaskResults
}

type parentReviewResponsePayload struct {
	Passed            *bool           `json:"passed"`
	ResumeTaskIDs     []string        `json:"resumeTaskIds"`
	FeedbackForResume string          `json:"feedbackForResume"`
	ReviewResults     json.RawMessage `json:"reviewResults"`
	TaskResults       json.RawMessage `json:"taskResults"`
}

// ParseParentReviewResponse extracts and validates parent review response JSON from agent output.
func ParseParentReviewResponse(output, parentTaskID string, parentChildIDs []string) (ParentReviewResponse, error) {
	parentTaskID = strings.TrimSpace(parentTaskID)
	if parentTaskID == "" {
		return ParentReviewResponse{}, fmt.Errorf("parent task id required")
	}

	payload, err := parseParentReviewResponsePayload(output, parentTaskID)
	if err != nil {
		return ParentReviewResponse{}, err
	}
	if payload.Passed == nil {
		return ParentReviewResponse{}, fmt.Errorf(
			`parse parent review response for %q: required field "passed" must be boolean`,
			parentTaskID,
		)
	}

	response := ParentReviewResponse{
		Passed:            *payload.Passed,
		ResumeTaskIDs:     append([]string{}, payload.ResumeTaskIDs...),
		FeedbackForResume: payload.FeedbackForResume,
		TaskResults:       parseParentReviewTaskResultsPayload(payload),
	}

	return ValidateParentReviewResponse(response, parentTaskID, parentChildIDs)
}

func parseParentReviewResponsePayload(output, parentTaskID string) (parentReviewResponsePayload, error) {
	candidates := findJSONObjectCandidates(output)
	if len(candidates) == 0 {
		return parentReviewResponsePayload{}, fmt.Errorf(
			"parse parent review response for %q: no valid JSON object found in agent output",
			parentTaskID,
		)
	}

	matches := make([]string, 0, 1)
	for _, candidate := range candidates {
		var header map[string]json.RawMessage
		if err := json.Unmarshal([]byte(candidate), &header); err != nil {
			continue
		}
		if _, ok := header["passed"]; !ok {
			continue
		}
		matches = append(matches, candidate)
	}

	if len(matches) == 0 {
		return parentReviewResponsePayload{}, fmt.Errorf(
			`parse parent review response for %q: missing required field "passed"`,
			parentTaskID,
		)
	}
	if len(matches) > 1 {
		return parentReviewResponsePayload{}, fmt.Errorf(
			`parse parent review response for %q: found %d JSON objects with field "passed"; expected exactly one`,
			parentTaskID,
			len(matches),
		)
	}

	var payload parentReviewResponsePayload
	if err := json.Unmarshal([]byte(matches[0]), &payload); err != nil {
		return parentReviewResponsePayload{}, fmt.Errorf("decode parent review response for %q: %w", parentTaskID, err)
	}
	return payload, nil
}

// ValidateParentReviewResponse validates parent review output against parent/child topology.
func ValidateParentReviewResponse(
	response ParentReviewResponse,
	parentTaskID string,
	parentChildIDs []string,
) (ParentReviewResponse, error) {
	parentTaskID = strings.TrimSpace(parentTaskID)
	if parentTaskID == "" {
		return ParentReviewResponse{}, fmt.Errorf("parent task id required")
	}
	if len(parentChildIDs) == 0 {
		return ParentReviewResponse{}, fmt.Errorf("validate parent review response for %q: parent child ids required", parentTaskID)
	}

	allowed := make(map[string]struct{}, len(parentChildIDs))
	for idx, childID := range parentChildIDs {
		childID = strings.TrimSpace(childID)
		if childID == "" {
			return ParentReviewResponse{}, fmt.Errorf(
				"validate parent review response for %q: parent child ids[%d] must be non-empty",
				parentTaskID,
				idx,
			)
		}
		allowed[childID] = struct{}{}
	}

	out := ParentReviewResponse{
		Passed:            response.Passed,
		FeedbackForResume: strings.TrimSpace(response.FeedbackForResume),
	}
	if len(response.ResumeTaskIDs) > 0 {
		out.ResumeTaskIDs = make([]string, 0, len(response.ResumeTaskIDs))
	}

	seen := make(map[string]struct{}, len(response.ResumeTaskIDs))
	for idx, resumeTaskID := range response.ResumeTaskIDs {
		resumeTaskID = strings.TrimSpace(resumeTaskID)
		if resumeTaskID == "" {
			return ParentReviewResponse{}, fmt.Errorf(
				"validate parent review response for %q: resumeTaskIds[%d] must be non-empty",
				parentTaskID,
				idx,
			)
		}
		if _, ok := seen[resumeTaskID]; ok {
			return ParentReviewResponse{}, fmt.Errorf(
				"validate parent review response for %q: duplicate resume task id %q",
				parentTaskID,
				resumeTaskID,
			)
		}
		if _, ok := allowed[resumeTaskID]; !ok {
			return ParentReviewResponse{}, fmt.Errorf(
				"validate parent review response for %q: resume task id %q is not a child of this parent",
				parentTaskID,
				resumeTaskID,
			)
		}
		seen[resumeTaskID] = struct{}{}
		out.ResumeTaskIDs = append(out.ResumeTaskIDs, resumeTaskID)
	}

	sort.Strings(out.ResumeTaskIDs)

	out.TaskResults = NormalizeParentReviewTaskResults(
		parentChildIDs,
		out.Passed,
		out.ResumeTaskIDs,
		out.FeedbackForResume,
		response.TaskResults,
	)

	if out.Passed {
		if len(out.ResumeTaskIDs) > 0 {
			return ParentReviewResponse{}, fmt.Errorf(
				"validate parent review response for %q: resumeTaskIds must be empty when passed=true",
				parentTaskID,
			)
		}
		if out.FeedbackForResume != "" {
			return ParentReviewResponse{}, fmt.Errorf(
				"validate parent review response for %q: feedbackForResume must be empty when passed=true",
				parentTaskID,
			)
		}
		return out, nil
	}

	if len(out.ResumeTaskIDs) == 0 {
		return ParentReviewResponse{}, fmt.Errorf(
			"validate parent review response for %q: resumeTaskIds required when passed=false",
			parentTaskID,
		)
	}
	if out.FeedbackForResume == "" {
		return ParentReviewResponse{}, fmt.Errorf(
			"validate parent review response for %q: feedbackForResume required when passed=false",
			parentTaskID,
		)
	}

	return out, nil
}

type parentReviewTaskResultPayload struct {
	TaskID   string `json:"taskId"`
	Status   string `json:"status"`
	Feedback string `json:"feedback"`
}

func parseParentReviewTaskResultsPayload(payload parentReviewResponsePayload) ParentReviewTaskResults {
	taskResults := make(ParentReviewTaskResults)
	appendPayloadResults(taskResults, decodeParentReviewTaskResultPayload(payload.TaskResults))
	appendPayloadResults(taskResults, decodeParentReviewTaskResultPayload(payload.ReviewResults))
	if len(taskResults) == 0 {
		return nil
	}
	return taskResults
}

func appendPayloadResults(taskResults ParentReviewTaskResults, payloadResults []parentReviewTaskResultPayload) {
	for _, payloadResult := range payloadResults {
		taskID := strings.TrimSpace(payloadResult.TaskID)
		if taskID == "" {
			continue
		}
		taskResults[taskID] = ParentReviewTaskResult{
			TaskID:   taskID,
			Status:   ParentReviewTaskStatus(strings.TrimSpace(payloadResult.Status)),
			Feedback: strings.TrimSpace(payloadResult.Feedback),
		}
	}
}

func decodeParentReviewTaskResultPayload(raw json.RawMessage) []parentReviewTaskResultPayload {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	var list []parentReviewTaskResultPayload
	if err := json.Unmarshal([]byte(trimmed), &list); err == nil {
		return list
	}

	keyed := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(trimmed), &keyed); err != nil {
		return nil
	}

	keys := make([]string, 0, len(keyed))
	for taskID := range keyed {
		keys = append(keys, taskID)
	}
	sort.Strings(keys)

	results := make([]parentReviewTaskResultPayload, 0, len(keys))
	for _, taskID := range keys {
		entryRaw := keyed[taskID]
		entry := parentReviewTaskResultPayload{TaskID: taskID}
		if err := json.Unmarshal(entryRaw, &entry); err == nil {
			if strings.TrimSpace(entry.TaskID) == "" {
				entry.TaskID = taskID
			}
			results = append(results, entry)
			continue
		}

		var status string
		if err := json.Unmarshal(entryRaw, &status); err == nil {
			entry.Status = status
			results = append(results, entry)
		}
	}

	return results
}
