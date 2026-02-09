package planquality

import (
	"fmt"
	"sort"
	"strings"
)

var fieldSortOrder = map[string]int{
	fieldDescription:        0,
	fieldAcceptanceCriteria: 1,
	fieldPrompt:             2,
	fieldTask:               3,
}

// HasBlocking reports whether findings contain at least one blocking issue.
func HasBlocking(findings []PlanQualityFinding) bool {
	for _, finding := range findings {
		if finding.Severity == SeverityBlocking {
			return true
		}
	}
	return false
}

// Summarize builds deterministic counts and task/field groupings for findings.
func Summarize(findings []PlanQualityFinding) FindingsSummary {
	ordered := sortFindings(findings)
	summary := FindingsSummary{
		Tasks: make([]TaskFindingSummary, 0),
	}

	taskIndex := -1
	fieldIndex := -1
	lastTaskID := ""
	lastField := ""

	for _, finding := range ordered {
		summary.Total++
		incrementSeverityCounts(&summary.Blocking, &summary.Warning, finding.Severity)

		if taskIndex < 0 || finding.TaskID != lastTaskID {
			summary.Tasks = append(summary.Tasks, TaskFindingSummary{
				TaskID: finding.TaskID,
				Fields: make([]FieldFindingSummary, 0),
			})
			taskIndex = len(summary.Tasks) - 1
			fieldIndex = -1
			lastTaskID = finding.TaskID
			lastField = ""
		}

		task := &summary.Tasks[taskIndex]
		task.Total++
		incrementSeverityCounts(&task.Blocking, &task.Warning, finding.Severity)

		if fieldIndex < 0 || finding.Field != lastField {
			task.Fields = append(task.Fields, FieldFindingSummary{
				Field:    finding.Field,
				Findings: make([]PlanQualityFinding, 0),
			})
			fieldIndex = len(task.Fields) - 1
			lastField = finding.Field
		}

		field := &task.Fields[fieldIndex]
		field.Total++
		incrementSeverityCounts(&field.Blocking, &field.Warning, finding.Severity)
		field.Findings = append(field.Findings, finding)
	}

	return summary
}

// BuildRefineRequest renders a deterministic, concise refine request from findings.
func BuildRefineRequest(findings []PlanQualityFinding) string {
	summary := Summarize(findings)
	if summary.Total == 0 {
		return "No plan-quality findings were provided. Return the plan unchanged."
	}

	var b strings.Builder
	b.WriteString("Refine the plan to resolve the findings below.\n\n")
	fmt.Fprintf(&b, "Quality findings: blocking=%d, warning=%d, total=%d.\n\n", summary.Blocking, summary.Warning, summary.Total)
	b.WriteString("Hard constraints:\n")
	b.WriteString("1. Preserve every existing task ID exactly.\n")
	b.WriteString("2. Preserve parent/child hierarchy and dependency relationships.\n")
	b.WriteString("3. Do not add, remove, split, merge, or reparent tasks unless a finding explicitly requires structural change.\n")
	b.WriteString("4. Modify only the listed fields; keep all other plan content unchanged.\n\n")
	b.WriteString("Requested edits (grouped by task and field):\n")

	for _, task := range summary.Tasks {
		fmt.Fprintf(&b, "- task `%s`\n", task.TaskID)
		for _, field := range task.Fields {
			fmt.Fprintf(&b, "  - field `%s`\n", field.Field)
			for _, finding := range field.Findings {
				fmt.Fprintf(&b, "    - [%s] %s\n", finding.Severity, finding.Code)
				fmt.Fprintf(&b, "      issue: %s\n", finding.Message)
				fmt.Fprintf(&b, "      requested change: %s\n", finding.Suggestion)
			}
		}
	}

	b.WriteString("\nReturn only the refined plan in the required response schema.")
	return b.String()
}

func incrementSeverityCounts(blocking, warning *int, severity Severity) {
	switch severity {
	case SeverityBlocking:
		*blocking = *blocking + 1
	case SeverityWarning:
		*warning = *warning + 1
	}
}

func sortFindings(findings []PlanQualityFinding) []PlanQualityFinding {
	ordered := append([]PlanQualityFinding(nil), findings...)
	sort.Slice(ordered, func(i, j int) bool {
		return findingLess(ordered[i], ordered[j])
	})
	return ordered
}

func findingLess(a, b PlanQualityFinding) bool {
	if a.TaskID != b.TaskID {
		return a.TaskID < b.TaskID
	}

	aFieldRank := fieldRank(a.Field)
	bFieldRank := fieldRank(b.Field)
	if aFieldRank != bFieldRank {
		return aFieldRank < bFieldRank
	}
	if a.Field != b.Field {
		return a.Field < b.Field
	}

	aSeverityRank := severityRank(a.Severity)
	bSeverityRank := severityRank(b.Severity)
	if aSeverityRank != bSeverityRank {
		return aSeverityRank < bSeverityRank
	}
	if a.Severity != b.Severity {
		return a.Severity < b.Severity
	}

	if a.Code != b.Code {
		return a.Code < b.Code
	}
	if a.Message != b.Message {
		return a.Message < b.Message
	}
	return a.Suggestion < b.Suggestion
}

func fieldRank(field string) int {
	if rank, ok := fieldSortOrder[field]; ok {
		return rank
	}
	return len(fieldSortOrder) + 1
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityBlocking:
		return 0
	case SeverityWarning:
		return 1
	default:
		return 2
	}
}
