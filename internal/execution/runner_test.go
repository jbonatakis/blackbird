package execution

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunExecuteResultsAndStatusUpdates(t *testing.T) {
	tests := []struct {
		name        string
		items       map[string]plan.WorkItem
		runtime     agent.Runtime
		expectRuns  int
		expectDone  map[string]plan.Status
		wantReason  ExecuteStopReason
		wantTaskID  string
		cancelAfter bool
	}{
		{
			name: "no ready tasks",
			items: map[string]plan.WorkItem{
				"task": makeItem("task", plan.StatusDone),
			},
			runtime:    agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
			expectRuns: 0,
			expectDone: map[string]plan.Status{
				"task": plan.StatusDone,
			},
			wantReason: ExecuteReasonCompleted,
		},
		{
			name: "executes ready tasks in order",
			items: map[string]plan.WorkItem{
				"a": makeItem("a", plan.StatusTodo),
				"b": makeItem("b", plan.StatusTodo),
			},
			runtime:    agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
			expectRuns: 2,
			expectDone: map[string]plan.Status{
				"a": plan.StatusDone,
				"b": plan.StatusDone,
			},
			wantReason: ExecuteReasonCompleted,
		},
		{
			name: "waiting user stops loop",
			items: map[string]plan.WorkItem{
				"task": makeItem("task", plan.StatusTodo),
			},
			runtime: agent.Runtime{
				Command:  `printf '{"tool":"AskUserQuestion","id":"q1","prompt":"Name?"}'`,
				UseShell: true,
				Timeout:  2 * time.Second,
			},
			expectRuns: 1,
			expectDone: map[string]plan.Status{
				"task": plan.StatusWaitingUser,
			},
			wantReason: ExecuteReasonWaitingUser,
			wantTaskID: "task",
		},
		{
			name: "failed task updates status and loop continues",
			items: map[string]plan.WorkItem{
				"a": makeItem("a", plan.StatusTodo),
				"b": makeItem("b", plan.StatusTodo),
			},
			runtime: agent.Runtime{
				Command:  "exit 2",
				UseShell: true,
				Timeout:  2 * time.Second,
			},
			expectRuns: 2,
			expectDone: map[string]plan.Status{
				"a": plan.StatusFailed,
				"b": plan.StatusFailed,
			},
			wantReason: ExecuteReasonCompleted,
		},
		{
			name: "context cancel stops after first task",
			items: map[string]plan.WorkItem{
				"a": makeItem("a", plan.StatusTodo),
				"b": makeItem("b", plan.StatusTodo),
			},
			runtime:    agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
			expectRuns: 1,
			expectDone: map[string]plan.Status{
				"a": plan.StatusDone,
				"b": plan.StatusTodo,
			},
			wantReason:  ExecuteReasonCanceled,
			wantTaskID:  "a",
			cancelAfter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			planPath := filepath.Join(tempDir, "blackbird.plan.json")

			g := plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items:         tt.items,
			}
			if err := plan.SaveAtomic(planPath, g); err != nil {
				t.Fatalf("save plan: %v", err)
			}

			ctx := context.Background()
			var cancel context.CancelFunc
			if tt.cancelAfter {
				ctx, cancel = context.WithCancel(ctx)
			}

			var started []string
			result, err := RunExecute(ctx, ExecuteConfig{
				PlanPath: planPath,
				Runtime:  tt.runtime,
				OnTaskStart: func(taskID string) {
					started = append(started, taskID)
				},
				OnTaskFinish: func(taskID string, record RunRecord, execErr error) {
					if tt.cancelAfter && cancel != nil {
						cancel()
					}
				},
			})
			if err != nil && tt.wantReason != ExecuteReasonError {
				t.Fatalf("RunExecute: %v", err)
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("expected reason %s, got %s", tt.wantReason, result.Reason)
			}
			if tt.wantTaskID != "" && result.TaskID != tt.wantTaskID {
				t.Fatalf("expected task id %s, got %s", tt.wantTaskID, result.TaskID)
			}

			updated, err := plan.Load(planPath)
			if err != nil {
				t.Fatalf("load plan: %v", err)
			}
			for id, wantStatus := range tt.expectDone {
				if updated.Items[id].Status != wantStatus {
					t.Fatalf("expected %s status %s, got %s", id, wantStatus, updated.Items[id].Status)
				}
			}

			for _, id := range started {
				if _, ok := tt.expectDone[id]; !ok {
					t.Fatalf("unexpected task started: %s", id)
				}
			}

			totalRuns := countRuns(t, tempDir, tt.items)
			if totalRuns != tt.expectRuns {
				t.Fatalf("expected %d runs, got %d", tt.expectRuns, totalRuns)
			}
		})
	}
}

func TestRunExecuteReadyOrder(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"b": makeItem("b", plan.StatusTodo),
			"a": makeItem("a", plan.StatusTodo),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	var started []string
	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath: planPath,
		Runtime:  agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
		OnTaskStart: func(taskID string) {
			started = append(started, taskID)
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonCompleted {
		t.Fatalf("expected completed, got %s", result.Reason)
	}
	if len(started) != 2 || started[0] != "a" || started[1] != "b" {
		t.Fatalf("expected order [a b], got %v", started)
	}
}

func TestRunExecuteSkipsParentReviewWhenDisabled(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	parentID := "parent"
	childID := "child"
	otherID := "other"

	parent := makeItem(parentID, plan.StatusTodo)
	parent.ChildIDs = []string{childID}
	parent.AcceptanceCriteria = []string{"Parent criteria must hold across child outputs."}

	child := makeItem(childID, plan.StatusTodo)
	childParent := parentID
	child.ParentID = &childParent

	other := makeItem(otherID, plan.StatusTodo)

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
			otherID:  other,
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		ParentReviewEnabled: false,
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":false,"resumeTaskIds":["child"],"feedbackForResume":"Fix child output before continuing."}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonCompleted {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonCompleted)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items[childID].Status != plan.StatusDone {
		t.Fatalf("child status = %q, want %q", updated.Items[childID].Status, plan.StatusDone)
	}
	if updated.Items[otherID].Status != plan.StatusDone {
		t.Fatalf("other status = %q, want %q", updated.Items[otherID].Status, plan.StatusDone)
	}

	parentRuns, err := ListRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListRuns(%s): %v", parentID, err)
	}
	if len(parentRuns) != 0 {
		t.Fatalf("expected 0 parent review runs when disabled, got %d", len(parentRuns))
	}
}

func TestRunExecuteStopsForParentReviewRequired(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	parentID := "parent"
	childID := "child"
	otherID := "other"

	parent := makeItem(parentID, plan.StatusTodo)
	parent.ChildIDs = []string{childID}
	parent.AcceptanceCriteria = []string{"Parent criteria must hold across child outputs."}

	child := makeItem(childID, plan.StatusTodo)
	childParent := parentID
	child.ParentID = &childParent

	other := makeItem(otherID, plan.StatusTodo)

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
			otherID:  other,
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		ParentReviewEnabled: true,
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":false,"resumeTaskIds":["child"],"feedbackForResume":"Fix child output before continuing."}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonParentReviewRequired {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonParentReviewRequired)
	}
	if result.TaskID != parentID {
		t.Fatalf("result.TaskID = %q, want %q", result.TaskID, parentID)
	}
	if result.Run == nil {
		t.Fatalf("expected parent review run in execute result")
	}
	if result.Run.Type != RunTypeReview {
		t.Fatalf("result.Run.Type = %q, want %q", result.Run.Type, RunTypeReview)
	}
	if result.Run.TaskID != parentID {
		t.Fatalf("result.Run.TaskID = %q, want %q", result.Run.TaskID, parentID)
	}
	if result.Run.ParentReviewPassed == nil || *result.Run.ParentReviewPassed {
		t.Fatalf("result.Run.ParentReviewPassed = %#v, want false", result.Run.ParentReviewPassed)
	}
	if len(result.Run.ParentReviewResumeTaskIDs) != 1 || result.Run.ParentReviewResumeTaskIDs[0] != childID {
		t.Fatalf("result.Run.ParentReviewResumeTaskIDs = %#v, want [child]", result.Run.ParentReviewResumeTaskIDs)
	}
	if result.Run.ParentReviewFeedback != "Fix child output before continuing." {
		t.Fatalf("result.Run.ParentReviewFeedback = %q", result.Run.ParentReviewFeedback)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items[childID].Status != plan.StatusDone {
		t.Fatalf("child status = %q, want %q", updated.Items[childID].Status, plan.StatusDone)
	}
	if updated.Items[otherID].Status != plan.StatusTodo {
		t.Fatalf("other status = %q, want %q", updated.Items[otherID].Status, plan.StatusTodo)
	}

	childRuns, err := ListRuns(tempDir, childID)
	if err != nil {
		t.Fatalf("ListRuns(%s): %v", childID, err)
	}
	if len(childRuns) != 1 {
		t.Fatalf("expected 1 child run, got %d", len(childRuns))
	}

	parentRuns, err := ListRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListRuns(%s): %v", parentID, err)
	}
	if len(parentRuns) != 1 {
		t.Fatalf("expected 1 parent review run, got %d", len(parentRuns))
	}
	if parentRuns[0].Type != RunTypeReview {
		t.Fatalf("parent run type = %q, want %q", parentRuns[0].Type, RunTypeReview)
	}

	otherRuns, err := ListRuns(tempDir, otherID)
	if err != nil {
		t.Fatalf("ListRuns(%s): %v", otherID, err)
	}
	if len(otherRuns) != 0 {
		t.Fatalf("expected 0 other runs, got %d", len(otherRuns))
	}
}

func TestRunExecuteCompletedIncludesLatestParentReviewRunWhenReviewPasses(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	parentID := "parent"
	childID := "child"

	parent := makeItem(parentID, plan.StatusTodo)
	parent.ChildIDs = []string{childID}
	parent.AcceptanceCriteria = []string{"Parent criteria must hold across child outputs."}

	child := makeItem(childID, plan.StatusTodo)
	childParent := parentID
	child.ParentID = &childParent

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            planPath,
		ParentReviewEnabled: true,
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":true}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("RunExecute: %v", err)
	}
	if result.Reason != ExecuteReasonCompleted {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonCompleted)
	}
	if result.Run == nil {
		t.Fatalf("expected latest parent review run in completed execute result")
	}
	if result.Run.Type != RunTypeReview {
		t.Fatalf("result.Run.Type = %q, want %q", result.Run.Type, RunTypeReview)
	}
	if result.Run.TaskID != parentID {
		t.Fatalf("result.Run.TaskID = %q, want %q", result.Run.TaskID, parentID)
	}
	if result.Run.ParentReviewPassed == nil || !*result.Run.ParentReviewPassed {
		t.Fatalf("result.Run.ParentReviewPassed = %#v, want true", result.Run.ParentReviewPassed)
	}
	if len(result.Run.ParentReviewResumeTaskIDs) != 0 {
		t.Fatalf("result.Run.ParentReviewResumeTaskIDs = %#v, want empty", result.Run.ParentReviewResumeTaskIDs)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items[childID].Status != plan.StatusDone {
		t.Fatalf("child status = %q, want %q", updated.Items[childID].Status, plan.StatusDone)
	}

	parentRuns, err := ListReviewRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns(%s): %v", parentID, err)
	}
	if len(parentRuns) != 1 {
		t.Fatalf("expected 1 parent review run, got %d", len(parentRuns))
	}
}

func TestRunResumeUpdatesStatusAndReturnsRecord(t *testing.T) {
	tests := []struct {
		name          string
		runtime       agent.Runtime
		expectStatus  RunStatus
		expectPlan    plan.Status
		expectErr     bool
		cancelContext bool
	}{
		{
			name:         "resume completes",
			runtime:      agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
			expectStatus: RunStatusSuccess,
			expectPlan:   plan.StatusDone,
		},
		{
			name: "resume waiting user",
			runtime: agent.Runtime{
				Command:  `printf '{"tool":"AskUserQuestion","id":"q2","prompt":"More?"}'`,
				UseShell: true,
				Timeout:  2 * time.Second,
			},
			expectStatus: RunStatusWaitingUser,
			expectPlan:   plan.StatusWaitingUser,
		},
		{
			name:          "resume canceled context",
			runtime:       agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
			expectErr:     true,
			cancelContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			planPath := filepath.Join(tempDir, "blackbird.plan.json")

			g := plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"task": makeItem("task", plan.StatusWaitingUser),
				},
			}
			if err := plan.SaveAtomic(planPath, g); err != nil {
				t.Fatalf("save plan: %v", err)
			}

			waitingRun := RunRecord{
				ID:        "run-wait",
				TaskID:    "task",
				StartedAt: time.Date(2026, 1, 30, 13, 0, 0, 0, time.UTC),
				Status:    RunStatusWaitingUser,
				Stdout:    `{"tool":"AskUserQuestion","id":"q1","prompt":"Name?"}`,
				Context: ContextPack{
					SchemaVersion: ContextPackSchemaVersion,
					Task:          TaskContext{ID: "task", Title: "Task"},
				},
			}
			if err := SaveRun(tempDir, waitingRun); err != nil {
				t.Fatalf("SaveRun: %v", err)
			}

			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			record, err := RunResume(ctx, ResumeConfig{
				PlanPath: planPath,
				TaskID:   "task",
				Answers: []agent.Answer{{
					ID:    "q1",
					Value: "answer",
				}},
				Runtime: tt.runtime,
			})
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RunResume: %v", err)
			}
			if record.Status != tt.expectStatus {
				t.Fatalf("expected status %s, got %s", tt.expectStatus, record.Status)
			}

			updated, err := plan.Load(planPath)
			if err != nil {
				t.Fatalf("load plan: %v", err)
			}
			if updated.Items["task"].Status != tt.expectPlan {
				t.Fatalf("expected plan status %s, got %s", tt.expectPlan, updated.Items["task"].Status)
			}
		})
	}
}

func TestRunResumeRejectsMixedAnswersAndFeedbackWithoutLaunching(t *testing.T) {
	tests := []struct {
		name             string
		explicitFeedback string
		seedPending      bool
		wantErr          string
	}{
		{
			name:             "explicit feedback and answers",
			explicitFeedback: "apply the fixes and rerun tests",
			wantErr:          "resume answers cannot be combined with feedback-based resume; provide either answers or feedback",
		},
		{
			name:        "pending feedback and answers",
			seedPending: true,
			wantErr:     `resume answers cannot be combined with pending parent-review feedback from "parent-1" (review run "review-1"); retry with answers omitted`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			planPath := filepath.Join(tempDir, "blackbird.plan.json")

			g := plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"task": makeItem("task", plan.StatusWaitingUser),
				},
			}
			if err := plan.SaveAtomic(planPath, g); err != nil {
				t.Fatalf("save plan: %v", err)
			}

			if tt.seedPending {
				if _, err := upsertPendingParentReviewFeedback(
					tempDir,
					"task",
					"parent-1",
					"review-1",
					"fix failing acceptance checks",
					time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC),
				); err != nil {
					t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
				}
			}

			startAttempts := 0
			_, err := RunResume(context.Background(), ResumeConfig{
				PlanPath: planPath,
				TaskID:   "task",
				Feedback: tt.explicitFeedback,
				Runtime:  agent.Runtime{Provider: "test", Command: "cat", Timeout: 2 * time.Second},
				OnTaskStart: func(string) {
					startAttempts++
				},
				Answers: []agent.Answer{{
					ID:    "q1",
					Value: "answer",
				}},
			})
			if err == nil {
				t.Fatalf("expected error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
			if startAttempts != 0 {
				t.Fatalf("expected zero resume start attempts, got %d", startAttempts)
			}

			runs, err := ListRuns(tempDir, "task")
			if err != nil {
				t.Fatalf("ListRuns: %v", err)
			}
			if len(runs) != 0 {
				t.Fatalf("expected no new runs, got %d", len(runs))
			}

			updated, err := plan.Load(planPath)
			if err != nil {
				t.Fatalf("load plan: %v", err)
			}
			if updated.Items["task"].Status != plan.StatusWaitingUser {
				t.Fatalf("plan status = %q, want %q", updated.Items["task"].Status, plan.StatusWaitingUser)
			}
		})
	}
}

func TestRunResumeConsumesAndClearsPendingParentFeedback(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeItem("task", plan.StatusWaitingUser),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	previousRun := RunRecord{
		ID:                 "run-previous",
		TaskID:             "task",
		Provider:           "codex",
		ProviderSessionRef: "session-123",
		StartedAt:          time.Date(2026, 2, 9, 11, 30, 0, 0, time.UTC),
		Status:             RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task", Title: "Task"},
		},
	}
	if err := SaveRun(tempDir, previousRun); err != nil {
		t.Fatalf("SaveRun(previousRun): %v", err)
	}

	if _, err := upsertPendingParentReviewFeedback(
		tempDir,
		"task",
		"parent-1",
		"review-7",
		"  address review feedback and retry  ",
		time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	record, err := RunResume(context.Background(), ResumeConfig{
		PlanPath: planPath,
		TaskID:   "task",
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  "true",
			Timeout:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("RunResume: %v", err)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("record.Status = %q, want %q", record.Status, RunStatusSuccess)
	}
	if record.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected parentReviewFeedback in resume record context")
	}
	wantFeedback := ParentReviewFeedbackContext{
		ParentTaskID: "parent-1",
		ReviewRunID:  "review-7",
		Feedback:     "address review feedback and retry",
	}
	if *record.Context.ParentReviewFeedback != wantFeedback {
		t.Fatalf("record parent feedback = %#v, want %#v", *record.Context.ParentReviewFeedback, wantFeedback)
	}

	// Run persistence must succeed before pending feedback is cleared.
	runs, err := ListRuns(tempDir, "task")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 run records, got %d", len(runs))
	}

	persisted, err := LoadRun(tempDir, "task", record.ID)
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	if persisted.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected persisted context to include parentReviewFeedback")
	}
	if *persisted.Context.ParentReviewFeedback != wantFeedback {
		t.Fatalf(
			"persisted parent feedback = %#v, want %#v",
			*persisted.Context.ParentReviewFeedback,
			wantFeedback,
		)
	}

	pending, err := LoadPendingParentReviewFeedback(tempDir, "task")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected pending feedback to be cleared, got %#v", pending)
	}

	updated, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["task"].Status != plan.StatusDone {
		t.Fatalf("plan status = %q, want %q", updated.Items["task"].Status, plan.StatusDone)
	}
}

func TestRunResumeLeavesPendingFeedbackWhenFeedbackResumeCannotStart(t *testing.T) {
	tests := []struct {
		name                string
		previousProvider    string
		previousProviderRef string
		runtimeProvider     string
		wantErrSubstring    string
	}{
		{
			name:                "missing provider session ref",
			previousProvider:    "codex",
			previousProviderRef: "",
			runtimeProvider:     "codex",
			wantErrSubstring:    "provider session ref",
		},
		{
			name:                "provider mismatch",
			previousProvider:    "codex",
			previousProviderRef: "session-123",
			runtimeProvider:     "claude",
			wantErrSubstring:    "provider mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			planPath := filepath.Join(tempDir, "blackbird.plan.json")

			g := plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"task": makeItem("task", plan.StatusWaitingUser),
				},
			}
			if err := plan.SaveAtomic(planPath, g); err != nil {
				t.Fatalf("save plan: %v", err)
			}

			previousRun := RunRecord{
				ID:                 "run-previous",
				TaskID:             "task",
				Provider:           tt.previousProvider,
				ProviderSessionRef: tt.previousProviderRef,
				StartedAt:          time.Date(2026, 2, 9, 11, 30, 0, 0, time.UTC),
				Status:             RunStatusSuccess,
				Context: ContextPack{
					SchemaVersion: ContextPackSchemaVersion,
					Task:          TaskContext{ID: "task", Title: "Task"},
				},
			}
			if err := SaveRun(tempDir, previousRun); err != nil {
				t.Fatalf("SaveRun(previousRun): %v", err)
			}

			pendingBefore, err := upsertPendingParentReviewFeedback(
				tempDir,
				"task",
				"parent-1",
				"review-7",
				"address review feedback and retry",
				time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
			)
			if err != nil {
				t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
			}

			startAttempts := 0
			_, err = RunResume(context.Background(), ResumeConfig{
				PlanPath: planPath,
				TaskID:   "task",
				Runtime: agent.Runtime{
					Provider: tt.runtimeProvider,
					Command:  "true",
					Timeout:  2 * time.Second,
				},
				OnTaskStart: func(string) {
					startAttempts++
				},
			})
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstring) {
				t.Fatalf("error = %q", err.Error())
			}
			if startAttempts != 0 {
				t.Fatalf("expected zero resume start attempts, got %d", startAttempts)
			}

			pending, err := LoadPendingParentReviewFeedback(tempDir, "task")
			if err != nil {
				t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
			}
			if pending == nil {
				t.Fatalf("expected pending feedback to remain")
			}
			if *pending != pendingBefore {
				t.Fatalf("pending feedback changed: got %#v want %#v", *pending, pendingBefore)
			}

			runs, err := ListRuns(tempDir, "task")
			if err != nil {
				t.Fatalf("ListRuns: %v", err)
			}
			if len(runs) != 1 {
				t.Fatalf("expected 1 run record, got %d", len(runs))
			}

			updated, err := plan.Load(planPath)
			if err != nil {
				t.Fatalf("load plan: %v", err)
			}
			if updated.Items["task"].Status != plan.StatusWaitingUser {
				t.Fatalf("plan status = %q, want %q", updated.Items["task"].Status, plan.StatusWaitingUser)
			}
		})
	}
}

func TestRunResumeLeavesPendingFeedbackWhenRunPersistenceFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission semantics are not reliable on windows")
	}

	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "blackbird.plan.json")

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": makeItem("task", plan.StatusWaitingUser),
		},
	}
	if err := plan.SaveAtomic(planPath, g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	previousRun := RunRecord{
		ID:                 "run-previous",
		TaskID:             "task",
		Provider:           "codex",
		ProviderSessionRef: "session-123",
		StartedAt:          time.Date(2026, 2, 9, 11, 30, 0, 0, time.UTC),
		Status:             RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task", Title: "Task"},
		},
	}
	if err := SaveRun(tempDir, previousRun); err != nil {
		t.Fatalf("SaveRun(previousRun): %v", err)
	}

	pendingBefore, err := upsertPendingParentReviewFeedback(
		tempDir,
		"task",
		"parent-1",
		"review-7",
		"address review feedback and retry",
		time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	runsTaskDir := filepath.Join(tempDir, runsDirName, "task")
	if err := os.Chmod(runsTaskDir, 0o555); err != nil {
		t.Fatalf("chmod runs task dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(runsTaskDir, 0o755) })

	_, err = RunResume(context.Background(), ResumeConfig{
		PlanPath: planPath,
		TaskID:   "task",
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  "true",
			Timeout:  2 * time.Second,
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "write run record") {
		t.Fatalf("error = %q", err.Error())
	}

	pending, err := LoadPendingParentReviewFeedback(tempDir, "task")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if pending == nil {
		t.Fatalf("expected pending feedback to remain after persistence failure")
	}
	if *pending != pendingBefore {
		t.Fatalf("pending feedback changed: got %#v want %#v", *pending, pendingBefore)
	}

	runs, err := ListRuns(tempDir, "task")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run record, got %d", len(runs))
	}
}

func makeItem(id string, status plan.Status) plan.WorkItem {
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
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

func countRuns(t *testing.T, baseDir string, items map[string]plan.WorkItem) int {
	t.Helper()
	total := 0
	for id := range items {
		runs, err := ListRuns(baseDir, id)
		if err != nil {
			t.Fatalf("ListRuns(%s): %v", id, err)
		}
		total += len(runs)
	}
	return total
}
