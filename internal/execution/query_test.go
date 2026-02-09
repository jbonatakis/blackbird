package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListRunsSortsByStartedAt(t *testing.T) {
	baseDir := t.TempDir()
	first := RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 10, 0, 0, 0, time.UTC),
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	second := RunRecord{
		ID:        "run-2",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 11, 0, 0, 0, time.UTC),
		Status:    RunStatusFailed,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	if err := SaveRun(baseDir, second); err != nil {
		t.Fatalf("SaveRun second: %v", err)
	}
	if err := SaveRun(baseDir, first); err != nil {
		t.Fatalf("SaveRun first: %v", err)
	}

	records, err := ListRuns(baseDir, "task-1")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "run-1" || records[1].ID != "run-2" {
		t.Fatalf("unexpected order: %#v", records)
	}
}

func TestListRunsEmptyWhenMissing(t *testing.T) {
	baseDir := t.TempDir()
	records, err := ListRuns(baseDir, "task-missing")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected empty records, got %v", records)
	}
}

func TestLoadRun(t *testing.T) {
	baseDir := t.TempDir()
	decisionRequested := time.Date(2026, 1, 28, 12, 5, 0, 0, time.UTC)
	decisionResolved := time.Date(2026, 1, 28, 12, 10, 0, 0, time.UTC)
	record := RunRecord{
		ID:                 "run-1",
		TaskID:             "task-1",
		ProviderSessionRef: "session-xyz",
		StartedAt:          time.Date(2026, 1, 28, 12, 0, 0, 0, time.UTC),
		Status:             RunStatusRunning,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
		DecisionRequired:    true,
		DecisionState:       DecisionStateApprovedContinue,
		DecisionRequestedAt: &decisionRequested,
		DecisionResolvedAt:  &decisionResolved,
		DecisionFeedback:    "Ship it",
		ReviewSummary: &ReviewSummary{
			Files: []string{"main.go"},
		},
	}
	if err := SaveRun(baseDir, record); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	loaded, err := LoadRun(baseDir, "task-1", "run-1")
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	if loaded.ID != record.ID || loaded.TaskID != record.TaskID {
		t.Fatalf("loaded mismatch: %#v", loaded)
	}
	if loaded.ProviderSessionRef != "session-xyz" {
		t.Fatalf("providerSessionRef mismatch: %#v", loaded.ProviderSessionRef)
	}
	if !loaded.DecisionRequired || loaded.DecisionState != DecisionStateApprovedContinue {
		t.Fatalf("decision gate mismatch: %#v", loaded)
	}
	if loaded.DecisionRequestedAt == nil || !loaded.DecisionRequestedAt.Equal(decisionRequested) {
		t.Fatalf("decisionRequestedAt mismatch: %#v", loaded.DecisionRequestedAt)
	}
	if loaded.DecisionResolvedAt == nil || !loaded.DecisionResolvedAt.Equal(decisionResolved) {
		t.Fatalf("decisionResolvedAt mismatch: %#v", loaded.DecisionResolvedAt)
	}
	if loaded.DecisionFeedback != "Ship it" {
		t.Fatalf("decisionFeedback mismatch: %#v", loaded.DecisionFeedback)
	}
	if loaded.ReviewSummary == nil || len(loaded.ReviewSummary.Files) != 1 {
		t.Fatalf("reviewSummary mismatch: %#v", loaded.ReviewSummary)
	}
}

func TestLoadRunMissingFile(t *testing.T) {
	baseDir := t.TempDir()
	_, err := LoadRun(baseDir, "task-1", "missing")
	if err == nil {
		t.Fatalf("expected error for missing run")
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, runsDirName)); statErr == nil {
		// ensure test doesn't accidentally create directories
		t.Fatalf("unexpected run directory created")
	}
}

func TestGetLatestRun(t *testing.T) {
	baseDir := t.TempDir()
	if latest, err := GetLatestRun(baseDir, "task-1"); err != nil || latest != nil {
		t.Fatalf("expected nil latest run, got %#v (err=%v)", latest, err)
	}

	first := RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 9, 0, 0, 0, time.UTC),
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}
	second := RunRecord{
		ID:        "run-2",
		TaskID:    "task-1",
		StartedAt: time.Date(2026, 1, 28, 10, 0, 0, 0, time.UTC),
		Status:    RunStatusFailed,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	if err := SaveRun(baseDir, first); err != nil {
		t.Fatalf("SaveRun first: %v", err)
	}
	if err := SaveRun(baseDir, second); err != nil {
		t.Fatalf("SaveRun second: %v", err)
	}

	latest, err := GetLatestRun(baseDir, "task-1")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil || latest.ID != "run-2" {
		t.Fatalf("unexpected latest: %#v", latest)
	}
}

func TestListRunsTieBreaksByRunID(t *testing.T) {
	baseDir := t.TempDir()
	started := time.Date(2026, 2, 9, 8, 0, 0, 0, time.UTC)

	runA := RunRecord{
		ID:        "run-a",
		TaskID:    "parent-1",
		Type:      RunTypeReview,
		StartedAt: started,
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "parent-1", Title: "Parent"},
		},
	}
	runB := RunRecord{
		ID:        "run-b",
		TaskID:    "parent-1",
		Type:      RunTypeReview,
		StartedAt: started,
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "parent-1", Title: "Parent"},
		},
	}

	if err := SaveRun(baseDir, runA); err != nil {
		t.Fatalf("SaveRun runA: %v", err)
	}
	if err := SaveRun(baseDir, runB); err != nil {
		t.Fatalf("SaveRun runB: %v", err)
	}

	runDir := filepath.Join(baseDir, runsDirName, "parent-1")
	if err := os.Rename(filepath.Join(runDir, "run-a.json"), filepath.Join(runDir, "z.json")); err != nil {
		t.Fatalf("rename run-a file: %v", err)
	}
	if err := os.Rename(filepath.Join(runDir, "run-b.json"), filepath.Join(runDir, "a.json")); err != nil {
		t.Fatalf("rename run-b file: %v", err)
	}

	records, err := ListRuns(baseDir, "parent-1")
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "run-a" || records[1].ID != "run-b" {
		t.Fatalf("unexpected deterministic tie order: %#v", records)
	}
}

func TestReviewRunHelpersExcludeExecuteRuns(t *testing.T) {
	baseDir := t.TempDir()
	taskID := "parent-1"

	fixtures := []RunRecord{
		testRunRecord(taskID, "exec-1", RunTypeExecute, time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC), ""),
		testRunRecord(taskID, "review-1", RunTypeReview, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), "sig-a"),
		testRunRecord(taskID, "exec-2", RunTypeExecute, time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), ""),
		testRunRecord(taskID, "review-2", RunTypeReview, time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC), "sig-b"),
	}
	for _, record := range fixtures {
		if err := SaveRun(baseDir, record); err != nil {
			t.Fatalf("SaveRun %s: %v", record.ID, err)
		}
	}

	reviews, err := ListReviewRuns(baseDir, taskID)
	if err != nil {
		t.Fatalf("ListReviewRuns: %v", err)
	}
	if len(reviews) != 2 {
		t.Fatalf("expected 2 review runs, got %d", len(reviews))
	}
	if reviews[0].ID != "review-1" || reviews[1].ID != "review-2" {
		t.Fatalf("unexpected review run order: %#v", reviews)
	}
	for _, review := range reviews {
		if review.Type != RunTypeReview {
			t.Fatalf("expected only review runs, got %q (%s)", review.Type, review.ID)
		}
	}
}

func TestGetLatestReviewRun(t *testing.T) {
	baseDir := t.TempDir()
	taskID := "parent-1"

	fixtures := []RunRecord{
		testRunRecord(taskID, "review-1", RunTypeReview, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), "sig-a"),
		testRunRecord(taskID, "review-2", RunTypeReview, time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), "sig-b"),
		testRunRecord(taskID, "exec-3", RunTypeExecute, time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC), ""),
	}
	for _, record := range fixtures {
		if err := SaveRun(baseDir, record); err != nil {
			t.Fatalf("SaveRun %s: %v", record.ID, err)
		}
	}

	latest, err := GetLatestReviewRun(baseDir, taskID)
	if err != nil {
		t.Fatalf("GetLatestReviewRun: %v", err)
	}
	if latest == nil {
		t.Fatalf("expected latest review run, got nil")
	}
	if latest.ID != "review-2" {
		t.Fatalf("latest review mismatch: got %s want %s", latest.ID, "review-2")
	}
	if latest.Type != RunTypeReview {
		t.Fatalf("expected review run type, got %q", latest.Type)
	}
}

func TestGetLatestReviewRunBySignature(t *testing.T) {
	baseDir := t.TempDir()
	taskID := "parent-1"

	fixtures := []RunRecord{
		testRunRecord(taskID, "review-1", RunTypeReview, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), "children:a,b"),
		testRunRecord(taskID, "review-2", RunTypeReview, time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), "children:a,b,c"),
		testRunRecord(taskID, "review-3", RunTypeReview, time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC), "children:a,b"),
		testRunRecord(taskID, "exec-1", RunTypeExecute, time.Date(2026, 2, 9, 13, 0, 0, 0, time.UTC), "children:a,b"),
	}
	for _, record := range fixtures {
		if err := SaveRun(baseDir, record); err != nil {
			t.Fatalf("SaveRun %s: %v", record.ID, err)
		}
	}

	latestMatching, err := GetLatestReviewRunBySignature(baseDir, taskID, "children:a,b")
	if err != nil {
		t.Fatalf("GetLatestReviewRunBySignature match: %v", err)
	}
	if latestMatching == nil || latestMatching.ID != "review-3" {
		t.Fatalf("latest signature match mismatch: %#v", latestMatching)
	}
	if latestMatching.Type != RunTypeReview {
		t.Fatalf("expected review run type for signature match, got %q", latestMatching.Type)
	}

	latestMissing, err := GetLatestReviewRunBySignature(baseDir, taskID, "children:missing")
	if err != nil {
		t.Fatalf("GetLatestReviewRunBySignature missing: %v", err)
	}
	if latestMissing != nil {
		t.Fatalf("expected nil for missing signature, got %#v", latestMissing)
	}
}

func testRunRecord(taskID, runID string, runType RunType, startedAt time.Time, signature string) RunRecord {
	return RunRecord{
		ID:        runID,
		TaskID:    taskID,
		Type:      runType,
		StartedAt: startedAt,
		Status:    RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: taskID, Title: "Task"},
		},
		ParentReviewCompletionSignature: signature,
	}
}
