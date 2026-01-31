package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestUpdateTaskStatusTransitions(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 19, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	planFile := filepath.Join(tempDir, plan.DefaultPlanFilename)
	if err := plan.SaveAtomic(planFile, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if err := UpdateTaskStatus(planFile, "task", plan.StatusQueued); err != nil {
		t.Fatalf("queue: %v", err)
	}
	if err := UpdateTaskStatus(planFile, "task", plan.StatusInProgress); err != nil {
		t.Fatalf("in_progress: %v", err)
	}
	if err := UpdateTaskStatus(planFile, "task", plan.StatusWaitingUser); err != nil {
		t.Fatalf("waiting_user: %v", err)
	}
	if err := UpdateTaskStatus(planFile, "task", plan.StatusInProgress); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := UpdateTaskStatus(planFile, "task", plan.StatusDone); err != nil {
		t.Fatalf("done: %v", err)
	}
}

func TestUpdateTaskStatusRejectsInvalidTransition(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 28, 19, 30, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusDone,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	planFile := filepath.Join(tempDir, plan.DefaultPlanFilename)
	if err := plan.SaveAtomic(planFile, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if err := UpdateTaskStatus(planFile, "task", plan.StatusInProgress); err == nil {
		t.Fatalf("expected invalid transition error")
	}
}

func TestUpdateTaskStatusAfterNormalize(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	agentTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	normalizeTime := time.Now().UTC().Add(-time.Hour)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          agentTime,
				UpdatedAt:          agentTime,
			},
		},
	}
	normalized := plan.NormalizeWorkGraphTimestamps(g, normalizeTime)
	planFile := filepath.Join(tempDir, plan.DefaultPlanFilename)
	if err := plan.SaveAtomic(planFile, normalized); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if err := UpdateTaskStatus(planFile, "task", plan.StatusQueued); err != nil {
		t.Fatalf("queue after normalize: %v", err)
	}

	updated, err := plan.Load(planFile)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if errs := plan.Validate(updated); len(errs) != 0 {
		t.Fatalf("plan validation failed: %v", errs)
	}
	item := updated.Items["task"]
	if item.UpdatedAt.Before(item.CreatedAt) {
		t.Fatalf("updatedAt before createdAt: %s < %s", item.UpdatedAt, item.CreatedAt)
	}
}
