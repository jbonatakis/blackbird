package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRefinePlanInMemoryRunsInProcess(t *testing.T) {
	tmp := t.TempDir()
	responsePath := filepath.Join(tmp, "response.json")
	response := `{"schemaVersion":1,"type":"plan_refine","plan":{"schemaVersion":1,"items":{"task":{"id":"task","title":"Updated Task","description":"","acceptanceCriteria":[],"prompt":"do it","parentId":null,"childIds":[],"deps":[],"status":"todo","createdAt":"2026-01-31T09:00:00Z","updatedAt":"2026-01-31T09:00:00Z"}}}}`
	if err := os.WriteFile(responsePath, []byte(response), 0o600); err != nil {
		t.Fatalf("write response: %v", err)
	}

	t.Setenv("BLACKBIRD_AGENT_CMD", "cat "+responsePath)

	base := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"base": {
				ID:                 "base",
				Title:              "Base Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
			},
		},
	}

	cmd := RefinePlanInMemory(context.Background(), "Change the plan", base)
	msg := cmd()
	result, ok := msg.(PlanGenerateInMemoryResult)
	if !ok {
		t.Fatalf("expected PlanGenerateInMemoryResult, got %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("refine failed: %v", result.Err)
	}
	if !result.Success {
		t.Fatalf("expected success true")
	}
	if result.Plan == nil {
		t.Fatalf("expected plan in result")
	}
	item, ok := result.Plan.Items["task"]
	if !ok {
		t.Fatalf("expected refined plan item")
	}
	if item.Title != "Updated Task" {
		t.Fatalf("unexpected title: %q", item.Title)
	}
}
