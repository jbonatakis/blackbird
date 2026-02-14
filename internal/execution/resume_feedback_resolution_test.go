package execution

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestResolveResumeFeedbackSourceNone(t *testing.T) {
	baseDir := t.TempDir()

	resolved, err := ResolveResumeFeedbackSource(baseDir, "child-a", "   ", nil)
	if err != nil {
		t.Fatalf("ResolveResumeFeedbackSource: %v", err)
	}
	if resolved.Source != ResumeFeedbackSourceNone {
		t.Fatalf("source = %q, want %q", resolved.Source, ResumeFeedbackSourceNone)
	}
	if resolved.Feedback != "" {
		t.Fatalf("feedback = %q, want empty", resolved.Feedback)
	}
}

func TestResolveResumeFeedbackSourceExplicitPrecedesPending(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)

	if _, err := upsertPendingParentReviewFeedback(
		baseDir,
		"child-a",
		"parent-1",
		"review-1",
		"pending feedback",
		now,
	); err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	resolved, err := ResolveResumeFeedbackSource(baseDir, "child-a", "  explicit feedback  ", nil)
	if err != nil {
		t.Fatalf("ResolveResumeFeedbackSource: %v", err)
	}
	if resolved.Source != ResumeFeedbackSourceExplicit {
		t.Fatalf("source = %q, want %q", resolved.Source, ResumeFeedbackSourceExplicit)
	}
	if resolved.Feedback != "explicit feedback" {
		t.Fatalf("feedback = %q, want %q", resolved.Feedback, "explicit feedback")
	}
	if resolved.ParentTaskID != "" || resolved.ReviewRunID != "" {
		t.Fatalf("expected no pending metadata, got parent=%q review=%q", resolved.ParentTaskID, resolved.ReviewRunID)
	}
}

func TestResolveResumeFeedbackSourcePendingParentReview(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 9, 30, 0, 0, time.UTC)

	if _, err := upsertPendingParentReviewFeedback(
		baseDir,
		"child-a",
		"parent-9",
		"review-7",
		"  retry with stricter validation checks  ",
		now,
	); err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	resolved, err := ResolveResumeFeedbackSource(baseDir, "child-a", "", nil)
	if err != nil {
		t.Fatalf("ResolveResumeFeedbackSource: %v", err)
	}
	if resolved.Source != ResumeFeedbackSourcePendingParentReview {
		t.Fatalf("source = %q, want %q", resolved.Source, ResumeFeedbackSourcePendingParentReview)
	}
	if resolved.ParentTaskID != "parent-9" {
		t.Fatalf("parentTaskID = %q, want %q", resolved.ParentTaskID, "parent-9")
	}
	if resolved.ReviewRunID != "review-7" {
		t.Fatalf("reviewRunID = %q, want %q", resolved.ReviewRunID, "review-7")
	}
	if resolved.Feedback != "retry with stricter validation checks" {
		t.Fatalf("feedback = %q", resolved.Feedback)
	}
}

func TestResolveResumeFeedbackSourceRejectsMixedAnswers(t *testing.T) {
	baseDir := t.TempDir()
	answers := []agent.Answer{{
		ID:    "q1",
		Value: "answer",
	}}

	if _, err := ResolveResumeFeedbackSource(baseDir, "child-a", "feedback", answers); err == nil {
		t.Fatalf("expected error")
	} else if !strings.Contains(err.Error(), "cannot be combined with feedback-based resume") {
		t.Fatalf("error = %q", err.Error())
	}

	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)
	if _, err := upsertPendingParentReviewFeedback(
		baseDir,
		"child-a",
		"parent-a",
		"review-a",
		"fix and retry",
		now,
	); err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback: %v", err)
	}

	if _, err := ResolveResumeFeedbackSource(baseDir, "child-a", "", answers); err == nil {
		t.Fatalf("expected error")
	} else if !strings.Contains(err.Error(), "cannot be combined with pending parent-review feedback") {
		t.Fatalf("error = %q", err.Error())
	}
}
