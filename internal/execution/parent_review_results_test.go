package execution

import (
	"reflect"
	"testing"
)

func TestNormalizeParentReviewTaskResultsIncludesAllChildren(t *testing.T) {
	results := NormalizeParentReviewTaskResults(
		[]string{"child-c", "child-a", "child-b"},
		false,
		[]string{"child-b"},
		"retry child-b",
		ParentReviewTaskResults{
			"child-a": {
				TaskID: "child-a",
				Status: ParentReviewTaskStatusPassed,
			},
			"child-c": {
				TaskID: "child-c",
				Status: ParentReviewTaskStatusFailed,
			},
		},
	)

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	if got := results["child-a"]; got.Status != ParentReviewTaskStatusPassed || got.Feedback != "" {
		t.Fatalf("child-a result = %#v, want passed with empty feedback", got)
	}
	if got := results["child-b"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "retry child-b" {
		t.Fatalf("child-b result = %#v, want failed with fallback feedback", got)
	}
	if got := results["child-c"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "retry child-b" {
		t.Fatalf("child-c result = %#v, want failed with fallback feedback", got)
	}
}

func TestParentReviewTaskResultsHelpersPreferStructuredPayload(t *testing.T) {
	record := RunRecord{
		ParentReviewResumeTaskIDs: []string{"legacy-child"},
		ParentReviewFeedback:      "legacy feedback",
		ParentReviewResults: ParentReviewTaskResults{
			"child-b": {
				TaskID:   "child-b",
				Status:   ParentReviewTaskStatusFailed,
				Feedback: "targeted child-b feedback",
			},
			"child-a": {
				TaskID: "child-a",
				Status: ParentReviewTaskStatusPassed,
			},
		},
	}

	failed := ParentReviewFailedTaskIDs(record)
	if want := []string{"child-b"}; !reflect.DeepEqual(failed, want) {
		t.Fatalf("ParentReviewFailedTaskIDs() = %#v, want %#v", failed, want)
	}
	if got := ParentReviewFeedbackForTask(record, "child-b"); got != "targeted child-b feedback" {
		t.Fatalf("ParentReviewFeedbackForTask(child-b) = %q, want %q", got, "targeted child-b feedback")
	}
	if got := ParentReviewFeedbackForTask(record, "legacy-child"); got != "" {
		t.Fatalf("ParentReviewFeedbackForTask(legacy-child) = %q, want empty when structured payload is present", got)
	}
	if got := ParentReviewPrimaryFeedback(record); got != "legacy feedback" {
		t.Fatalf("ParentReviewPrimaryFeedback() = %q, want %q", got, "legacy feedback")
	}
}

func TestParentReviewTaskResultsHelpersFallbackToLegacyFields(t *testing.T) {
	record := RunRecord{
		ParentReviewResumeTaskIDs: []string{" child-b ", "child-a", "child-b"},
		ParentReviewFeedback:      "legacy feedback",
	}

	failed := ParentReviewFailedTaskIDs(record)
	if want := []string{"child-a", "child-b"}; !reflect.DeepEqual(failed, want) {
		t.Fatalf("ParentReviewFailedTaskIDs() = %#v, want %#v", failed, want)
	}
	if got := ParentReviewFeedbackForTask(record, "child-a"); got != "legacy feedback" {
		t.Fatalf("ParentReviewFeedbackForTask(child-a) = %q, want %q", got, "legacy feedback")
	}
	if got := ParentReviewPrimaryFeedback(record); got != "legacy feedback" {
		t.Fatalf("ParentReviewPrimaryFeedback() = %q, want %q", got, "legacy feedback")
	}
}
