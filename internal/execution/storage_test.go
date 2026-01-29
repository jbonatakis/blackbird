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

	record := RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		Provider:  "test",
		StartedAt: started,
		Status:    RunStatusRunning,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task: TaskContext{ID: "task-1", Title: "Task"},
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
