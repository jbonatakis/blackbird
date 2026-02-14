package execution

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestGetLatestCompletedChildRunsDeterministicOrder(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)
	parentID := "parent-1"

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID:  childRunTestItem(parentID, plan.StatusTodo, now, "child-b", "child-a"),
			"child-a": childRunTestItem("child-a", plan.StatusDone, now),
			"child-b": childRunTestItem("child-b", plan.StatusDone, now),
		},
	}

	fixtures := []RunRecord{
		childRunTestRecord("child-a", "run-a-1", time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), RunStatusSuccess),
		childRunTestRecord("child-a", "run-a-2", time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), RunStatusSuccess),
		childRunTestRecord("child-b", "run-b-1", time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC), RunStatusSuccess),
	}
	for _, record := range fixtures {
		if err := SaveRun(baseDir, record); err != nil {
			t.Fatalf("SaveRun %s: %v", record.ID, err)
		}
	}

	contexts, err := GetLatestCompletedChildRuns(g, baseDir, parentID)
	if err != nil {
		t.Fatalf("GetLatestCompletedChildRuns: %v", err)
	}
	if len(contexts) != 2 {
		t.Fatalf("expected 2 child run contexts, got %d", len(contexts))
	}
	if contexts[0].ChildID != "child-b" || contexts[1].ChildID != "child-a" {
		t.Fatalf("unexpected child order: %#v", contexts)
	}
	if contexts[0].Run.ID != "run-b-1" {
		t.Fatalf("unexpected latest run for child-b: %#v", contexts[0].Run)
	}
	if contexts[1].Run.ID != "run-a-2" {
		t.Fatalf("unexpected latest run for child-a: %#v", contexts[1].Run)
	}
}

func TestGetLatestCompletedChildRunsMissingRunIncludesChildID(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)
	parentID := "parent-1"

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID:  childRunTestItem(parentID, plan.StatusTodo, now, "child-a", "child-b"),
			"child-a": childRunTestItem("child-a", plan.StatusDone, now),
			"child-b": childRunTestItem("child-b", plan.StatusDone, now),
		},
	}

	if err := SaveRun(baseDir, childRunTestRecord("child-a", "run-a-1", time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), RunStatusSuccess)); err != nil {
		t.Fatalf("SaveRun child-a: %v", err)
	}

	contexts, err := GetLatestCompletedChildRuns(g, baseDir, parentID)
	if err == nil {
		t.Fatalf("expected missing child run error")
	}
	if len(contexts) != 0 {
		t.Fatalf("expected no contexts on error, got %#v", contexts)
	}
	if !strings.Contains(err.Error(), "missing completed runs") {
		t.Fatalf("expected missing completed runs error, got %q", err)
	}
	if !strings.Contains(err.Error(), "child-b") {
		t.Fatalf("expected missing child ID in error, got %q", err)
	}
}

func TestGetLatestCompletedChildRunsMixedChildStatusAndNonTerminalRun(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)
	parentID := "parent-1"

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID:  childRunTestItem(parentID, plan.StatusTodo, now, "child-a", "child-b", "child-c"),
			"child-a": childRunTestItem("child-a", plan.StatusDone, now),
			"child-b": childRunTestItem("child-b", plan.StatusInProgress, now),
			"child-c": childRunTestItem("child-c", plan.StatusDone, now),
		},
	}

	fixtures := []RunRecord{
		childRunTestRecord("child-a", "run-a-1", time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), RunStatusSuccess),
		childRunTestRecord("child-c", "run-c-1", time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), RunStatusWaitingUser),
	}
	for _, record := range fixtures {
		if err := SaveRun(baseDir, record); err != nil {
			t.Fatalf("SaveRun %s: %v", record.ID, err)
		}
	}

	contexts, err := GetLatestCompletedChildRuns(g, baseDir, parentID)
	if err == nil {
		t.Fatalf("expected mixed child-state error")
	}
	if len(contexts) != 0 {
		t.Fatalf("expected no contexts on error, got %#v", contexts)
	}

	errText := err.Error()
	if !strings.Contains(errText, "children not done") || !strings.Contains(errText, "child-b(in_progress)") {
		t.Fatalf("expected non-done child details, got %q", errText)
	}
	if !strings.Contains(errText, "non-terminal") || !strings.Contains(errText, "child-c(waiting_user)") {
		t.Fatalf("expected non-terminal child details, got %q", errText)
	}
}

func childRunTestItem(id string, status plan.Status, now time.Time, childIDs ...string) plan.WorkItem {
	return plan.WorkItem{
		ID:        id,
		Title:     "Task " + id,
		Prompt:    "do it",
		ChildIDs:  append([]string{}, childIDs...),
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func childRunTestRecord(taskID, runID string, startedAt time.Time, status RunStatus) RunRecord {
	record := RunRecord{
		ID:        runID,
		TaskID:    taskID,
		Type:      RunTypeExecute,
		StartedAt: startedAt,
		Status:    status,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: taskID, Title: "Task"},
		},
	}
	if status != RunStatusRunning {
		completedAt := startedAt.Add(2 * time.Minute)
		record.CompletedAt = &completedAt
	}
	return record
}
