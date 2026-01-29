package execution

import (
	"context"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestExecuteTask(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "test",
		Command:  "cat",
		Timeout:  2 * time.Second,
	}

	now := time.Date(2026, 1, 28, 23, 0, 0, 0, time.UTC)
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

	record, err := ExecuteTask(context.Background(), g, g.Items["task"], runtime)
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", record.Status)
	}
	if record.TaskID != "task" {
		t.Fatalf("expected task id, got %s", record.TaskID)
	}
}
