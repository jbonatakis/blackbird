package tui

import (
	"os"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestLoadPlanData(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:          "task-1",
				Title:       "Plan item",
				Description: "Plan loader test.",
				Prompt:      "Prompt",
				Status:      plan.StatusTodo,
				ParentID:    nil,
				ChildIDs:    []string{},
				Deps:        []string{},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	if err := plan.SaveAtomic(plan.DefaultPlanFilename, g); err != nil {
		t.Fatalf("SaveAtomic: %v", err)
	}

	model := Model{}
	msg := model.LoadPlanData()()
	loaded, ok := msg.(PlanDataLoaded)
	if !ok {
		t.Fatalf("expected PlanDataLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadPlanData error: %v", loaded.Err)
	}
	if len(loaded.Plan.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(loaded.Plan.Items))
	}
	if _, ok := loaded.Plan.Items["task-1"]; !ok {
		t.Fatalf("expected task-1 to be loaded")
	}
}
