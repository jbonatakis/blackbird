package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunRunsDisplaysTable(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 15, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": newWorkItem("task-1", now),
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	record := execution.RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: now,
		Status:    execution.RunStatusSuccess,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task:          execution.TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	if err := execution.SaveRun(tempDir, record); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	output, err := captureStdout(func() error {
		return runRuns([]string{"task-1"})
	})
	if err != nil {
		t.Fatalf("runRuns: %v", err)
	}
	if !strings.Contains(output, "Run ID") || !strings.Contains(output, "run-1") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunRunsVerboseOutput(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 16, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": newWorkItem("task-1", now),
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	record := execution.RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: now,
		Status:    execution.RunStatusSuccess,
		Stdout:    "hello",
		Stderr:    "warning",
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task:          execution.TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	if err := execution.SaveRun(tempDir, record); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	output, err := captureStdout(func() error {
		return runRuns([]string{"--verbose", "task-1"})
	})
	if err != nil {
		t.Fatalf("runRuns: %v", err)
	}
	if !strings.Contains(output, "Stdout:") || !strings.Contains(output, "hello") {
		t.Fatalf("expected stdout in verbose output: %q", output)
	}
	if !strings.Contains(output, "Stderr:") || !strings.Contains(output, "warning") {
		t.Fatalf("expected stderr in verbose output: %q", output)
	}
}

func TestRunRunsNoRuns(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 17, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": newWorkItem("task-1", now),
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error {
		return runRuns([]string{"task-1"})
	})
	if err != nil {
		t.Fatalf("runRuns: %v", err)
	}
	if !strings.Contains(output, "no runs found") {
		t.Fatalf("expected no runs message, got %q", output)
	}
}
