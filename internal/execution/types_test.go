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

	var decoded RunRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != record.ID || decoded.TaskID != record.TaskID || decoded.Status != record.Status {
		t.Fatalf("decoded record mismatch: %#v", decoded)
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
}
