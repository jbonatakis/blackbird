package tui

import (
	"context"
	"os"
	"strings"
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

	planPath := plan.PlanPath()
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

	planPath := plan.PlanPath()
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

func TestResumeCmdWithContextRejectsAnswersWhenPendingFeedbackExists(t *testing.T) {
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

	planPath := plan.PlanPath()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeTestItem("task", plan.StatusWaitingUser),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if _, err := execution.UpsertPendingParentReviewFeedback(
		tempDir,
		"task",
		"parent-1",
		"review-1",
		"apply requested changes",
	); err != nil {
		t.Fatalf("UpsertPendingParentReviewFeedback: %v", err)
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
	if complete.Err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(complete.Err.Error(), "cannot be combined with pending parent-review feedback") {
		t.Fatalf("error = %q", complete.Err.Error())
	}
}

func TestResumePendingParentFeedbackCmdWithContextRunsInProcess(t *testing.T) {
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

	planPath := plan.PlanPath()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeTestItem("task", plan.StatusDone),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)
	saveResumeFeedbackFixture(t, tempDir, "task", now)

	cmd := ResumePendingParentFeedbackCmdWithContext(context.Background(), "task")
	msg := cmd()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success, got action=%s success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}
	if complete.Output != "completed task" {
		t.Fatalf("output = %q, want %q", complete.Output, "completed task")
	}

	pending, err := execution.LoadPendingParentReviewFeedback(tempDir, "task")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected pending feedback to clear after successful resume, got %#v", pending)
	}
}

func TestResumePendingParentFeedbackTargetCmdWithContextUsesProvidedFeedback(t *testing.T) {
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

	planPath := plan.PlanPath()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeTestItem("task", plan.StatusDone),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	now := time.Date(2026, 2, 9, 9, 2, 0, 0, time.UTC)
	saveResumeFeedbackFixture(t, tempDir, "task", now)

	cmd := ResumePendingParentFeedbackTargetCmdWithContext(
		context.Background(),
		ResumePendingParentFeedbackTarget{
			TaskID:   "task",
			Feedback: "  apply targeted fix for task  ",
		},
	)
	msg := cmd()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success, got action=%s success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}

	latest, err := execution.GetLatestRun(tempDir, "task")
	if err != nil {
		t.Fatalf("GetLatestRun(task): %v", err)
	}
	if latest == nil {
		t.Fatalf("expected latest run for task")
	}
	if latest.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected ParentReviewFeedback in resumed run context")
	}
	if latest.Context.ParentReviewFeedback.Feedback != "apply targeted fix for task" {
		t.Fatalf(
			"ParentReviewFeedback.Feedback = %q, want %q",
			latest.Context.ParentReviewFeedback.Feedback,
			"apply targeted fix for task",
		)
	}
	if latest.Context.ParentReviewFeedback.ParentTaskID != "parent-1" {
		t.Fatalf(
			"ParentReviewFeedback.ParentTaskID = %q, want %q",
			latest.Context.ParentReviewFeedback.ParentTaskID,
			"parent-1",
		)
	}
	if latest.Context.ParentReviewFeedback.ReviewRunID != "review-task" {
		t.Fatalf(
			"ParentReviewFeedback.ReviewRunID = %q, want %q",
			latest.Context.ParentReviewFeedback.ReviewRunID,
			"review-task",
		)
	}

	pending, err := execution.LoadPendingParentReviewFeedback(tempDir, "task")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(task): %v", err)
	}
	if pending != nil {
		t.Fatalf("expected pending feedback cleared for task, got %#v", pending)
	}
}

func TestResumePendingParentFeedbackTargetsCmdWithContextReportsPerTaskResults(t *testing.T) {
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

	planPath := plan.PlanPath()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-a": makeTestItem("task-a", plan.StatusDone),
			"task-b": makeTestItem("task-b", plan.StatusDone),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	now := time.Date(2026, 2, 9, 9, 5, 0, 0, time.UTC)
	saveResumeFeedbackFixture(t, tempDir, "task-a", now)
	if _, err := execution.UpsertPendingParentReviewFeedback(
		tempDir,
		"task-b",
		"parent-1",
		"review-2",
		"retry child-b",
	); err != nil {
		t.Fatalf("UpsertPendingParentReviewFeedback(task-b): %v", err)
	}

	cmd := ResumePendingParentFeedbackTargetsCmdWithContext(
		context.Background(),
		[]string{" task-a ", "task-b", "task-a"},
	)
	msg := cmd()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success output, got action=%s success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}
	if complete.Err != nil {
		t.Fatalf("expected nil error, got %v", complete.Err)
	}

	lines := strings.Split(complete.Output, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 per-task output lines, got %d (%q)", len(lines), complete.Output)
	}
	if lines[0] != "completed task-a" {
		t.Fatalf("first output line = %q, want %q", lines[0], "completed task-a")
	}
	if !strings.Contains(lines[1], "failed task-b: no runs found for task-b") {
		t.Fatalf("second output line = %q", lines[1])
	}
}

func TestResumePendingParentFeedbackTargetsWithFeedbackCmdUsesPerTaskFeedback(t *testing.T) {
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

	planPath := plan.PlanPath()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-a": makeTestItem("task-a", plan.StatusDone),
			"task-b": makeTestItem("task-b", plan.StatusDone),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	now := time.Date(2026, 2, 9, 9, 8, 0, 0, time.UTC)
	saveResumeFeedbackFixture(t, tempDir, "task-a", now)
	saveResumeFeedbackFixture(t, tempDir, "task-b", now.Add(2*time.Minute))

	cmd := ResumePendingParentFeedbackTargetsWithFeedbackCmdWithContext(
		context.Background(),
		[]ResumePendingParentFeedbackTarget{
			{TaskID: "task-a", Feedback: "fix task-a precision and retry"},
			{TaskID: "task-b", Feedback: "fix task-b retry loop and retry"},
			{TaskID: "task-a", Feedback: "ignored duplicate"},
		},
	)
	msg := cmd()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete, got %T", msg)
	}
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success output, got action=%s success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}

	lines := strings.Split(complete.Output, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d (%q)", len(lines), complete.Output)
	}
	if lines[0] != "completed task-a" {
		t.Fatalf("first output line = %q, want completed task-a", lines[0])
	}
	if lines[1] != "completed task-b" {
		t.Fatalf("second output line = %q, want completed task-b", lines[1])
	}

	latestA, err := execution.GetLatestRun(tempDir, "task-a")
	if err != nil {
		t.Fatalf("GetLatestRun(task-a): %v", err)
	}
	if latestA == nil || latestA.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected task-a resumed run with parent-review feedback context")
	}
	if latestA.Context.ParentReviewFeedback.Feedback != "fix task-a precision and retry" {
		t.Fatalf(
			"task-a feedback = %q, want %q",
			latestA.Context.ParentReviewFeedback.Feedback,
			"fix task-a precision and retry",
		)
	}

	latestB, err := execution.GetLatestRun(tempDir, "task-b")
	if err != nil {
		t.Fatalf("GetLatestRun(task-b): %v", err)
	}
	if latestB == nil || latestB.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected task-b resumed run with parent-review feedback context")
	}
	if latestB.Context.ParentReviewFeedback.Feedback != "fix task-b retry loop and retry" {
		t.Fatalf(
			"task-b feedback = %q, want %q",
			latestB.Context.ParentReviewFeedback.Feedback,
			"fix task-b retry loop and retry",
		)
	}

	for _, taskID := range []string{"task-a", "task-b"} {
		pending, err := execution.LoadPendingParentReviewFeedback(tempDir, taskID)
		if err != nil {
			t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", taskID, err)
		}
		if pending != nil {
			t.Fatalf("expected pending feedback cleared for %s, got %#v", taskID, pending)
		}
	}
}

func TestResponseToPlanNormalizesFullPlanTimestamps(t *testing.T) {
	now := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	agentTime := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	resp := agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan: &plan.WorkGraph{
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
					CreatedAt:          agentTime,
					UpdatedAt:          agentTime,
				},
			},
		},
	}

	result, err := agent.ResponseToPlan(plan.NewEmptyWorkGraph(), resp, now)
	if err != nil {
		t.Fatalf("responseToPlan: %v", err)
	}
	if errs := plan.Validate(result); len(errs) != 0 {
		t.Fatalf("plan validation failed: %v", errs)
	}
	item := result.Items["task"]
	if !item.CreatedAt.Equal(now) || !item.UpdatedAt.Equal(now) {
		t.Fatalf("timestamps not normalized: got %s/%s want %s", item.CreatedAt, item.UpdatedAt, now)
	}
}

func TestContinuePlanGenerationWithAnswersTooManyRounds(t *testing.T) {
	cmd := ContinuePlanGenerationWithAnswers(
		"project",
		nil,
		"",
		[]agent.Answer{{ID: "q1", Value: "answer"}},
		agent.MaxPlanQuestionRounds,
	)

	msg := cmd()
	result, ok := msg.(PlanGenerateInMemoryResult)
	if !ok {
		t.Fatalf("expected PlanGenerateInMemoryResult, got %T", msg)
	}
	if result.Err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(result.Err.Error(), "too many clarification rounds") {
		t.Fatalf("error = %q, want clarification-rounds message", result.Err.Error())
	}
}

func TestContinuePlanRefineWithAnswersTooManyRounds(t *testing.T) {
	cmd := ContinuePlanRefineWithAnswers(
		"update plan",
		plan.NewEmptyWorkGraph(),
		[]agent.Answer{{ID: "q1", Value: "answer"}},
		agent.MaxPlanQuestionRounds,
	)

	msg := cmd()
	result, ok := msg.(PlanGenerateInMemoryResult)
	if !ok {
		t.Fatalf("expected PlanGenerateInMemoryResult, got %T", msg)
	}
	if result.Err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(result.Err.Error(), "too many clarification rounds") {
		t.Fatalf("error = %q, want clarification-rounds message", result.Err.Error())
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

func saveResumeFeedbackFixture(t *testing.T, baseDir string, taskID string, now time.Time) {
	t.Helper()

	previousRun := execution.RunRecord{
		ID:                 "run-previous-" + taskID,
		TaskID:             taskID,
		Provider:           "codex",
		ProviderSessionRef: "session-" + taskID,
		StartedAt:          now,
		Status:             execution.RunStatusSuccess,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task: execution.TaskContext{
				ID:    taskID,
				Title: "Task " + taskID,
			},
		},
	}
	if err := execution.SaveRun(baseDir, previousRun); err != nil {
		t.Fatalf("SaveRun(%s): %v", taskID, err)
	}

	if _, err := execution.UpsertPendingParentReviewFeedback(
		baseDir,
		taskID,
		"parent-1",
		"review-"+taskID,
		"retry "+taskID,
	); err != nil {
		t.Fatalf("UpsertPendingParentReviewFeedback(%s): %v", taskID, err)
	}
}
