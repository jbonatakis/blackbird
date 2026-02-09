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
}

func TestParseParentReviewResponseValidFail(t *testing.T) {
	output := `log line
{"passed":false,"resumeTaskIds":[" child-b ","child-a"],"feedbackForResume":"  Child outputs miss required validation paths.  "}`

	response, err := ParseParentReviewResponse(output, "parent-1", []string{"child-a", "child-b"})
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
