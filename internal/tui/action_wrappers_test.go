package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestExecuteCmdWithContextRunsInProcess(t *testing.T) {
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

	planPath := filepath.Join(tempDir, plan.DefaultPlanFilename)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeTestItem("task", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	cmd := ExecuteCmdWithContext(context.Background())
	msg := cmd()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if complete.Action != "execute" || !complete.Success {
		t.Fatalf("expected execute success, got action=%s success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}
	if complete.Output == "" {
		t.Fatalf("expected output summary")
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["task"].Status != plan.StatusDone {
		t.Fatalf("expected task done, got %s", updated.Items["task"].Status)
	}
}

func TestResumeCmdWithContextRunsInProcess(t *testing.T) {
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

	planPath := filepath.Join(tempDir, plan.DefaultPlanFilename)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeTestItem("task", plan.StatusWaitingUser),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	waitingRun := execution.RunRecord{
		ID:        "run-wait",
		TaskID:    "task",
		StartedAt: time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC),
		Status:    execution.RunStatusWaitingUser,
		Stdout:    `{"tool":"AskUserQuestion","id":"q1","prompt":"Name?"}`,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task:          execution.TaskContext{ID: "task", Title: "Task"},
		},
	}
	if err := execution.SaveRun(tempDir, waitingRun); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	cmd := ResumeCmdWithContext(context.Background(), "task", []agent.Answer{{
		ID:    "q1",
		Value: "answer",
	}})
	msg := cmd()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success, got action=%s success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}
	if complete.Output == "" {
		t.Fatalf("expected output summary")
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["task"].Status != plan.StatusDone {
		t.Fatalf("expected task done, got %s", updated.Items["task"].Status)
	}
}

func makeTestItem(id string, status plan.Status) plan.WorkItem {
	now := time.Date(2026, 1, 31, 8, 0, 0, 0, time.UTC)
	return plan.WorkItem{
		ID:                 id,
		Title:              "Task " + id,
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "do it",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             status,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
