package execution

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRunRecordJSONRoundTrip(t *testing.T) {
	started := time.Date(2026, 1, 28, 10, 30, 0, 0, time.UTC)
	completed := time.Date(2026, 1, 28, 10, 35, 0, 0, time.UTC)
	decisionRequested := time.Date(2026, 1, 28, 10, 36, 0, 0, time.UTC)
	decisionResolved := time.Date(2026, 1, 28, 10, 40, 0, 0, time.UTC)
	exitCode := 0

	record := RunRecord{
		ID:                 "run-123",
		TaskID:             "task-456",
		Type:               RunTypeExecute,
		Provider:           "test-provider",
		ProviderSessionRef: "session-abc",
		StartedAt:          started,
		CompletedAt:        &completed,
		Status:             RunStatusSuccess,
		ExitCode:           &exitCode,
		Stdout:             "hello",
		Stderr:             "",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task: TaskContext{
				ID:     "task-456",
				Title:  "Test Task",
				Prompt: "Do the thing",
			},
			ProjectSnapshot: "snapshot",
		},
		DecisionRequired:    true,
		DecisionState:       DecisionStateApprovedContinue,
		DecisionRequestedAt: &decisionRequested,
		DecisionResolvedAt:  &decisionResolved,
		DecisionFeedback:    "Looks good",
		ReviewSummary: &ReviewSummary{
			Files:    []string{"main.go"},
			DiffStat: "1 file changed, 2 insertions(+)",
			Snippets: []ReviewSnippet{{File: "main.go", Snippet: "fmt.Println(\"hi\")"}},
		},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(data), "\"startedAt\":\"2026-01-28T10:30:00Z\"") {
		t.Fatalf("expected startedAt in UTC, got %s", string(data))
	}
	if !strings.Contains(string(data), "\"completedAt\":\"2026-01-28T10:35:00Z\"") {
		t.Fatalf("expected completedAt in UTC, got %s", string(data))
	}
	if !strings.Contains(string(data), "\"run_type\":\"execute\"") {
		t.Fatalf("expected run_type in payload, got %s", string(data))
	}

	var decoded RunRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != record.ID || decoded.TaskID != record.TaskID || decoded.Status != record.Status {
		t.Fatalf("decoded record mismatch: %#v", decoded)
	}
	if decoded.Type != RunTypeExecute {
		t.Fatalf("type mismatch: %#v", decoded.Type)
	}
	if decoded.ProviderSessionRef != "session-abc" {
		t.Fatalf("providerSessionRef mismatch: %#v", decoded.ProviderSessionRef)
	}
	if decoded.CompletedAt == nil || !decoded.CompletedAt.Equal(completed) {
		t.Fatalf("completedAt mismatch: %#v", decoded.CompletedAt)
	}
	if decoded.ExitCode == nil || *decoded.ExitCode != exitCode {
		t.Fatalf("exitCode mismatch: %#v", decoded.ExitCode)
	}
	if !decoded.DecisionRequired {
		t.Fatalf("decisionRequired mismatch: %#v", decoded.DecisionRequired)
	}
	if decoded.DecisionState != DecisionStateApprovedContinue {
		t.Fatalf("decisionState mismatch: %#v", decoded.DecisionState)
	}
	if decoded.DecisionRequestedAt == nil || !decoded.DecisionRequestedAt.Equal(decisionRequested) {
		t.Fatalf("decisionRequestedAt mismatch: %#v", decoded.DecisionRequestedAt)
	}
	if decoded.DecisionResolvedAt == nil || !decoded.DecisionResolvedAt.Equal(decisionResolved) {
		t.Fatalf("decisionResolvedAt mismatch: %#v", decoded.DecisionResolvedAt)
	}
	if decoded.DecisionFeedback != "Looks good" {
		t.Fatalf("decisionFeedback mismatch: %#v", decoded.DecisionFeedback)
	}
	if decoded.ReviewSummary == nil || decoded.ReviewSummary.DiffStat != "1 file changed, 2 insertions(+)" {
		t.Fatalf("reviewSummary mismatch: %#v", decoded.ReviewSummary)
	}
	if len(decoded.ReviewSummary.Snippets) != 1 || decoded.ReviewSummary.Snippets[0].File != "main.go" {
		t.Fatalf("reviewSummary snippets mismatch: %#v", decoded.ReviewSummary.Snippets)
	}
}

func TestRunRecordJSONOmitEmptyFields(t *testing.T) {
	record := RunRecord{
		ID:        "run-omit",
		TaskID:    "task-omit",
		StartedAt: time.Date(2026, 1, 28, 12, 0, 0, 0, time.UTC),
		Status:    RunStatusRunning,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-omit", Title: "Task"},
		},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := string(data)

	if strings.Contains(payload, "completedAt") {
		t.Fatalf("expected completedAt to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "exitCode") {
		t.Fatalf("expected exitCode to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "decision_required") {
		t.Fatalf("expected decision_required to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "decision_state") {
		t.Fatalf("expected decision_state to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "decision_requested_at") {
		t.Fatalf("expected decision_requested_at to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "decision_resolved_at") {
		t.Fatalf("expected decision_resolved_at to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "decision_feedback") {
		t.Fatalf("expected decision_feedback to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "review_summary") {
		t.Fatalf("expected review_summary to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "provider_session_ref") {
		t.Fatalf("expected provider_session_ref to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "parent_review_passed") {
		t.Fatalf("expected parent_review_passed to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "parent_review_resume_task_ids") {
		t.Fatalf("expected parent_review_resume_task_ids to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "parent_review_feedback") {
		t.Fatalf("expected parent_review_feedback to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "parent_review_results") {
		t.Fatalf("expected parent_review_results to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "parent_review_completion_signature") {
		t.Fatalf("expected parent_review_completion_signature to be omitted, got %s", payload)
	}
	if strings.Contains(payload, "parentReviewFeedback") {
		t.Fatalf("expected parentReviewFeedback to be omitted, got %s", payload)
	}
	if !strings.Contains(payload, "\"run_type\":\"execute\"") {
		t.Fatalf("expected run_type default to execute, got %s", payload)
	}
}

func TestRunRecordJSONUnmarshalLegacyShapeDefaultsRunType(t *testing.T) {
	legacy := `{"id":"run-legacy","taskId":"task-legacy","startedAt":"2026-01-28T12:00:00Z","status":"success","context":{"schemaVersion":1,"task":{"id":"task-legacy","title":"Task"}}}`

	var decoded RunRecord
	if err := json.Unmarshal([]byte(legacy), &decoded); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}

	if decoded.Type != RunTypeExecute {
		t.Fatalf("legacy run type mismatch: got %q want %q", decoded.Type, RunTypeExecute)
	}
	if decoded.ParentReviewPassed != nil {
		t.Fatalf("expected parent_review_passed to remain nil, got %#v", decoded.ParentReviewPassed)
	}
	if len(decoded.ParentReviewResumeTaskIDs) != 0 {
		t.Fatalf("expected parent_review_resume_task_ids empty, got %#v", decoded.ParentReviewResumeTaskIDs)
	}
	if decoded.ParentReviewFeedback != "" {
		t.Fatalf("expected parent_review_feedback empty, got %#v", decoded.ParentReviewFeedback)
	}
	if len(decoded.ParentReviewResults) != 0 {
		t.Fatalf("expected parent_review_results empty, got %#v", decoded.ParentReviewResults)
	}
	if decoded.ParentReviewCompletionSignature != "" {
		t.Fatalf("expected parent_review_completion_signature empty, got %#v", decoded.ParentReviewCompletionSignature)
	}
}

func TestRunRecordJSONReviewShapeRoundTrip(t *testing.T) {
	started := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)
	passed := false
	record := RunRecord{
		ID:        "run-review-1",
		TaskID:    "parent-1",
		Type:      RunTypeReview,
		StartedAt: started,
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task: TaskContext{
				ID:    "parent-1",
				Title: "Parent Review",
			},
		},
		ParentReviewPassed:        &passed,
		ParentReviewResumeTaskIDs: []string{"child-a", "child-b"},
		ParentReviewFeedback:      "Child task output misses acceptance criteria coverage.",
		ParentReviewResults: ParentReviewTaskResults{
			"child-c": {
				TaskID: "child-c",
				Status: ParentReviewTaskStatusPassed,
			},
			"child-b": {
				TaskID:   "child-b",
				Status:   ParentReviewTaskStatusFailed,
				Feedback: "Fix child-b coverage.",
			},
			"child-a": {
				TaskID:   "child-a",
				Status:   ParentReviewTaskStatusFailed,
				Feedback: "Fix child-a coverage.",
			},
		},
		ParentReviewCompletionSignature: "children:child-a,child-b",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal review shape: %v", err)
	}
	payload := string(data)
	if !strings.Contains(payload, "\"run_type\":\"review\"") {
		t.Fatalf("expected run_type review, got %s", payload)
	}
	if !strings.Contains(payload, "\"parent_review_passed\":false") {
		t.Fatalf("expected explicit failed review result, got %s", payload)
	}
	if !strings.Contains(payload, "\"parent_review_resume_task_ids\":[\"child-a\",\"child-b\"]") {
		t.Fatalf("expected resume task ids, got %s", payload)
	}
	if !strings.Contains(payload, "\"parent_review_feedback\":\"Child task output misses acceptance criteria coverage.\"") {
		t.Fatalf("expected parent review feedback, got %s", payload)
	}
	if !strings.Contains(payload, "\"parent_review_results\"") {
		t.Fatalf("expected parent review task results, got %s", payload)
	}
	if !strings.Contains(payload, "\"parent_review_completion_signature\":\"children:child-a,child-b\"") {
		t.Fatalf("expected parent review completion signature, got %s", payload)
	}

	var decoded RunRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal review shape: %v", err)
	}

	if decoded.Type != RunTypeReview {
		t.Fatalf("run type mismatch: got %q want %q", decoded.Type, RunTypeReview)
	}
	if decoded.ParentReviewPassed == nil || *decoded.ParentReviewPassed {
		t.Fatalf("parent_review_passed mismatch: %#v", decoded.ParentReviewPassed)
	}
	if len(decoded.ParentReviewResumeTaskIDs) != 2 {
		t.Fatalf("parent_review_resume_task_ids mismatch: %#v", decoded.ParentReviewResumeTaskIDs)
	}
	if decoded.ParentReviewResumeTaskIDs[0] != "child-a" || decoded.ParentReviewResumeTaskIDs[1] != "child-b" {
		t.Fatalf("parent_review_resume_task_ids values mismatch: %#v", decoded.ParentReviewResumeTaskIDs)
	}
	if decoded.ParentReviewFeedback != "Child task output misses acceptance criteria coverage." {
		t.Fatalf("parent_review_feedback mismatch: %#v", decoded.ParentReviewFeedback)
	}
	if len(decoded.ParentReviewResults) != 3 {
		t.Fatalf("parent_review_results length mismatch: %#v", decoded.ParentReviewResults)
	}
	if got := decoded.ParentReviewResults["child-a"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "Fix child-a coverage." {
		t.Fatalf("parent_review_results[child-a] mismatch: %#v", got)
	}
	if got := decoded.ParentReviewResults["child-b"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "Fix child-b coverage." {
		t.Fatalf("parent_review_results[child-b] mismatch: %#v", got)
	}
	if got := decoded.ParentReviewResults["child-c"]; got.Status != ParentReviewTaskStatusPassed || got.Feedback != "" {
		t.Fatalf("parent_review_results[child-c] mismatch: %#v", got)
	}
	if decoded.ParentReviewCompletionSignature != "children:child-a,child-b" {
		t.Fatalf("parent_review_completion_signature mismatch: %#v", decoded.ParentReviewCompletionSignature)
	}
}

func TestRunRecordJSONRoundTripWithParentReviewFeedbackContext(t *testing.T) {
	started := time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC)
	record := RunRecord{
		ID:        "run-resume-feedback-1",
		TaskID:    "child-1",
		StartedAt: started,
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task: TaskContext{
				ID:    "child-1",
				Title: "Child Task",
			},
			ParentReviewFeedback: &ParentReviewFeedbackContext{
				ParentTaskID: "parent-1",
				ReviewRunID:  "review-run-3",
				Feedback:     "Fix failing checkout validation before retrying.",
			},
		},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := string(data)
	if !strings.Contains(payload, `"parentReviewFeedback":{"parentTaskId":"parent-1","reviewRunId":"review-run-3","feedback":"Fix failing checkout validation before retrying."}`) {
		t.Fatalf("expected parent review feedback section, got %s", payload)
	}

	var decoded RunRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected parent review feedback after round trip")
	}
	if decoded.Context.ParentReviewFeedback.ParentTaskID != "parent-1" {
		t.Fatalf("ParentTaskID mismatch: %#v", decoded.Context.ParentReviewFeedback.ParentTaskID)
	}
	if decoded.Context.ParentReviewFeedback.ReviewRunID != "review-run-3" {
		t.Fatalf("ReviewRunID mismatch: %#v", decoded.Context.ParentReviewFeedback.ReviewRunID)
	}
	if decoded.Context.ParentReviewFeedback.Feedback != "Fix failing checkout validation before retrying." {
		t.Fatalf("Feedback mismatch: %#v", decoded.Context.ParentReviewFeedback.Feedback)
	}
}
