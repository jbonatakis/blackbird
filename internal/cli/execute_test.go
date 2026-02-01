package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunExecuteSingleTask(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "cat")

	now := time.Date(2026, 1, 28, 20, 0, 0, 0, time.UTC)
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
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if _, err := captureStdout(func() error { return runExecute([]string{}) }); err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["task"].Status != plan.StatusDone {
		t.Fatalf("expected task done, got %s", updated.Items["task"].Status)
	}

	runsDir := filepath.Join(tempDir, ".blackbird", "runs", "task")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 run record, got %d", len(entries))
	}
}

func TestRunExecuteFailureContinues(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "exit 2")

	now := time.Date(2026, 1, 28, 20, 30, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Task A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "fail",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"b": {
				ID:                 "b",
				Title:              "Task B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "fail",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if _, err := captureStdout(func() error { return runExecute([]string{}) }); err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusFailed {
		t.Fatalf("expected task a failed, got %s", updated.Items["a"].Status)
	}
	if updated.Items["b"].Status != plan.StatusFailed {
		t.Fatalf("expected task b failed, got %s", updated.Items["b"].Status)
	}

	runsDir := filepath.Join(tempDir, ".blackbird", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 run task dirs, got %d", len(entries))
	}
}
