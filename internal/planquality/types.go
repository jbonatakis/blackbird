package planquality

// Severity represents the impact of a plan quality finding.
type Severity string

const (
	SeverityBlocking Severity = "blocking"
	SeverityWarning  Severity = "warning"
)

// PlanQualityFinding captures one deterministic quality issue for a task field.
type PlanQualityFinding struct {
	Severity   Severity `json:"severity"`
	Code       string   `json:"code"`
	TaskID     string   `json:"taskId"`
	Field      string   `json:"field"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion"`
}

// FindingsSummary contains deterministic counts and grouped findings for rendering.
type FindingsSummary struct {
	Total    int                  `json:"total"`
	Blocking int                  `json:"blocking"`
	Warning  int                  `json:"warning"`
	Tasks    []TaskFindingSummary `json:"tasks,omitempty"`
}

// TaskFindingSummary groups findings for a single task in deterministic field order.
type TaskFindingSummary struct {
	TaskID   string                `json:"taskId"`
	Total    int                   `json:"total"`
	Blocking int                   `json:"blocking"`
	Warning  int                   `json:"warning"`
	Fields   []FieldFindingSummary `json:"fields,omitempty"`
}

// FieldFindingSummary groups findings for one task field in deterministic order.
type FieldFindingSummary struct {
	Field    string               `json:"field"`
	Total    int                  `json:"total"`
	Blocking int                  `json:"blocking"`
	Warning  int                  `json:"warning"`
	Findings []PlanQualityFinding `json:"findings,omitempty"`
}
