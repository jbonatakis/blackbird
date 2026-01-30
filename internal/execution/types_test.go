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
	exitCode := 0

	record := RunRecord{
		ID:          "run-123",
		TaskID:      "task-456",
		Provider:    "test-provider",
		StartedAt:   started,
		CompletedAt: &completed,
		Status:      RunStatusSuccess,
		ExitCode:    &exitCode,
		Stdout:      "hello",
		Stderr:      "",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task: TaskContext{
				ID:     "task-456",
				Title:  "Test Task",
				Prompt: "Do the thing",
			},
			ProjectSnapshot: "snapshot",
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
	if decoded.CompletedAt == nil || !decoded.CompletedAt.Equal(completed) {
		t.Fatalf("completedAt mismatch: %#v", decoded.CompletedAt)
	}
	if decoded.ExitCode == nil || *decoded.ExitCode != exitCode {
		t.Fatalf("exitCode mismatch: %#v", decoded.ExitCode)
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
}
