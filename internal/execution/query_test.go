package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListRunsSortsByStartedAt(t *testing.T) {
	baseDir := t.TempDir()
	first := RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 10, 0, 0, 0, time.UTC),
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	second := RunRecord{
		ID:        "run-2",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 11, 0, 0, 0, time.UTC),
		Status:    RunStatusFailed,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	if err := SaveRun(baseDir, second); err != nil {
		t.Fatalf("SaveRun second: %v", err)
	}
	if err := SaveRun(baseDir, first); err != nil {
		t.Fatalf("SaveRun first: %v", err)
	}

	records, err := ListRuns(baseDir, "task-1")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "run-1" || records[1].ID != "run-2" {
		t.Fatalf("unexpected order: %#v", records)
	}
}

func TestListRunsEmptyWhenMissing(t *testing.T) {
	baseDir := t.TempDir()
	records, err := ListRuns(baseDir, "task-missing")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected empty records, got %v", records)
	}
}

func TestLoadRun(t *testing.T) {
	baseDir := t.TempDir()
	record := RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 12, 0, 0, 0, time.UTC),
		Status:    RunStatusRunning,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	if err := SaveRun(baseDir, record); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	loaded, err := LoadRun(baseDir, "task-1", "run-1")
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	if loaded.ID != record.ID || loaded.TaskID != record.TaskID {
		t.Fatalf("loaded mismatch: %#v", loaded)
	}
}

func TestLoadRunMissingFile(t *testing.T) {
	baseDir := t.TempDir()
	_, err := LoadRun(baseDir, "task-1", "missing")
	if err == nil {
		t.Fatalf("expected error for missing run")
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, runsDirName)); statErr == nil {
		// ensure test doesn't accidentally create directories
		t.Fatalf("unexpected run directory created")
	}
}

func TestGetLatestRun(t *testing.T) {
	baseDir := t.TempDir()
	if latest, err := GetLatestRun(baseDir, "task-1"); err != nil || latest != nil {
		t.Fatalf("expected nil latest run, got %#v (err=%v)", latest, err)
	}

	first := RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 9, 0, 0, 0, time.UTC),
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	second := RunRecord{
		ID:        "run-2",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 10, 0, 0, 0, time.UTC),
		Status:    RunStatusFailed,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	if err := SaveRun(baseDir, first); err != nil {
		t.Fatalf("SaveRun first: %v", err)
	}
	if err := SaveRun(baseDir, second); err != nil {
		t.Fatalf("SaveRun second: %v", err)
	}

	latest, err := GetLatestRun(baseDir, "task-1")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil || latest.ID != "run-2" {
		t.Fatalf("unexpected latest: %#v", latest)
	}
}
