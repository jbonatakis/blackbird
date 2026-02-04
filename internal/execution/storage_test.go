package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveRunWritesRecord(t *testing.T) {
	baseDir := t.TempDir()
	started := time.Date(2026, 1, 28, 14, 0, 0, 0, time.UTC)
	decisionRequested := time.Date(2026, 1, 28, 14, 5, 0, 0, time.UTC)
	decisionResolved := time.Date(2026, 1, 28, 14, 10, 0, 0, time.UTC)

	record := RunRecord{
		ID:                 "run-1",
		TaskID:             "task-1",
		Provider:           "test",
		ProviderSessionRef: "session-1",
		StartedAt:          started,
		Status:             RunStatusRunning,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
		DecisionRequired:    true,
		DecisionState:       DecisionStateChangesRequested,
		DecisionRequestedAt: &decisionRequested,
		DecisionResolvedAt:  &decisionResolved,
		DecisionFeedback:    "Please adjust",
		ReviewSummary: &ReviewSummary{
			Files:    []string{"main.go", "README.md"},
			DiffStat: "2 files changed, 4 insertions(+)",
		},
	}

	if err := SaveRun(baseDir, record); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	path := filepath.Join(baseDir, runsDirName, "task-1", "run-1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read run record: %v", err)
	}

	var decoded RunRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != record.ID || decoded.TaskID != record.TaskID {
		t.Fatalf("decoded mismatch: %#v", decoded)
	}
	if decoded.ProviderSessionRef != "session-1" {
		t.Fatalf("providerSessionRef mismatch: %#v", decoded.ProviderSessionRef)
	}
	if !decoded.DecisionRequired || decoded.DecisionState != DecisionStateChangesRequested {
		t.Fatalf("decision gate mismatch: %#v", decoded)
	}
	if decoded.DecisionRequestedAt == nil || !decoded.DecisionRequestedAt.Equal(decisionRequested) {
		t.Fatalf("decisionRequestedAt mismatch: %#v", decoded.DecisionRequestedAt)
	}
	if decoded.DecisionResolvedAt == nil || !decoded.DecisionResolvedAt.Equal(decisionResolved) {
		t.Fatalf("decisionResolvedAt mismatch: %#v", decoded.DecisionResolvedAt)
	}
	if decoded.DecisionFeedback != "Please adjust" {
		t.Fatalf("decisionFeedback mismatch: %#v", decoded.DecisionFeedback)
	}
	if decoded.ReviewSummary == nil || len(decoded.ReviewSummary.Files) != 2 {
		t.Fatalf("reviewSummary mismatch: %#v", decoded.ReviewSummary)
	}
}

func TestSaveRunValidation(t *testing.T) {
	baseDir := t.TempDir()
	record := RunRecord{TaskID: "task"}
	if err := SaveRun(baseDir, record); err == nil {
		t.Fatalf("expected error for missing run id")
	}

	record = RunRecord{ID: "run"}
	if err := SaveRun(baseDir, record); err == nil {
		t.Fatalf("expected error for missing task id")
	}
}
