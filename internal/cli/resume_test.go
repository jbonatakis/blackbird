package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunResumePromptsWaitingQuestionWhenNoPendingParentFeedback(t *testing.T) {
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

	now := time.Date(2026, 1, 28, 21, 0, 0, 0, time.UTC)
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
				Status:             plan.StatusWaitingUser,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	waitingRun := execution.RunRecord{
		ID:        "run-wait",
		TaskID:    "task",
		StartedAt: now,
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

	setPromptReader(strings.NewReader("answer\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error { return runResume("task") })
	if err != nil {
		t.Fatalf("runResume: %v", err)
	}
	if !strings.Contains(output, "Name?: ") {
		t.Fatalf("expected interactive question prompt output, got %q", output)
	}
	if !strings.Contains(output, "completed task") {
		t.Fatalf("expected completed output after answering waiting-user prompt, got %q", output)
	}
	if strings.Index(output, "Name?: ") > strings.Index(output, "completed task") {
		t.Fatalf("expected question prompt before completion output, got %q", output)
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
	if len(entries) != 2 {
		t.Fatalf("expected 2 run records, got %d", len(entries))
	}

	latest, err := execution.GetLatestRun(tempDir, "task")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil {
		t.Fatalf("expected latest run")
	}
	if len(latest.Context.Answers) != 1 {
		t.Fatalf("expected 1 answer in resumed context, got %d", len(latest.Context.Answers))
	}
	if latest.Context.Answers[0].ID != "q1" || latest.Context.Answers[0].Value != "answer" {
		t.Fatalf("unexpected resumed answer payload: %#v", latest.Context.Answers[0])
	}
}

func TestRunResumeUsesPendingParentFeedbackWithoutPromptingForAnswers(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "true")
	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "codex")

	now := time.Date(2026, 1, 28, 21, 0, 0, 0, time.UTC)
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
				Status:             plan.StatusWaitingUser,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	waitingRun := execution.RunRecord{
		ID:        "run-wait",
		TaskID:    "task",
		StartedAt: now,
		Status:    execution.RunStatusWaitingUser,
		Stdout:    `{"tool":"AskUserQuestion","id":"q1","prompt":"Name?"}`,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task:          execution.TaskContext{ID: "task", Title: "Task"},
		},
	}
	if err := execution.SaveRun(tempDir, waitingRun); err != nil {
		t.Fatalf("SaveRun(waiting): %v", err)
	}

	previousRun := execution.RunRecord{
		ID:                 "run-previous",
		TaskID:             "task",
		Provider:           "codex",
		ProviderSessionRef: "session-123",
		StartedAt:          now.Add(time.Minute),
		Status:             execution.RunStatusSuccess,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task:          execution.TaskContext{ID: "task", Title: "Task"},
		},
	}
	if err := execution.SaveRun(tempDir, previousRun); err != nil {
		t.Fatalf("SaveRun(previous): %v", err)
	}

	if _, err := execution.UpsertPendingParentReviewFeedback(
		tempDir,
		"task",
		"parent-1",
		"review-1",
		"address feedback and retry",
	); err != nil {
		t.Fatalf("UpsertPendingParentReviewFeedback: %v", err)
	}

	failReader := &readErrorReader{err: errors.New("prompt reader should not be used when pending parent feedback exists")}
	setPromptReader(failReader)
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error { return runResume("task") })
	if err != nil {
		t.Fatalf("runResume: %v", err)
	}
	if strings.TrimSpace(output) != "completed task" {
		t.Fatalf("expected deterministic completion output, got %q", output)
	}
	if failReader.called {
		t.Fatalf("expected no prompt reads when pending parent feedback exists")
	}
	if strings.Contains(output, "Name?: ") {
		t.Fatalf("expected no waiting-user prompt when pending parent feedback exists, got %q", output)
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
	if len(entries) != 3 {
		t.Fatalf("expected 3 run records, got %d", len(entries))
	}

	pending, err := execution.LoadPendingParentReviewFeedback(tempDir, "task")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected pending feedback cleared after successful resume, got %#v", pending)
	}
}

type readErrorReader struct {
	err    error
	called bool
}

func (r *readErrorReader) Read(_ []byte) (int, error) {
	r.called = true
	if r.err != nil {
		return 0, r.err
	}
	return 0, errors.New("read error")
}
