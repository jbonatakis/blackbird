package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/execution"
)

func TestFormatParentReviewRequiredLinesDeterministicOrdering(t *testing.T) {
	run := &execution.RunRecord{
		TaskID: "parent-checkout",
		ParentReviewResumeTaskIDs: []string{
			" child-b ",
			"child-a",
			"child-b",
			"",
		},
		ParentReviewFeedback: "  Child outputs miss required validation paths.\n\nRetry with coverage.  ",
	}

	got := formatParentReviewRequiredLines("ignored-task-id", run)
	want := []string{
		"running parent review for parent-checkout",
		"parent review failed for parent-checkout",
		"resume tasks: child-a, child-b",
		"feedback: Child outputs miss required validation paths. Retry with coverage.",
		"next step: blackbird resume child-a",
		"next step: blackbird resume child-b",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("formatParentReviewRequiredLines mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParentReviewFeedbackExcerptTruncatesLongFeedback(t *testing.T) {
	run := &execution.RunRecord{
		ParentReviewFeedback: strings.Repeat("x", parentReviewFeedbackExcerptMaxLen+20),
	}

	got := parentReviewFeedbackExcerpt(run)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated suffix, got %q", got)
	}
	if len(got) != parentReviewFeedbackExcerptMaxLen {
		t.Fatalf("feedback length = %d, want %d", len(got), parentReviewFeedbackExcerptMaxLen)
	}
}
