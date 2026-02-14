package execution

import (
	"testing"
	"time"
)

func TestParentReviewCompletionSignatureStableAcrossRepeatedCalls(t *testing.T) {
	parentTaskID := "parent-1"
	completions := []ChildCompletion{
		{ChildID: "child-b", CompletedAt: time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC)},
		{ChildID: "child-a", CompletedAt: time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)},
	}

	first, err := ParentReviewCompletionSignature(parentTaskID, completions)
	if err != nil {
		t.Fatalf("ParentReviewCompletionSignature first call: %v", err)
	}
	if first == "" {
		t.Fatalf("expected non-empty signature")
	}

	for i := 0; i < 5; i++ {
		got, err := ParentReviewCompletionSignature(parentTaskID, completions)
		if err != nil {
			t.Fatalf("ParentReviewCompletionSignature repeat %d: %v", i, err)
		}
		if got != first {
			t.Fatalf("repeat signature mismatch: got %q want %q", got, first)
		}
	}
}

func TestParentReviewCompletionSignatureStableAcrossMapAndSliceOrdering(t *testing.T) {
	parentTaskID := "parent-1"
	tA := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)
	tB := time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC)
	tC := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)

	completionMap := map[string]time.Time{
		"child-a": tA,
		"child-b": tB,
		"child-c": tC,
	}

	fromMap, err := ParentReviewCompletionSignatureFromMap(parentTaskID, completionMap)
	if err != nil {
		t.Fatalf("ParentReviewCompletionSignatureFromMap baseline: %v", err)
	}

	for i := 0; i < 25; i++ {
		got, err := ParentReviewCompletionSignatureFromMap(parentTaskID, completionMap)
		if err != nil {
			t.Fatalf("ParentReviewCompletionSignatureFromMap repeat %d: %v", i, err)
		}
		if got != fromMap {
			t.Fatalf("map-order signature mismatch on repeat %d: got %q want %q", i, got, fromMap)
		}
	}

	reorderedSlice := []ChildCompletion{
		{ChildID: "child-c", CompletedAt: tC},
		{ChildID: "child-a", CompletedAt: tA},
		{ChildID: "child-b", CompletedAt: tB},
	}
	fromReorderedSlice, err := ParentReviewCompletionSignature(parentTaskID, reorderedSlice)
	if err != nil {
		t.Fatalf("ParentReviewCompletionSignature reordered slice: %v", err)
	}
	if fromReorderedSlice != fromMap {
		t.Fatalf("slice-order signature mismatch: got %q want %q", fromReorderedSlice, fromMap)
	}

	anotherSliceOrder := []ChildCompletion{
		{ChildID: "child-b", CompletedAt: tB},
		{ChildID: "child-c", CompletedAt: tC},
		{ChildID: "child-a", CompletedAt: tA},
	}
	fromAnotherSliceOrder, err := ParentReviewCompletionSignature(parentTaskID, anotherSliceOrder)
	if err != nil {
		t.Fatalf("ParentReviewCompletionSignature another slice: %v", err)
	}
	if fromAnotherSliceOrder != fromMap {
		t.Fatalf("slice-order signature mismatch (second shuffle): got %q want %q", fromAnotherSliceOrder, fromMap)
	}
}

func TestParentReviewCompletionSignatureChangesWhenCompletionStateChanges(t *testing.T) {
	parentTaskID := "parent-1"

	baseState := map[string]time.Time{
		"child-a": time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC),
		"child-b": time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC),
	}
	baseline, err := ParentReviewCompletionSignatureFromMap(parentTaskID, baseState)
	if err != nil {
		t.Fatalf("ParentReviewCompletionSignatureFromMap baseline: %v", err)
	}

	tests := []struct {
		name        string
		completions map[string]time.Time
	}{
		{
			name: "timestamp change",
			completions: map[string]time.Time{
				"child-a": time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC),
				"child-b": time.Date(2026, 2, 9, 11, 0, 1, 0, time.UTC),
			},
		},
		{
			name: "child added",
			completions: map[string]time.Time{
				"child-a": time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC),
				"child-b": time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC),
				"child-c": time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "child removed",
			completions: map[string]time.Time{
				"child-a": time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParentReviewCompletionSignatureFromMap(parentTaskID, tc.completions)
			if err != nil {
				t.Fatalf("ParentReviewCompletionSignatureFromMap(%s): %v", tc.name, err)
			}
			if got == baseline {
				t.Fatalf("expected different signature for %s, got unchanged %q", tc.name, got)
			}
		})
	}
}

func TestShouldRunParentReviewForSignature(t *testing.T) {
	taskID := "parent-1"
	sigA := "sig-a"
	sigB := "sig-b"

	tests := []struct {
		name          string
		runs          []RunRecord
		signature     string
		wantShouldRun bool
	}{
		{
			name:          "no review runs requires review",
			signature:     sigA,
			wantShouldRun: true,
		},
		{
			name: "latest review matching signature skips review",
			runs: []RunRecord{
				parentReviewSignatureTestRunRecord(taskID, "review-1", RunTypeReview, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), sigA),
				parentReviewSignatureTestRunRecord(taskID, "exec-1", RunTypeExecute, time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), ""),
			},
			signature:     sigA,
			wantShouldRun: false,
		},
		{
			name: "latest review signature missing requires review",
			runs: []RunRecord{
				parentReviewSignatureTestRunRecord(taskID, "review-1", RunTypeReview, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), ""),
			},
			signature:     sigA,
			wantShouldRun: true,
		},
		{
			name: "latest review different signature requires review",
			runs: []RunRecord{
				parentReviewSignatureTestRunRecord(taskID, "review-1", RunTypeReview, time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC), sigA),
				parentReviewSignatureTestRunRecord(taskID, "review-2", RunTypeReview, time.Date(2026, 2, 9, 11, 0, 0, 0, time.UTC), sigB),
			},
			signature:     sigA,
			wantShouldRun: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			baseDir := t.TempDir()
			for _, run := range tc.runs {
				if err := SaveRun(baseDir, run); err != nil {
					t.Fatalf("SaveRun(%s): %v", run.ID, err)
				}
			}

			got, err := ShouldRunParentReviewForSignature(baseDir, taskID, tc.signature)
			if err != nil {
				t.Fatalf("ShouldRunParentReviewForSignature: %v", err)
			}
			if got != tc.wantShouldRun {
				t.Fatalf("ShouldRunParentReviewForSignature() = %v, want %v", got, tc.wantShouldRun)
			}
		})
	}
}

func parentReviewSignatureTestRunRecord(taskID, runID string, runType RunType, startedAt time.Time, signature string) RunRecord {
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
