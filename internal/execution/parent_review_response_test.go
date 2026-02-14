package execution

import (
	"strings"
	"testing"
)

func TestParseParentReviewResponseValidPass(t *testing.T) {
	output := `review complete
{"passed":true}`

	response, err := ParseParentReviewResponse(output, "parent-1", []string{"child-a", "child-b"})
	if err != nil {
		t.Fatalf("ParseParentReviewResponse: %v", err)
	}
	if !response.Passed {
		t.Fatalf("response.Passed = false, want true")
	}
	if len(response.ResumeTaskIDs) != 0 {
		t.Fatalf("response.ResumeTaskIDs = %#v, want empty", response.ResumeTaskIDs)
	}
	if response.FeedbackForResume != "" {
		t.Fatalf("response.FeedbackForResume = %q, want empty", response.FeedbackForResume)
	}
	if len(response.TaskResults) != 2 {
		t.Fatalf("len(response.TaskResults) = %d, want 2", len(response.TaskResults))
	}
	for _, taskID := range []string{"child-a", "child-b"} {
		result, ok := response.TaskResults[taskID]
		if !ok {
			t.Fatalf("missing task result for %s", taskID)
		}
		if result.TaskID != taskID {
			t.Fatalf("result.TaskID (%s) = %q, want %q", taskID, result.TaskID, taskID)
		}
		if result.Status != ParentReviewTaskStatusPassed {
			t.Fatalf("result.Status (%s) = %q, want %q", taskID, result.Status, ParentReviewTaskStatusPassed)
		}
		if result.Feedback != "" {
			t.Fatalf("result.Feedback (%s) = %q, want empty", taskID, result.Feedback)
		}
	}
}

func TestParseParentReviewResponseValidFail(t *testing.T) {
	output := `log line
{"passed":false,"resumeTaskIds":[" child-b ","child-a"],"feedbackForResume":"  Child outputs miss required validation paths.  "}`

	response, err := ParseParentReviewResponse(output, "parent-1", []string{"child-a", "child-b", "child-c"})
	if err != nil {
		t.Fatalf("ParseParentReviewResponse: %v", err)
	}
	if response.Passed {
		t.Fatalf("response.Passed = true, want false")
	}
	if len(response.ResumeTaskIDs) != 2 || response.ResumeTaskIDs[0] != "child-a" || response.ResumeTaskIDs[1] != "child-b" {
		t.Fatalf("response.ResumeTaskIDs = %#v, want [child-a child-b]", response.ResumeTaskIDs)
	}
	if response.FeedbackForResume != "Child outputs miss required validation paths." {
		t.Fatalf("response.FeedbackForResume = %q", response.FeedbackForResume)
	}
	if len(response.TaskResults) != 3 {
		t.Fatalf("len(response.TaskResults) = %d, want 3", len(response.TaskResults))
	}
	for _, taskID := range []string{"child-a", "child-b"} {
		result, ok := response.TaskResults[taskID]
		if !ok {
			t.Fatalf("missing task result for %s", taskID)
		}
		if result.Status != ParentReviewTaskStatusFailed {
			t.Fatalf("result.Status (%s) = %q, want %q", taskID, result.Status, ParentReviewTaskStatusFailed)
		}
		if result.Feedback != "Child outputs miss required validation paths." {
			t.Fatalf("result.Feedback (%s) = %q", taskID, result.Feedback)
		}
	}
	childC, ok := response.TaskResults["child-c"]
	if !ok {
		t.Fatalf("missing task result for child-c")
	}
	if childC.Status != ParentReviewTaskStatusPassed {
		t.Fatalf("child-c status = %q, want %q", childC.Status, ParentReviewTaskStatusPassed)
	}
	if childC.Feedback != "" {
		t.Fatalf("child-c feedback = %q, want empty", childC.Feedback)
	}
}

func TestParseParentReviewResponsePartialTaskResultsFallsBackSafely(t *testing.T) {
	output := `{"passed":false,"resumeTaskIds":["child-b"],"feedbackForResume":"  Global fallback feedback.  ","reviewResults":[{"taskId":"child-a","status":"passed","feedback":"ignore this"},{"taskId":"child-b"},{"taskId":"child-c","status":"failed"}]}`

	response, err := ParseParentReviewResponse(output, "parent-1", []string{"child-a", "child-b", "child-c"})
	if err != nil {
		t.Fatalf("ParseParentReviewResponse: %v", err)
	}

	if len(response.TaskResults) != 3 {
		t.Fatalf("len(response.TaskResults) = %d, want 3", len(response.TaskResults))
	}

	if got := response.TaskResults["child-a"]; got.Status != ParentReviewTaskStatusPassed || got.Feedback != "" {
		t.Fatalf("child-a result = %#v, want passed with empty feedback", got)
	}
	if got := response.TaskResults["child-b"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "Global fallback feedback." {
		t.Fatalf("child-b result = %#v, want failed with global fallback feedback", got)
	}
	if got := response.TaskResults["child-c"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "Global fallback feedback." {
		t.Fatalf("child-c result = %#v, want failed with global fallback feedback", got)
	}

	if len(response.ResumeTaskIDs) != 1 || response.ResumeTaskIDs[0] != "child-b" {
		t.Fatalf("response.ResumeTaskIDs = %#v, want [child-b]", response.ResumeTaskIDs)
	}
}

func TestParseParentReviewResponseMalformedJSON(t *testing.T) {
	output := `{"passed":false,"resumeTaskIds":["child-a"],"feedbackForResume":"fix"`

	_, err := ParseParentReviewResponse(output, "parent-1", []string{"child-a"})
	if err == nil {
		t.Fatalf("expected parse error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "no valid JSON object found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseParentReviewResponseUnknownChildID(t *testing.T) {
	output := `{"passed":false,"resumeTaskIds":["child-c"],"feedbackForResume":"fix child-c"}`

	_, err := ParseParentReviewResponse(output, "parent-1", []string{"child-a", "child-b"})
	if err == nil {
		t.Fatalf("expected validation error for unknown child id")
	}
	if !strings.Contains(err.Error(), "not a child of this parent") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseParentReviewResponseMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantErr string
	}{
		{
			name:    "missing passed",
			output:  `{"resumeTaskIds":["child-a"],"feedbackForResume":"fix"}`,
			wantErr: `missing required field "passed"`,
		},
		{
			name:    "missing resumeTaskIds when failed",
			output:  `{"passed":false,"feedbackForResume":"fix"}`,
			wantErr: `resumeTaskIds required when passed=false`,
		},
		{
			name:    "missing feedback when failed",
			output:  `{"passed":false,"resumeTaskIds":["child-a"]}`,
			wantErr: `feedbackForResume required when passed=false`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseParentReviewResponse(tc.output, "parent-1", []string{"child-a", "child-b"})
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
