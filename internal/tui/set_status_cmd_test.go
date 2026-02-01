package tui

import (
	"os"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestSetStatusCmdUpdatesPlanInProcess(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	parentID := "parent"
	childID := "child"
	parent := makeTestItem(parentID, plan.StatusTodo)
	child := makeTestItem(childID, plan.StatusTodo)
	child.ParentID = &parentID
	parent.ChildIDs = []string{childID}

	graph := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
		},
	}

	planFile := plan.PlanPath()
	if err := plan.SaveAtomic(planFile, graph); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	cmd := SetStatusCmd(childID, "done")
	msg := cmd()
	result, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if result.Err != nil {
		t.Fatalf("SetStatusCmd failed: %v", result.Err)
	}
	if !result.Success {
		t.Fatalf("expected success true")
	}

	updated, err := plan.Load(planFile)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	updatedChild := updated.Items[childID]
	updatedParent := updated.Items[parentID]
	if updatedChild.Status != plan.StatusDone {
		t.Fatalf("child status=%s want %s", updatedChild.Status, plan.StatusDone)
	}
	if updatedParent.Status != plan.StatusDone {
		t.Fatalf("parent status=%s want %s", updatedParent.Status, plan.StatusDone)
	}
	if !updatedChild.UpdatedAt.After(child.UpdatedAt) {
		t.Fatalf("child updatedAt was not updated")
	}
	if !updatedParent.UpdatedAt.After(parent.UpdatedAt) {
		t.Fatalf("parent updatedAt was not updated")
	}
	if time.Since(updatedChild.UpdatedAt) < 0 {
		t.Fatalf("unexpected updatedAt in the future")
	}
}
