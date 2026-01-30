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
	if loaded.ValidationErr != "" {
		t.Fatalf("expected empty validation error, got %q", loaded.ValidationErr)
	}
	if !loaded.PlanExists {
		t.Fatalf("expected PlanExists to be true when plan file is present")
	}
	if len(loaded.Plan.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(loaded.Plan.Items))
	}
	if _, ok := loaded.Plan.Items["task-1"]; !ok {
		t.Fatalf("expected task-1 to be loaded")
	}
}

func TestLoadPlanDataMissingPlan(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	model := Model{}
	msg := model.LoadPlanData()()
	loaded, ok := msg.(PlanDataLoaded)
	if !ok {
		t.Fatalf("expected PlanDataLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadPlanData error: %v", loaded.Err)
	}
	if loaded.ValidationErr != "" {
		t.Fatalf("expected empty validation error, got %q", loaded.ValidationErr)
	}
	if loaded.PlanExists {
		t.Fatalf("expected PlanExists to be false when plan file is missing")
	}
	if len(loaded.Plan.Items) != 0 {
		t.Fatalf("expected empty plan when missing, got %d items", len(loaded.Plan.Items))
	}
	if loaded.Plan.SchemaVersion != plan.SchemaVersion {
		t.Fatalf("expected schema version %d, got %d", plan.SchemaVersion, loaded.Plan.SchemaVersion)
	}
}

func TestLoadPlanDataValidationError(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	invalidPlan := `{"schemaVersion":1,"items":{"task-1":{"id":"task-1","title":"","description":"","prompt":"","status":"todo","createdAt":"2026-01-29T12:00:00Z","updatedAt":"2026-01-29T12:00:00Z"}}}`
	if err := os.WriteFile(plan.DefaultPlanFilename, []byte(invalidPlan), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	model := Model{}
	msg := model.LoadPlanData()()
	loaded, ok := msg.(PlanDataLoaded)
	if !ok {
		t.Fatalf("expected PlanDataLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("expected validation error to be stored, got load error: %v", loaded.Err)
	}
	if loaded.ValidationErr == "" {
		t.Fatalf("expected validation error summary, got empty string")
	}
	if !loaded.PlanExists {
		t.Fatalf("expected PlanExists to be true when plan file is present")
	}
	if len(loaded.Plan.Items) != 0 {
		t.Fatalf("expected empty plan when invalid, got %d items", len(loaded.Plan.Items))
	}
}
