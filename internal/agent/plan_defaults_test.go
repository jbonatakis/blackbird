package agent

import (
	"strings"
	"testing"
)

func TestPlanDefaultsConstants(t *testing.T) {
	if MaxPlanQuestionRounds != 2 {
		t.Fatalf("MaxPlanQuestionRounds = %d, want 2", MaxPlanQuestionRounds)
	}
	if MaxPlanGenerateRevisions != 1 {
		t.Fatalf("MaxPlanGenerateRevisions = %d, want 1", MaxPlanGenerateRevisions)
	}
}

func TestDefaultPlanJSONSchemaIncludesCoreFields(t *testing.T) {
	schema := DefaultPlanJSONSchema()
	if schema == "" {
		t.Fatalf("DefaultPlanJSONSchema() returned empty schema")
	}
	if strings.TrimSpace(schema) != schema {
		t.Fatalf("DefaultPlanJSONSchema() should be trimmed")
	}
	for _, needle := range []string{
		`"schemaVersion"`,
		`"plan_refine"`,
		`"patch"`,
		`"questions"`,
		`"workGraph"`,
		`"workItem"`,
		`"patchOp"`,
		`"question"`,
	} {
		if !strings.Contains(schema, needle) {
			t.Fatalf("DefaultPlanJSONSchema() missing %q", needle)
		}
	}
}

func TestDefaultPlanSystemPromptIncludesDirectives(t *testing.T) {
	prompt := DefaultPlanSystemPrompt()
	if prompt == "" {
		t.Fatalf("DefaultPlanSystemPrompt() returned empty prompt")
	}
	if strings.TrimSpace(prompt) != prompt {
		t.Fatalf("DefaultPlanSystemPrompt() should be trimmed")
	}
	for _, needle := range []string{
		"Return exactly one JSON object",
		"How to use request inputs:",
		"Granularity guidance:",
		"Plan requirements:",
		"Plan quality heuristics:",
		"Task detail standards (especially for leaf tasks):",
		"Patch requirements:",
		"Questions:",
	} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("DefaultPlanSystemPrompt() missing %q", needle)
		}
	}
}
