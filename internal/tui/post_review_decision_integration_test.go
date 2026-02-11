package tui

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestPostReviewContinueLeavesTaskStateUnchanged(t *testing.T) {
	now := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	g := postReviewIntegrationGraph(now, map[string]plan.Status{
		"parent-1":       plan.StatusDone,
		"child-fail-a":   plan.StatusFailed,
		"child-fail-b":   plan.StatusFailed,
		"child-pass-one": plan.StatusDone,
	})
	run := postReviewIntegrationRun(execution.ParentReviewTaskResults{
		"child-fail-a": {
			TaskID:   "child-fail-a",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Fix child-fail-a checks.",
		},
		"child-fail-b": {
			TaskID:   "child-fail-b",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Fix child-fail-b retries.",
		},
		"child-pass-one": {
			TaskID: "child-pass-one",
			Status: execution.ParentReviewTaskStatusPassed,
		},
	})

	model := postReviewIntegrationModel(g, run)
	model.selectedID = "child-fail-a"
	model.runData = map[string]execution.RunRecord{
		"child-fail-a": {
			ID:      "run-child-fail-a",
			TaskID:  "child-fail-a",
			Status:  execution.RunStatusFailed,
			Context: execution.ContextPack{Task: execution.TaskContext{ID: "child-fail-a", Title: "Task child-fail-a"}},
		},
	}
	model.pendingParentFeedback = map[string]execution.PendingParentReviewFeedback{
		"child-fail-a": {
			ParentTaskID: "parent-1",
			ReviewRunID:  "review-parent-1",
			Feedback:     "retry child-fail-a",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	beforePlan := plan.Clone(model.plan)
	beforeRunData := cloneRunRecordMap(model.runData)
	beforePending := clonePendingFeedbackMap(model.pendingParentFeedback)
	beforeSelected := model.selectedID

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)

	if cmd != nil {
		t.Fatalf("expected no command for continue branch")
	}
	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v", updated.actionMode, ActionModeNone)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared on continue")
	}
	if updated.actionInProgress {
		t.Fatalf("expected actionInProgress=false on continue")
	}
	if updated.selectedID != beforeSelected {
		t.Fatalf("selectedID = %q, want %q", updated.selectedID, beforeSelected)
	}
	if !reflect.DeepEqual(updated.plan, beforePlan) {
		t.Fatalf("expected continue to keep plan unchanged")
	}
	if !reflect.DeepEqual(updated.runData, beforeRunData) {
		t.Fatalf("expected continue to keep runData unchanged")
	}
	if !reflect.DeepEqual(updated.pendingParentFeedback, beforePending) {
		t.Fatalf("expected continue to keep pending parent feedback unchanged")
	}
}

func TestPostReviewDiscardCancelAndConfirmBranches(t *testing.T) {
	now := time.Date(2026, 1, 31, 9, 10, 0, 0, time.UTC)
	g := postReviewIntegrationGraph(now, map[string]plan.Status{
		"parent-1":     plan.StatusDone,
		"child-fail-a": plan.StatusFailed,
	})
	run := postReviewIntegrationRun(execution.ParentReviewTaskResults{
		"child-fail-a": {
			TaskID:   "child-fail-a",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Fix child-fail-a checks.",
		},
	})
	model := postReviewIntegrationModel(g, run)
	beforePlan := plan.Clone(model.plan)

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd != nil {
		t.Fatalf("expected no command when opening discard confirmation")
	}
	if updated.parentReviewForm == nil || updated.parentReviewForm.Mode() != ParentReviewModalModeConfirmDiscard {
		t.Fatalf("expected discard confirmation mode to open")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("actionMode = %v, want %v while confirming discard", updated.actionMode, ActionModeParentReview)
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd != nil {
		t.Fatalf("expected no command when canceling discard confirmation")
	}
	if updated.parentReviewForm == nil || updated.parentReviewForm.Mode() != ParentReviewModalModeActions {
		t.Fatalf("expected discard cancel to return to action mode")
	}
	if updated.actionMode != ActionModeParentReview {
		t.Fatalf("actionMode = %v, want %v after discard cancel", updated.actionMode, ActionModeParentReview)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated = updatedModel.(Model)
	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd != nil {
		t.Fatalf("expected no command for confirmed discard")
	}
	if updated.actionMode != ActionModeNone {
		t.Fatalf("actionMode = %v, want %v after discard confirm", updated.actionMode, ActionModeNone)
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared after discard confirm")
	}
	if !reflect.DeepEqual(updated.plan, beforePlan) {
		t.Fatalf("discard flow should not mutate task content/state")
	}
}

func TestPostReviewResumeAllFailedResumesOnlyFailedTasksWithReviewFeedback(t *testing.T) {
	tempDir := postReviewIntegrationWorkspace(t)
	now := time.Date(2026, 1, 31, 9, 20, 0, 0, time.UTC)

	g := postReviewIntegrationGraph(now, map[string]plan.Status{
		"parent-1":       plan.StatusDone,
		"child-fail-a":   plan.StatusDone,
		"child-fail-b":   plan.StatusDone,
		"child-pass-one": plan.StatusDone,
	})
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	postReviewSaveResumeFixture(t, tempDir, "child-fail-a", "stale feedback a", now)
	postReviewSaveResumeFixture(t, tempDir, "child-fail-b", "stale feedback b", now.Add(2*time.Minute))
	postReviewSaveResumeFixture(t, tempDir, "child-pass-one", "stale feedback pass", now.Add(4*time.Minute))

	run := postReviewIntegrationRun(execution.ParentReviewTaskResults{
		"child-fail-a": {
			TaskID:   "child-fail-a",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Use review feedback for child-fail-a.",
		},
		"child-fail-b": {
			TaskID:   "child-fail-b",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Use review feedback for child-fail-b.",
		},
		"child-pass-one": {
			TaskID: "child-pass-one",
			Status: execution.ParentReviewTaskStatusPassed,
		},
	})
	model := postReviewIntegrationModel(g, run)
	if got, want := model.parentReviewForm.ResumeTargets(), []string{"child-fail-a", "child-fail-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ResumeTargets() = %#v, want %#v", got, want)
	}

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated := updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if !updated.actionInProgress {
		t.Fatalf("expected actionInProgress=true after resume-all starts")
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared while resume-all runs")
	}

	complete := executePostReviewResumeCmd(t, cmd)
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success, got action=%q success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}
	if complete.Err != nil {
		t.Fatalf("expected nil error for resume-all, got %v", complete.Err)
	}
	lines := strings.Split(strings.TrimSpace(complete.Output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines for failed tasks only, got %d (%q)", len(lines), complete.Output)
	}
	if strings.Contains(complete.Output, "child-pass-one") {
		t.Fatalf("expected pass task to be excluded from bulk resume output, got %q", complete.Output)
	}

	postReviewAssertLatestFeedback(t, tempDir, "child-fail-a", "Use review feedback for child-fail-a.")
	postReviewAssertLatestFeedback(t, tempDir, "child-fail-b", "Use review feedback for child-fail-b.")

	for _, taskID := range []string{"child-fail-a", "child-fail-b"} {
		pending, err := execution.LoadPendingParentReviewFeedback(tempDir, taskID)
		if err != nil {
			t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", taskID, err)
		}
		if pending != nil {
			t.Fatalf("expected pending feedback cleared for %s, got %#v", taskID, pending)
		}
	}

	latestPass, err := execution.GetLatestRun(tempDir, "child-pass-one")
	if err != nil {
		t.Fatalf("GetLatestRun(child-pass-one): %v", err)
	}
	if latestPass == nil {
		t.Fatalf("expected existing run for child-pass-one")
	}
	if latestPass.ID != "run-previous-child-pass-one" {
		t.Fatalf("expected pass task run unchanged, got latest ID %q", latestPass.ID)
	}
	pendingPass, err := execution.LoadPendingParentReviewFeedback(tempDir, "child-pass-one")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-pass-one): %v", err)
	}
	if pendingPass == nil {
		t.Fatalf("expected pass task pending feedback to remain because it should not be resumed")
	}
}

func TestPostReviewResumeOneResumesSelectedTaskWithReviewFeedback(t *testing.T) {
	tempDir := postReviewIntegrationWorkspace(t)
	now := time.Date(2026, 1, 31, 9, 30, 0, 0, time.UTC)

	g := postReviewIntegrationGraph(now, map[string]plan.Status{
		"parent-1":     plan.StatusDone,
		"child-fail-a": plan.StatusDone,
		"child-fail-b": plan.StatusDone,
	})
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	postReviewSaveResumeFixture(t, tempDir, "child-fail-a", "stale feedback a", now)
	postReviewSaveResumeFixture(t, tempDir, "child-fail-b", "stale feedback b", now.Add(2*time.Minute))

	run := postReviewIntegrationRun(execution.ParentReviewTaskResults{
		"child-fail-a": {
			TaskID:   "child-fail-a",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Use review feedback for child-fail-a.",
		},
		"child-fail-b": {
			TaskID:   "child-fail-b",
			Status:   execution.ParentReviewTaskStatusFailed,
			Feedback: "Use review feedback for child-fail-b.",
		},
	})
	model := postReviewIntegrationModel(g, run)

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	updated := updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = updatedModel.(Model)
	if got, want := updated.parentReviewForm.SelectedTarget(), "child-fail-b"; got != want {
		t.Fatalf("SelectedTarget() = %q, want %q", got, want)
	}

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if !updated.actionInProgress {
		t.Fatalf("expected actionInProgress=true after resume-one starts")
	}
	if updated.parentReviewForm != nil {
		t.Fatalf("expected parentReviewForm cleared while resume-one runs")
	}

	complete := executePostReviewResumeCmd(t, cmd)
	if complete.Action != "resume" || !complete.Success {
		t.Fatalf("expected resume success, got action=%q success=%v err=%v", complete.Action, complete.Success, complete.Err)
	}
	if got, want := strings.TrimSpace(complete.Output), "completed child-fail-b"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	postReviewAssertLatestFeedback(t, tempDir, "child-fail-b", "Use review feedback for child-fail-b.")
	latestUnselected, err := execution.GetLatestRun(tempDir, "child-fail-a")
	if err != nil {
		t.Fatalf("GetLatestRun(child-fail-a): %v", err)
	}
	if latestUnselected == nil {
		t.Fatalf("expected existing run for child-fail-a")
	}
	if latestUnselected.ID != "run-previous-child-fail-a" {
		t.Fatalf("expected unselected task run unchanged, got latest ID %q", latestUnselected.ID)
	}
	pendingUnselected, err := execution.LoadPendingParentReviewFeedback(tempDir, "child-fail-a")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-fail-a): %v", err)
	}
	if pendingUnselected == nil {
		t.Fatalf("expected unselected task pending feedback to remain")
	}
}

func postReviewIntegrationWorkspace(t *testing.T) string {
	t.Helper()

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
	return tempDir
}

func postReviewIntegrationGraph(now time.Time, statuses map[string]plan.Status) plan.WorkGraph {
	items := make(map[string]plan.WorkItem, len(statuses))
	for id, status := range statuses {
		items[id] = plan.WorkItem{
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
	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items:         items,
	}
}

func postReviewIntegrationRun(results execution.ParentReviewTaskResults) execution.RunRecord {
	passed := false
	now := time.Date(2026, 1, 31, 8, 45, 0, 0, time.UTC)
	return execution.RunRecord{
		ID:                   "review-parent-1",
		TaskID:               "parent-1",
		Type:                 execution.RunTypeReview,
		StartedAt:            now,
		Status:               execution.RunStatusSuccess,
		ParentReviewPassed:   &passed,
		ParentReviewResults:  results,
		ParentReviewFeedback: "Fallback parent-review feedback.",
		Context: execution.ContextPack{
			SchemaVersion: execution.ContextPackSchemaVersion,
			Task: execution.TaskContext{
				ID:    "parent-1",
				Title: "Task parent-1",
			},
			ParentReview: &execution.ParentReviewContext{
				ParentTaskID:    "parent-1",
				ParentTaskTitle: "Task parent-1",
			},
		},
	}
}

func postReviewIntegrationModel(g plan.WorkGraph, run execution.RunRecord) Model {
	form := NewParentReviewForm(run, g)
	return Model{
		plan:                  g,
		viewMode:              ViewModeMain,
		planExists:            true,
		actionMode:            ActionModeParentReview,
		parentReviewForm:      &form,
		windowWidth:           120,
		windowHeight:          32,
		runData:               map[string]execution.RunRecord{},
		pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
	}
}

func executePostReviewResumeCmd(t *testing.T, cmd tea.Cmd) ExecuteActionComplete {
	t.Helper()

	if cmd == nil {
		t.Fatalf("expected resume command")
	}
	msg := cmd()
	if complete, ok := msg.(ExecuteActionComplete); ok {
		return complete
	}

	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete or tea.BatchMsg, got %T", msg)
	}
	if len(batch) == 0 || batch[0] == nil {
		t.Fatalf("expected resume command in batch")
	}

	msg = batch[0]()
	complete, ok := msg.(ExecuteActionComplete)
	if !ok {
		t.Fatalf("expected ExecuteActionComplete from resume command, got %T", msg)
	}
	return complete
}

func postReviewSaveResumeFixture(
	t *testing.T,
	baseDir string,
	taskID string,
	feedback string,
	now time.Time,
) {
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
		feedback,
	); err != nil {
		t.Fatalf("UpsertPendingParentReviewFeedback(%s): %v", taskID, err)
	}
}

func postReviewAssertLatestFeedback(t *testing.T, baseDir string, taskID string, expectedFeedback string) {
	t.Helper()

	latest, err := execution.GetLatestRun(baseDir, taskID)
	if err != nil {
		t.Fatalf("GetLatestRun(%s): %v", taskID, err)
	}
	if latest == nil {
		t.Fatalf("expected latest run for %s", taskID)
	}
	if latest.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected ParentReviewFeedback context for %s", taskID)
	}
	if got := latest.Context.ParentReviewFeedback.Feedback; got != expectedFeedback {
		t.Fatalf("feedback for %s = %q, want %q", taskID, got, expectedFeedback)
	}
}

func cloneRunRecordMap(in map[string]execution.RunRecord) map[string]execution.RunRecord {
	out := make(map[string]execution.RunRecord, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func clonePendingFeedbackMap(in map[string]execution.PendingParentReviewFeedback) map[string]execution.PendingParentReviewFeedback {
	out := make(map[string]execution.PendingParentReviewFeedback, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
