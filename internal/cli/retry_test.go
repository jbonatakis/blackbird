package cli

import (
	"os"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunRetryResetsFailedTask(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 22, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusFailed,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	failedRun := execution.RunRecord{
		ID:        "run-failed",
		TaskID:    "task",
		StartedAt: now,
		Status:    execution.RunStatusFailed,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task:          execution.TaskContext{ID: "task", Title: "Task"},
		},
	}
	if err := execution.SaveRun(tempDir, failedRun); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	if _, err := captureStdout(func() error { return runRetry("task") }); err != nil {
		t.Fatalf("runRetry: %v", err)
	}

	updated, err := plan.Load(planPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["task"].Status != plan.StatusTodo {
		t.Fatalf("expected task todo, got %s", updated.Items["task"].Status)
	}
}

func TestRunRetryRequiresFailedRun(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 22, 15, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusFailed,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if err := runRetry("task"); err == nil {
		t.Fatalf("expected error for missing failed runs")
	}
}
