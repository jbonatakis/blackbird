package execution

import (
	"context"
	"path/filepath"
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
