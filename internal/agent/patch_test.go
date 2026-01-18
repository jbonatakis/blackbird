package agent

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestApplyPatch_AddAndDeps(t *testing.T) {
	now := time.Now().UTC()
	g := plan.NewEmptyWorkGraph()
	root := plan.WorkItem{
		ID:                 "root",
		Title:              "Root",
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := plan.AddItem(&g, root, nil, nil, now); err != nil {
		t.Fatalf("add root: %v", err)
	}

	parentID := "root"
	ops := []PatchOp{
		{
			Op:       PatchAdd,
			ParentID: &parentID,
			Item: &plan.WorkItem{
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do work",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
			},
		},
		{
			Op:    PatchAddDep,
			ID:    "task",
			DepID: "root",
			DepRationale: map[string]string{
				"root": "must exist first",
			},
		},
	}

	if err := ApplyPatch(&g, ops, now); err != nil {
		t.Fatalf("apply patch: %v", err)
	}

	task, ok := g.Items["task"]
	if !ok {
		t.Fatalf("task not found after patch")
	}
	if task.ParentID == nil || *task.ParentID != "root" {
		t.Fatalf("expected task parent root, got %v", task.ParentID)
	}
	if len(task.Deps) != 1 || task.Deps[0] != "root" {
		t.Fatalf("expected deps [root], got %v", task.Deps)
	}
	if task.DepRationale["root"] != "must exist first" {
		t.Fatalf("expected dep rationale set, got %v", task.DepRationale)
	}
}

func TestApplyPatch_DeleteWithChildrenFails(t *testing.T) {
	now := time.Now().UTC()
	g := plan.NewEmptyWorkGraph()
	root := plan.WorkItem{
		ID:                 "root",
		Title:              "Root",
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	child := plan.WorkItem{
		ID:                 "child",
		Title:              "Child",
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := plan.AddItem(&g, root, nil, nil, now); err != nil {
		t.Fatalf("add root: %v", err)
	}
	pid := "root"
	if err := plan.AddItem(&g, child, &pid, nil, now); err != nil {
		t.Fatalf("add child: %v", err)
	}

	ops := []PatchOp{
		{Op: PatchDelete, ID: "root"},
	}
	if err := ApplyPatch(&g, ops, now); err == nil {
		t.Fatalf("expected delete error due to children, got nil")
	}
}
