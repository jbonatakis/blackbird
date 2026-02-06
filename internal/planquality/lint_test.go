package planquality

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestLintRuleTriggerAndNonTriggerFixtures(t *testing.T) {
	identity := func(it plan.WorkItem) plan.WorkItem { return it }

	fixtures := []struct {
		code       string
		severity   Severity
		field      string
		trigger    func(plan.WorkItem) plan.WorkItem
		nonTrigger func(plan.WorkItem) plan.WorkItem
	}{
		{
			code:     RuleLeafDescriptionMissingOrPlaceholder,
			severity: SeverityBlocking,
			field:    fieldDescription,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.Description = "TODO"
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleLeafAcceptanceCriteriaMissing,
			severity: SeverityBlocking,
			field:    fieldAcceptanceCriteria,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.AcceptanceCriteria = []string{}
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleLeafAcceptanceCriteriaNonVerifiable,
			severity: SeverityBlocking,
			field:    fieldAcceptanceCriteria,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.AcceptanceCriteria = []string{
					"Works well in practice.",
					"Is robust and easy to maintain.",
				}
				return it
			},
			nonTrigger: func(it plan.WorkItem) plan.WorkItem {
				it.AcceptanceCriteria = []string{
					"Works well in practice.",
					"`go test ./internal/planquality/...` passes.",
				}
				return it
			},
		},
		{
			code:     RuleLeafPromptMissing,
			severity: SeverityBlocking,
			field:    fieldPrompt,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.Prompt = ""
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleLeafPromptNotActionable,
			severity: SeverityBlocking,
			field:    fieldPrompt,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.Prompt = "Please improve this."
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleLeafDescriptionTooThin,
			severity: SeverityWarning,
			field:    fieldDescription,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.Description = "Add request tracing."
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleLeafAcceptanceCriteriaLowCount,
			severity: SeverityWarning,
			field:    fieldAcceptanceCriteria,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.AcceptanceCriteria = []string{
					"`go test ./internal/planquality/...` passes.",
				}
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleLeafPromptMissingVerificationHint,
			severity: SeverityWarning,
			field:    fieldPrompt,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.Prompt = "Implement lint rule evaluation in `internal/planquality/lint.go` and wire it into the plan quality package."
				return it
			},
			nonTrigger: identity,
		},
		{
			code:     RuleVagueLanguageDetected,
			severity: SeverityWarning,
			field:    fieldTask,
			trigger: func(it plan.WorkItem) plan.WorkItem {
				it.Description = "Improve plan quality behavior across generation and review paths without explicit outcomes."
				return it
			},
			nonTrigger: identity,
		},
	}

	for _, fixture := range fixtures {
		fixture := fixture

		t.Run(fixture.code+"_trigger", func(t *testing.T) {
			leaf := fixture.trigger(validLeafTask("leaf"))
			findings := Lint(graphWithItems(leaf))

			finding, ok := findingByCode(findings, fixture.code)
			if !ok {
				t.Fatalf("expected code %q in findings, got %v", fixture.code, findingCodes(findings))
			}
			if finding.Severity != fixture.severity {
				t.Fatalf("severity for %q = %q, want %q", fixture.code, finding.Severity, fixture.severity)
			}
			if finding.Field != fixture.field {
				t.Fatalf("field for %q = %q, want %q", fixture.code, finding.Field, fixture.field)
			}
			if strings.TrimSpace(finding.Message) == "" {
				t.Fatalf("message for %q is empty", fixture.code)
			}
			if strings.TrimSpace(finding.Suggestion) == "" {
				t.Fatalf("suggestion for %q is empty", fixture.code)
			}
		})

		t.Run(fixture.code+"_non_trigger", func(t *testing.T) {
			leaf := fixture.nonTrigger(validLeafTask("leaf"))
			findings := Lint(graphWithItems(leaf))
			if hasFindingCode(findings, fixture.code) {
				t.Fatalf("did not expect code %q, got %v", fixture.code, findingCodes(findings))
			}
		})
	}
}

func TestLeafAcceptanceCriteriaNonVerifiableRequiresPresentAndFullyVagueCriteria(t *testing.T) {
	missing := validLeafTask("leaf-missing")
	missing.AcceptanceCriteria = []string{}
	missingFindings := Lint(graphWithItems(missing))
	if hasFindingCode(missingFindings, RuleLeafAcceptanceCriteriaNonVerifiable) {
		t.Fatalf("unexpected %q when criteria are missing", RuleLeafAcceptanceCriteriaNonVerifiable)
	}
	if !hasFindingCode(missingFindings, RuleLeafAcceptanceCriteriaMissing) {
		t.Fatalf("expected %q when criteria are missing", RuleLeafAcceptanceCriteriaMissing)
	}

	mixed := validLeafTask("leaf-mixed")
	mixed.AcceptanceCriteria = []string{
		"Works well for users.",
		"`go test ./internal/planquality/...` passes.",
	}
	mixedFindings := Lint(graphWithItems(mixed))
	if hasFindingCode(mixedFindings, RuleLeafAcceptanceCriteriaNonVerifiable) {
		t.Fatalf("unexpected %q for mixed criteria", RuleLeafAcceptanceCriteriaNonVerifiable)
	}

	allVague := validLeafTask("leaf-vague")
	allVague.AcceptanceCriteria = []string{
		"Works well for users.",
		"Is robust and maintainable.",
	}
	allVagueFindings := Lint(graphWithItems(allVague))
	if !hasFindingCode(allVagueFindings, RuleLeafAcceptanceCriteriaNonVerifiable) {
		t.Fatalf("expected %q for entirely vague criteria", RuleLeafAcceptanceCriteriaNonVerifiable)
	}
}

func TestWordCountHeuristicsNeverProduceBlockingFindingsByThemselves(t *testing.T) {
	leaf := validLeafTask("leaf-thin")
	leaf.Description = "Add retries."

	findings := Lint(graphWithItems(leaf))
	if !hasFindingCode(findings, RuleLeafDescriptionTooThin) {
		t.Fatalf("expected %q warning, got %v", RuleLeafDescriptionTooThin, findingCodes(findings))
	}
	for _, finding := range findings {
		if finding.Severity == SeverityBlocking {
			t.Fatalf("unexpected blocking finding from word-count-only case: %#v", finding)
		}
	}
}

func TestLintSkipsNonLeafTasks(t *testing.T) {
	parent := validLeafTask("parent")
	parent.ChildIDs = []string{"leaf"}
	parent.Description = ""
	parent.AcceptanceCriteria = []string{}
	parent.Prompt = ""

	leaf := validLeafTask("leaf")

	findings := Lint(graphWithItems(parent, leaf))
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findingCodes(findings))
	}
}

func TestLintDeterministicOrderByTaskIDAndRuleOrder(t *testing.T) {
	leafA := validLeafTask("leaf-a")
	leafA.Description = "TODO"
	leafA.AcceptanceCriteria = []string{}
	leafA.Prompt = ""

	leafB := validLeafTask("leaf-b")
	leafB.Prompt = "Please improve this."

	findings := Lint(graphWithItems(leafB, leafA))
	got := findingTaskCodePairs(findings)
	want := []string{
		"leaf-a:" + RuleLeafDescriptionMissingOrPlaceholder,
		"leaf-a:" + RuleLeafAcceptanceCriteriaMissing,
		"leaf-a:" + RuleLeafPromptMissing,
		"leaf-b:" + RuleLeafPromptNotActionable,
		"leaf-b:" + RuleLeafPromptMissingVerificationHint,
		"leaf-b:" + RuleVagueLanguageDetected,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("finding order = %v, want %v", got, want)
	}
}

func TestLintDeterministicOrderAndCoverageAcrossAllRuleCodes(t *testing.T) {
	leafA := validLeafTask("leaf-a")
	leafA.Description = "TODO"
	leafA.AcceptanceCriteria = []string{}
	leafA.Prompt = ""

	leafB := validLeafTask("leaf-b")
	leafB.Description = "Implement deterministic lint output ordering for blocking prompt and criteria checks."
	leafB.AcceptanceCriteria = []string{
		"Works well in practice.",
		"Is robust and maintainable.",
	}
	leafB.Prompt = "Please improve this."

	leafC := validLeafTask("leaf-c")
	leafC.Description = "Add tracing."
	leafC.AcceptanceCriteria = []string{
		"`go test ./internal/planquality/...` passes.",
	}
	leafC.Prompt = "Implement request tracing and verify with `go test ./internal/planquality/...`."

	graph := graphWithItems(leafC, leafA, leafB)
	wantOrder := []string{
		"leaf-a:" + RuleLeafDescriptionMissingOrPlaceholder,
		"leaf-a:" + RuleLeafAcceptanceCriteriaMissing,
		"leaf-a:" + RuleLeafPromptMissing,
		"leaf-b:" + RuleLeafAcceptanceCriteriaNonVerifiable,
		"leaf-b:" + RuleLeafPromptNotActionable,
		"leaf-b:" + RuleLeafPromptMissingVerificationHint,
		"leaf-b:" + RuleVagueLanguageDetected,
		"leaf-c:" + RuleLeafDescriptionTooThin,
		"leaf-c:" + RuleLeafAcceptanceCriteriaLowCount,
	}
	expectedCodes := []string{
		RuleLeafDescriptionMissingOrPlaceholder,
		RuleLeafAcceptanceCriteriaMissing,
		RuleLeafAcceptanceCriteriaNonVerifiable,
		RuleLeafPromptMissing,
		RuleLeafPromptNotActionable,
		RuleLeafDescriptionTooThin,
		RuleLeafAcceptanceCriteriaLowCount,
		RuleLeafPromptMissingVerificationHint,
		RuleVagueLanguageDetected,
	}

	findings := Lint(graph)
	gotOrder := findingTaskCodePairs(findings)
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("finding order = %v, want %v", gotOrder, wantOrder)
	}

	seenCodes := make(map[string]struct{}, len(findings))
	for _, finding := range findings {
		seenCodes[finding.Code] = struct{}{}
	}
	for _, code := range expectedCodes {
		if _, ok := seenCodes[code]; !ok {
			t.Fatalf("missing expected rule code %q in findings: %v", code, findingCodes(findings))
		}
	}
	if len(seenCodes) != len(expectedCodes) {
		t.Fatalf("unexpected code coverage size = %d, want %d (codes=%v)", len(seenCodes), len(expectedCodes), findingCodes(findings))
	}

	for i := 0; i < 50; i++ {
		repeatOrder := findingTaskCodePairs(Lint(graph))
		if !reflect.DeepEqual(repeatOrder, wantOrder) {
			t.Fatalf("finding order changed on run %d: got %v, want %v", i+1, repeatOrder, wantOrder)
		}
	}
}

func validLeafTask(id string) plan.WorkItem {
	return plan.WorkItem{
		ID:          id,
		Title:       "Task " + id,
		Description: "Implement deterministic lint evaluation for leaf tasks with explicit quality checks and execution constraints.",
		AcceptanceCriteria: []string{
			"`Lint` returns blocking findings for placeholder descriptions.",
			"`go test ./internal/planquality/...` passes with rule coverage updates.",
		},
		Prompt:   "Implement lint rule evaluation in `internal/planquality/lint.go`, update tests in `internal/planquality/lint_test.go`, and run `go test ./internal/planquality/...` to verify completion.",
		ChildIDs: []string{},
	}
}

func graphWithItems(items ...plan.WorkItem) plan.WorkGraph {
	graphItems := make(map[string]plan.WorkItem, len(items))
	for _, item := range items {
		graphItems[item.ID] = item
	}
	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items:         graphItems,
	}
}

func findingByCode(findings []PlanQualityFinding, code string) (PlanQualityFinding, bool) {
	for _, finding := range findings {
		if finding.Code == code {
			return finding, true
		}
	}
	return PlanQualityFinding{}, false
}

func hasFindingCode(findings []PlanQualityFinding, code string) bool {
	_, ok := findingByCode(findings, code)
	return ok
}

func findingCodes(findings []PlanQualityFinding) []string {
	codes := make([]string, 0, len(findings))
	for _, finding := range findings {
		codes = append(codes, finding.Code)
	}
	return codes
}

func findingTaskCodePairs(findings []PlanQualityFinding) []string {
	pairs := make([]string, 0, len(findings))
	for _, finding := range findings {
		pairs = append(pairs, finding.TaskID+":"+finding.Code)
	}
	return pairs
}
