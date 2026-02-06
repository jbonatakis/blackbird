package planquality

import (
	"strings"
	"unicode"

	"github.com/jbonatakis/blackbird/internal/plan"
)

const (
	RuleLeafDescriptionMissingOrPlaceholder = "leaf_description_missing_or_placeholder"
	RuleLeafAcceptanceCriteriaMissing       = "leaf_acceptance_criteria_missing"
	RuleLeafAcceptanceCriteriaNonVerifiable = "leaf_acceptance_criteria_non_verifiable"
	RuleLeafPromptMissing                   = "leaf_prompt_missing"
	RuleLeafPromptNotActionable             = "leaf_prompt_not_actionable"

	RuleLeafDescriptionTooThin            = "leaf_description_too_thin"
	RuleLeafAcceptanceCriteriaLowCount    = "leaf_acceptance_criteria_low_count"
	RuleLeafPromptMissingVerificationHint = "leaf_prompt_missing_verification_hint"
	RuleVagueLanguageDetected             = "vague_language_detected"
)

const (
	fieldTask               = "task"
	fieldDescription        = "description"
	fieldAcceptanceCriteria = "acceptanceCriteria"
	fieldPrompt             = "prompt"

	descriptionTooThinWordThreshold      = 8
	recommendedAcceptanceCriteriaMinimum = 2
)

var (
	descriptionPlaceholderPhrases = []string{
		"todo",
		"tbd",
		"to be determined",
		"placeholder",
		"implement feature",
		"fill in later",
		"coming soon",
	}

	promptPlaceholderPhrases = []string{
		"todo",
		"tbd",
		"do this",
		"work on this",
		"implement this task",
		"handle this",
		"as needed",
	}

	implementationDirectionPhrases = []string{
		"implement",
		"add",
		"create",
		"update",
		"modify",
		"refactor",
		"remove",
		"rename",
		"wire",
		"introduce",
		"migrate",
		"write",
		"build",
		"document",
	}

	completionSignalPhrases = []string{
		"done when",
		"acceptance criteria",
		"verify",
		"validated",
		"validation",
		"tests pass",
		"passes",
		"assert",
		"confirm",
	}

	verificationHintPhrases = []string{
		"go test",
		"test",
		"tests",
		"verify",
		"validation",
		"assert",
		"check",
		"checks",
		"pass",
		"passes",
		"unit test",
		"integration test",
		"smoke test",
	}

	verifiableCriteriaPhrases = []string{
		"go test",
		"unit test",
		"integration test",
		"assert",
		"returns",
		"contains",
		"creates",
		"updates",
		"writes",
		"logs",
		"emits",
		"rejects",
		"allows",
		"status code",
		"exit code",
		"response",
		"output",
		"error",
		"fails",
		"passes",
		"file",
		"path",
		"command",
		"json",
		"http",
	}

	vagueLanguagePhrases = []string{
		"improve",
		"fix",
		"handle",
		"optimize",
		"enhance",
		"clean up",
		"cleanup",
		"better",
		"robust",
	}

	expectedBehaviorSignalPhrases = []string{
		"returns",
		"contains",
		"creates",
		"updates",
		"writes",
		"logs",
		"emits",
		"rejects",
		"allows",
		"prevents",
		"when",
		"if",
		"verify",
		"validation",
		"assert",
		"tests",
		"go test",
		"status code",
		"exit code",
		"output",
		"error",
	}
)

// Lint applies deterministic leaf-task quality checks and returns findings in stable order.
func Lint(g plan.WorkGraph) []PlanQualityFinding {
	findings := make([]PlanQualityFinding, 0)
	for _, taskID := range LeafTaskIDs(g) {
		it, ok := g.Items[taskID]
		if !ok {
			continue
		}
		findings = append(findings, lintLeafTask(taskID, it)...)
	}
	return findings
}

func lintLeafTask(taskID string, it plan.WorkItem) []PlanQualityFinding {
	findings := make([]PlanQualityFinding, 0, 8)

	descriptionMissingOrPlaceholder := strings.TrimSpace(it.Description) == "" ||
		ContainsAnyNormalizedPhrase(it.Description, descriptionPlaceholderPhrases)
	criteria := nonEmptyCriteria(it.AcceptanceCriteria)
	criteriaCount := len(criteria)
	promptMissing := strings.TrimSpace(it.Prompt) == ""
	promptNotActionable := !promptMissing && isPromptNotActionable(it.Prompt)

	// Blocking rule order is fixed to keep output deterministic and predictable.
	if descriptionMissingOrPlaceholder {
		findings = append(findings, newFinding(
			SeverityBlocking,
			RuleLeafDescriptionMissingOrPlaceholder,
			taskID,
			fieldDescription,
			"Leaf task description is missing or placeholder text.",
			"Describe the specific implementation intent, scope, and constraints for this task.",
		))
	}

	if criteriaCount == 0 {
		findings = append(findings, newFinding(
			SeverityBlocking,
			RuleLeafAcceptanceCriteriaMissing,
			taskID,
			fieldAcceptanceCriteria,
			"Leaf task has no acceptance criteria.",
			"Add objective acceptance criteria that can be verified during or after implementation.",
		))
	}

	if criteriaCount > 0 && allCriteriaNonVerifiable(criteria) {
		findings = append(findings, newFinding(
			SeverityBlocking,
			RuleLeafAcceptanceCriteriaNonVerifiable,
			taskID,
			fieldAcceptanceCriteria,
			"Leaf task acceptance criteria are present but non-verifiable.",
			"Rewrite criteria with observable outcomes (tests, outputs, files, or explicit behaviors).",
		))
	}

	if promptMissing {
		findings = append(findings, newFinding(
			SeverityBlocking,
			RuleLeafPromptMissing,
			taskID,
			fieldPrompt,
			"Leaf task prompt is missing.",
			"Add an execution prompt with concrete implementation steps and expected completion signals.",
		))
	}

	if promptNotActionable {
		findings = append(findings, newFinding(
			SeverityBlocking,
			RuleLeafPromptNotActionable,
			taskID,
			fieldPrompt,
			"Leaf task prompt is not actionable.",
			"Rewrite the prompt with concrete implementation actions and clear done conditions.",
		))
	}

	if !descriptionMissingOrPlaceholder && wordCount(it.Description) < descriptionTooThinWordThreshold {
		findings = append(findings, newFinding(
			SeverityWarning,
			RuleLeafDescriptionTooThin,
			taskID,
			fieldDescription,
			"Leaf task description appears too thin.",
			"Expand the description with intent, in-scope work, and critical boundaries.",
		))
	}

	if criteriaCount > 0 && criteriaCount < recommendedAcceptanceCriteriaMinimum {
		findings = append(findings, newFinding(
			SeverityWarning,
			RuleLeafAcceptanceCriteriaLowCount,
			taskID,
			fieldAcceptanceCriteria,
			"Leaf task has fewer than the recommended number of acceptance criteria.",
			"Add another objective criterion to reduce ambiguity during execution.",
		))
	}

	if !promptMissing && !hasVerificationHint(it.Prompt) {
		findings = append(findings, newFinding(
			SeverityWarning,
			RuleLeafPromptMissingVerificationHint,
			taskID,
			fieldPrompt,
			"Leaf task prompt does not mention verification or testing.",
			"Include how completion should be validated (for example, tests to run or checks to perform).",
		))
	}

	if hasVagueLanguage(it) {
		findings = append(findings, newFinding(
			SeverityWarning,
			RuleVagueLanguageDetected,
			taskID,
			fieldTask,
			"Vague language detected without explicit expected behavior.",
			"Replace vague verbs with concrete outcomes and observable behavior.",
		))
	}

	return findings
}

func newFinding(
	severity Severity,
	code string,
	taskID string,
	field string,
	message string,
	suggestion string,
) PlanQualityFinding {
	return PlanQualityFinding{
		Severity:   severity,
		Code:       code,
		TaskID:     taskID,
		Field:      field,
		Message:    message,
		Suggestion: suggestion,
	}
}

func nonEmptyCriteria(criteria []string) []string {
	out := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		if strings.TrimSpace(criterion) == "" {
			continue
		}
		out = append(out, criterion)
	}
	return out
}

func allCriteriaNonVerifiable(criteria []string) bool {
	for _, criterion := range criteria {
		if criterionLooksVerifiable(criterion) {
			return false
		}
	}
	return true
}

func criterionLooksVerifiable(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	if strings.Contains(value, "`") || containsDigit(value) {
		return true
	}
	return ContainsAnyNormalizedPhrase(value, verifiableCriteriaPhrases)
}

func isPromptNotActionable(prompt string) bool {
	if ContainsAnyNormalizedPhrase(prompt, promptPlaceholderPhrases) {
		return true
	}
	hasDirection := ContainsAnyNormalizedPhrase(prompt, implementationDirectionPhrases)
	hasCompletionSignal := ContainsAnyNormalizedPhrase(prompt, completionSignalPhrases)
	return !hasDirection && !hasCompletionSignal
}

func hasVerificationHint(prompt string) bool {
	return ContainsAnyNormalizedPhrase(prompt, verificationHintPhrases)
}

func hasVagueLanguage(it plan.WorkItem) bool {
	texts := make([]string, 0, len(it.AcceptanceCriteria)+2)
	texts = append(texts, it.Description, it.Prompt)
	texts = append(texts, it.AcceptanceCriteria...)

	for _, text := range texts {
		if !ContainsAnyNormalizedPhrase(text, vagueLanguagePhrases) {
			continue
		}
		if hasExpectedBehaviorSignal(text) {
			continue
		}
		return true
	}

	return false
}

func hasExpectedBehaviorSignal(value string) bool {
	if strings.Contains(value, "`") || containsDigit(value) {
		return true
	}
	return ContainsAnyNormalizedPhrase(value, expectedBehaviorSignalPhrases)
}

func wordCount(value string) int {
	normalized := NormalizeText(value)
	if normalized == "" {
		return 0
	}
	return len(strings.Fields(normalized))
}

func containsDigit(value string) bool {
	for _, r := range value {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
