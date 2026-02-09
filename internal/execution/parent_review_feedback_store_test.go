package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPendingParentReviewFeedbackRoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)

	written, err := upsertPendingParentReviewFeedback(
		baseDir,
		"child-1",
		"parent-1",
		"review-run-1",
		"Address acceptance criteria gap in parsing.",
		now,
	)
	if err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	if written.ParentTaskID != "parent-1" {
		t.Fatalf("parent task id mismatch: got %q", written.ParentTaskID)
	}
	if written.ReviewRunID != "review-run-1" {
		t.Fatalf("review run id mismatch: got %q", written.ReviewRunID)
	}
	if written.Feedback != "Address acceptance criteria gap in parsing." {
		t.Fatalf("feedback mismatch: got %q", written.Feedback)
	}
	if !written.CreatedAt.Equal(now) {
		t.Fatalf("createdAt mismatch: got %s want %s", written.CreatedAt, now)
	}
	if !written.UpdatedAt.Equal(now) {
		t.Fatalf("updatedAt mismatch: got %s want %s", written.UpdatedAt, now)
	}

	path := filepath.Join(baseDir, parentReviewFeedbackDirName, "child-1.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected feedback file at %s: %v", path, err)
	}

	loaded, err := LoadPendingParentReviewFeedback(baseDir, "child-1")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected loaded feedback record, got nil")
	}

	if loaded.ParentTaskID != "parent-1" || loaded.ReviewRunID != "review-run-1" {
		t.Fatalf("loaded IDs mismatch: %#v", loaded)
	}
	if loaded.Feedback != "Address acceptance criteria gap in parsing." {
		t.Fatalf("loaded feedback mismatch: %q", loaded.Feedback)
	}
	if !loaded.CreatedAt.Equal(now) || !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("loaded timestamps mismatch: %#v", loaded)
	}
}

func TestPendingParentReviewFeedbackOverwritePreservesCreatedAt(t *testing.T) {
	baseDir := t.TempDir()
	first := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	second := first.Add(2 * time.Minute)

	if _, err := upsertPendingParentReviewFeedback(baseDir, "child-1", "parent-a", "review-a", "first feedback", first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if _, err := upsertPendingParentReviewFeedback(baseDir, "child-1", "parent-b", "review-b", "second feedback", second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	loaded, err := LoadPendingParentReviewFeedback(baseDir, "child-1")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected loaded feedback record, got nil")
	}

	if loaded.ParentTaskID != "parent-b" {
		t.Fatalf("parent task id mismatch: got %q want %q", loaded.ParentTaskID, "parent-b")
	}
	if loaded.ReviewRunID != "review-b" {
		t.Fatalf("review run id mismatch: got %q want %q", loaded.ReviewRunID, "review-b")
	}
	if loaded.Feedback != "second feedback" {
		t.Fatalf("feedback mismatch: got %q want %q", loaded.Feedback, "second feedback")
	}
	if !loaded.CreatedAt.Equal(first) {
		t.Fatalf("createdAt mismatch: got %s want %s", loaded.CreatedAt, first)
	}
	if !loaded.UpdatedAt.Equal(second) {
		t.Fatalf("updatedAt mismatch: got %s want %s", loaded.UpdatedAt, second)
	}
}

func TestClearPendingParentReviewFeedback(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	if _, err := upsertPendingParentReviewFeedback(baseDir, "child-1", "parent-1", "review-1", "feedback", now); err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	if err := ClearPendingParentReviewFeedback(baseDir, "child-1"); err != nil {
		t.Fatalf("ClearPendingParentReviewFeedback: %v", err)
	}

	loaded, err := LoadPendingParentReviewFeedback(baseDir, "child-1")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback after clear: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil record after clear, got %#v", loaded)
	}

	if err := ClearPendingParentReviewFeedback(baseDir, "child-1"); err != nil {
		t.Fatalf("ClearPendingParentReviewFeedback should be idempotent: %v", err)
	}
}

func TestLoadPendingParentReviewFeedbackMissingFile(t *testing.T) {
	baseDir := t.TempDir()
	loaded, err := LoadPendingParentReviewFeedback(baseDir, "child-missing")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil for missing feedback file, got %#v", loaded)
	}

	storeDir := filepath.Join(baseDir, parentReviewFeedbackDirName)
	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("expected store dir to remain absent, err=%v", err)
	}
}

func TestPendingParentReviewFeedbackRejectsTraversalTaskID(t *testing.T) {
	baseDir := t.TempDir()

	if _, err := UpsertPendingParentReviewFeedback(baseDir, "../escape", "parent-1", "review-1", "feedback"); err == nil {
		t.Fatalf("expected traversal task id to fail")
	}
	if _, err := LoadPendingParentReviewFeedback(baseDir, "../escape"); err == nil {
		t.Fatalf("expected traversal task id load to fail")
	}
	if err := ClearPendingParentReviewFeedback(baseDir, "../escape"); err == nil {
		t.Fatalf("expected traversal task id clear to fail")
	}
}
