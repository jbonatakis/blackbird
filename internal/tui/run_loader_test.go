package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestLoadRunDataMissingDir(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID: "task-1",
				},
			},
		},
	}

	msg := model.LoadRunData()()
	loaded, ok := msg.(RunDataLoaded)
	if !ok {
		t.Fatalf("expected RunDataLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadRunData error: %v", loaded.Err)
	}
	if len(loaded.Data) != 0 {
		t.Fatalf("expected empty run data, got %d", len(loaded.Data))
	}
	if len(loaded.PendingParentFeedback) != 0 {
		t.Fatalf("expected empty pending feedback data, got %d", len(loaded.PendingParentFeedback))
	}
}

func TestLoadRunDataReturnsLatestByTask(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)
	first := execution.RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: now.Add(-2 * time.Minute),
		Status:    execution.RunStatusFailed,
	}
	second := execution.RunRecord{
		ID:        "run-2",
		TaskID:    "task-1",
		StartedAt: now.Add(-1 * time.Minute),
		Status:    execution.RunStatusRunning,
	}
	other := execution.RunRecord{
		ID:        "run-3",
		TaskID:    "task-2",
		StartedAt: now.Add(-30 * time.Second),
		Status:    execution.RunStatusSuccess,
	}

	if err := execution.SaveRun(tempDir, first); err != nil {
		t.Fatalf("SaveRun first: %v", err)
	}
	if err := execution.SaveRun(tempDir, second); err != nil {
		t.Fatalf("SaveRun second: %v", err)
	}
	if err := execution.SaveRun(tempDir, other); err != nil {
		t.Fatalf("SaveRun other: %v", err)
	}
	pending, err := execution.UpsertPendingParentReviewFeedback(
		tempDir,
		"task-2",
		"parent-1",
		"review-1",
		"retry task-2 with coverage updates",
	)
	if err != nil {
		t.Fatalf("UpsertPendingParentReviewFeedback: %v", err)
	}

	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID: "task-1",
				},
				"task-2": {
					ID: "task-2",
				},
			},
		},
	}

	msg := model.LoadRunData()()
	loaded, ok := msg.(RunDataLoaded)
	if !ok {
		t.Fatalf("expected RunDataLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadRunData error: %v", loaded.Err)
	}
	if len(loaded.Data) != 2 {
		t.Fatalf("expected 2 run records, got %d", len(loaded.Data))
	}
	if got := loaded.Data["task-1"].ID; got != "run-2" {
		t.Fatalf("expected latest run for task-1, got %q", got)
	}
	if got := loaded.Data["task-2"].ID; got != "run-3" {
		t.Fatalf("expected run-3 for task-2, got %q", got)
	}
	if len(loaded.PendingParentFeedback) != 1 {
		t.Fatalf("expected 1 pending feedback entry, got %d", len(loaded.PendingParentFeedback))
	}
	if got := loaded.PendingParentFeedback["task-2"]; got != pending {
		t.Fatalf("unexpected pending feedback payload: got %#v want %#v", got, pending)
	}
}

func TestLoadRunDataReturnsErrorWhenPendingFeedbackDecodeFails(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 2, 9, 2, 0, 0, 0, time.UTC)
	run := execution.RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: now,
		Status:    execution.RunStatusSuccess,
	}
	if err := execution.SaveRun(tempDir, run); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	pendingPath := filepath.Join(tempDir, ".blackbird", "parent-review-feedback", "task-1.json")
	if err := os.MkdirAll(filepath.Dir(pendingPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(pendingPath, []byte("{invalid json}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	model := Model{
		plan: plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				"task-1": {
					ID: "task-1",
				},
			},
		},
	}

	msg := model.LoadRunData()()
	loaded, ok := msg.(RunDataLoaded)
	if !ok {
		t.Fatalf("expected RunDataLoaded, got %T", msg)
	}
	if loaded.Err == nil {
		t.Fatalf("expected LoadRunData error")
	}
	if got := loaded.Data["task-1"].ID; got != "run-1" {
		t.Fatalf("expected partial run data with latest run preserved, got %q", got)
	}
	if len(loaded.PendingParentFeedback) != 0 {
		t.Fatalf("expected no pending feedback entries on decode error, got %d", len(loaded.PendingParentFeedback))
	}
}
