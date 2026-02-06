package planquality

import (
	"reflect"
	"strings"
	"testing"
)

func TestHasBlocking(t *testing.T) {
	tests := []struct {
		name     string
		findings []PlanQualityFinding
		want     bool
	}{
		{
			name:     "empty",
			findings: nil,
			want:     false,
		},
		{
			name: "warnings_only",
			findings: []PlanQualityFinding{
				{
					Severity: SeverityWarning,
					Code:     RuleLeafDescriptionTooThin,
					TaskID:   "task-a",
					Field:    fieldDescription,
				},
			},
			want: false,
		},
		{
			name: "contains_blocking",
			findings: []PlanQualityFinding{
				{
					Severity: SeverityWarning,
					Code:     RuleLeafDescriptionTooThin,
					TaskID:   "task-a",
					Field:    fieldDescription,
				},
				{
					Severity: SeverityBlocking,
					Code:     RuleLeafPromptMissing,
					TaskID:   "task-b",
					Field:    fieldPrompt,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := HasBlocking(tt.findings); got != tt.want {
				t.Fatalf("HasBlocking() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeDeterministicCountsAndGrouping(t *testing.T) {
	findings := stableFindingFixture()
	reversed := reverseFindings(findings)

	got := Summarize(findings)
	gotReversed := Summarize(reversed)
	if !reflect.DeepEqual(got, gotReversed) {
		t.Fatalf("Summarize() changed with input order:\nforward=%#v\nreversed=%#v", got, gotReversed)
	}

	want := FindingsSummary{
		Total:    6,
		Blocking: 3,
		Warning:  3,
		Tasks: []TaskFindingSummary{
			{
				TaskID:   "task-a",
				Total:    4,
				Blocking: 2,
				Warning:  2,
				Fields: []FieldFindingSummary{
					{
						Field:    fieldDescription,
						Total:    2,
						Blocking: 1,
						Warning:  1,
						Findings: []PlanQualityFinding{
							{
								Severity:   SeverityBlocking,
								Code:       RuleLeafDescriptionMissingOrPlaceholder,
								TaskID:     "task-a",
								Field:      fieldDescription,
								Message:    "Leaf task description is missing or placeholder text.",
								Suggestion: "Describe the specific implementation intent, scope, and constraints for this task.",
							},
							{
								Severity:   SeverityWarning,
								Code:       RuleLeafDescriptionTooThin,
								TaskID:     "task-a",
								Field:      fieldDescription,
								Message:    "Leaf task description appears too thin.",
								Suggestion: "Expand the description with intent, in-scope work, and critical boundaries.",
							},
						},
					},
					{
						Field:    fieldAcceptanceCriteria,
						Total:    1,
						Blocking: 1,
						Warning:  0,
						Findings: []PlanQualityFinding{
							{
								Severity:   SeverityBlocking,
								Code:       RuleLeafAcceptanceCriteriaMissing,
								TaskID:     "task-a",
								Field:      fieldAcceptanceCriteria,
								Message:    "Leaf task has no acceptance criteria.",
								Suggestion: "Add objective acceptance criteria that can be verified during or after implementation.",
							},
						},
					},
					{
						Field:    fieldTask,
						Total:    1,
						Blocking: 0,
						Warning:  1,
						Findings: []PlanQualityFinding{
							{
								Severity:   SeverityWarning,
								Code:       RuleVagueLanguageDetected,
								TaskID:     "task-a",
								Field:      fieldTask,
								Message:    "Vague language detected without explicit expected behavior.",
								Suggestion: "Replace vague verbs with concrete outcomes and observable behavior.",
							},
						},
					},
				},
			},
			{
				TaskID:   "task-b",
				Total:    2,
				Blocking: 1,
				Warning:  1,
				Fields: []FieldFindingSummary{
					{
						Field:    fieldPrompt,
						Total:    2,
						Blocking: 1,
						Warning:  1,
						Findings: []PlanQualityFinding{
							{
								Severity:   SeverityBlocking,
								Code:       RuleLeafPromptMissing,
								TaskID:     "task-b",
								Field:      fieldPrompt,
								Message:    "Leaf task prompt is missing.",
								Suggestion: "Add an execution prompt with concrete implementation steps and expected completion signals.",
							},
							{
								Severity:   SeverityWarning,
								Code:       RuleLeafPromptMissingVerificationHint,
								TaskID:     "task-b",
								Field:      fieldPrompt,
								Message:    "Leaf task prompt does not mention verification or testing.",
								Suggestion: "Include how completion should be validated (for example, tests to run or checks to perform).",
							},
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Summarize() = %#v, want %#v", got, want)
	}
}

func TestBuildRefineRequestDeterministicOutput(t *testing.T) {
	findings := stableFindingFixture()
	reversed := reverseFindings(findings)

	got := BuildRefineRequest(findings)
	gotReversed := BuildRefineRequest(reversed)
	if got != gotReversed {
		t.Fatalf("BuildRefineRequest() changed with input order:\nforward:\n%s\n\nreversed:\n%s", got, gotReversed)
	}

	want := strings.Join([]string{
		"Refine the plan to resolve the findings below.",
		"",
		"Quality findings: blocking=3, warning=3, total=6.",
		"",
		"Hard constraints:",
		"1. Preserve every existing task ID exactly.",
		"2. Preserve parent/child hierarchy and dependency relationships.",
		"3. Do not add, remove, split, merge, or reparent tasks unless a finding explicitly requires structural change.",
		"4. Modify only the listed fields; keep all other plan content unchanged.",
		"",
		"Requested edits (grouped by task and field):",
		"- task `task-a`",
		"  - field `description`",
		"    - [blocking] leaf_description_missing_or_placeholder",
		"      issue: Leaf task description is missing or placeholder text.",
		"      requested change: Describe the specific implementation intent, scope, and constraints for this task.",
		"    - [warning] leaf_description_too_thin",
		"      issue: Leaf task description appears too thin.",
		"      requested change: Expand the description with intent, in-scope work, and critical boundaries.",
		"  - field `acceptanceCriteria`",
		"    - [blocking] leaf_acceptance_criteria_missing",
		"      issue: Leaf task has no acceptance criteria.",
		"      requested change: Add objective acceptance criteria that can be verified during or after implementation.",
		"  - field `task`",
		"    - [warning] vague_language_detected",
		"      issue: Vague language detected without explicit expected behavior.",
		"      requested change: Replace vague verbs with concrete outcomes and observable behavior.",
		"- task `task-b`",
		"  - field `prompt`",
		"    - [blocking] leaf_prompt_missing",
		"      issue: Leaf task prompt is missing.",
		"      requested change: Add an execution prompt with concrete implementation steps and expected completion signals.",
		"    - [warning] leaf_prompt_missing_verification_hint",
		"      issue: Leaf task prompt does not mention verification or testing.",
		"      requested change: Include how completion should be validated (for example, tests to run or checks to perform).",
		"",
		"Return only the refined plan in the required response schema.",
	}, "\n")

	if got != want {
		t.Fatalf("BuildRefineRequest() output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildRefineRequestEmptyFindings(t *testing.T) {
	got := BuildRefineRequest(nil)
	want := "No plan-quality findings were provided. Return the plan unchanged."
	if got != want {
		t.Fatalf("BuildRefineRequest(nil) = %q, want %q", got, want)
	}
}

func stableFindingFixture() []PlanQualityFinding {
	return []PlanQualityFinding{
		{
			Severity:   SeverityWarning,
			Code:       RuleLeafPromptMissingVerificationHint,
			TaskID:     "task-b",
			Field:      fieldPrompt,
			Message:    "Leaf task prompt does not mention verification or testing.",
			Suggestion: "Include how completion should be validated (for example, tests to run or checks to perform).",
		},
		{
			Severity:   SeverityBlocking,
			Code:       RuleLeafAcceptanceCriteriaMissing,
			TaskID:     "task-a",
			Field:      fieldAcceptanceCriteria,
			Message:    "Leaf task has no acceptance criteria.",
			Suggestion: "Add objective acceptance criteria that can be verified during or after implementation.",
		},
		{
			Severity:   SeverityWarning,
			Code:       RuleVagueLanguageDetected,
			TaskID:     "task-a",
			Field:      fieldTask,
			Message:    "Vague language detected without explicit expected behavior.",
			Suggestion: "Replace vague verbs with concrete outcomes and observable behavior.",
		},
		{
			Severity:   SeverityBlocking,
			Code:       RuleLeafDescriptionMissingOrPlaceholder,
			TaskID:     "task-a",
			Field:      fieldDescription,
			Message:    "Leaf task description is missing or placeholder text.",
			Suggestion: "Describe the specific implementation intent, scope, and constraints for this task.",
		},
		{
			Severity:   SeverityBlocking,
			Code:       RuleLeafPromptMissing,
			TaskID:     "task-b",
			Field:      fieldPrompt,
			Message:    "Leaf task prompt is missing.",
			Suggestion: "Add an execution prompt with concrete implementation steps and expected completion signals.",
		},
		{
			Severity:   SeverityWarning,
			Code:       RuleLeafDescriptionTooThin,
			TaskID:     "task-a",
			Field:      fieldDescription,
			Message:    "Leaf task description appears too thin.",
			Suggestion: "Expand the description with intent, in-scope work, and critical boundaries.",
		},
	}
}

func reverseFindings(findings []PlanQualityFinding) []PlanQualityFinding {
	out := append([]PlanQualityFinding(nil), findings...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
