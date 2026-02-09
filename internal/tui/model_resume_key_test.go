package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestUpdateResumeKeyStartsDirectResumeWhenPendingParentFeedbackExists(t *testing.T) {
	now := time.Date(2026, 2, 9, 5, 0, 0, 0, time.UTC)
	model := Model{
		viewMode:   ViewModeMain,
		planExists: true,
		selectedID: "task-1",
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID:        "task-1",
					Status:    plan.StatusDone,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{
			"task-1": {
				ParentTaskID: "parent-1",
				ReviewRunID:  "review-1",
				Feedback:     "retry after parent review",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
	}

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	updated := updatedModel.(Model)

	if !updated.actionInProgress {
		t.Fatalf("expected actionInProgress to be true")
	}
	if updated.actionName != "Resuming..." {
		t.Fatalf("actionName = %q, want Resuming...", updated.actionName)
	}
	if updated.pendingResumeTask != "" {
		t.Fatalf("pendingResumeTask = %q, want empty for direct feedback resume", updated.pendingResumeTask)
	}
	if updated.agentQuestionForm != nil {
		t.Fatalf("expected no question modal for pending parent feedback resume")
	}
	if cmd == nil {
		t.Fatalf("expected resume command to be returned")
	}
}

func TestUpdateResumeKeyWaitingUserPathUnchangedOpensQuestionModal(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 2, 9, 6, 0, 0, 0, time.UTC)
	waitingRun := execution.RunRecord{
		ID:        "run-waiting-1",
		TaskID:    "task-1",
		StartedAt: now,
		Status:    execution.RunStatusWaitingUser,
		Stdout:    `{"tool":"AskUserQuestion","id":"q1","prompt":"Please confirm the API base URL."}`,
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task: execution.TaskContext{
				ID:    "task-1",
				Title: "Task 1",
			},
		},
	}
	if err := execution.SaveRun(tempDir, waitingRun); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	model := Model{
		viewMode:   ViewModeMain,
		planExists: true,
		selectedID: "task-1",
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID:        "task-1",
					Status:    plan.StatusWaitingUser,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		runData: map[string]execution.RunRecord{
			"task-1": waitingRun,
		},
		pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
		windowWidth:           120,
		windowHeight:          32,
	}

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	updated := updatedModel.(Model)

	if cmd != nil {
		t.Fatalf("expected no command when opening waiting-user question modal")
	}
	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false while user answers waiting questions")
	}
	if updated.actionMode != ActionModeAgentQuestion {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeAgentQuestion)
	}
	if updated.pendingResumeTask != "task-1" {
		t.Fatalf("pendingResumeTask = %q, want task-1", updated.pendingResumeTask)
	}
	if updated.agentQuestionForm == nil {
		t.Fatalf("expected agent question modal to open")
	}
	if got := updated.agentQuestionForm.CurrentQuestion().ID; got != "q1" {
		t.Fatalf("CurrentQuestion().ID = %q, want q1", got)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parent review modal to remain closed for waiting-user resume")
	}
}
