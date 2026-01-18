package plan

import "time"

const (
	DefaultPlanFilename = "blackbird.plan.json"
	SchemaVersion       = 1
)

type WorkGraph struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Items         map[string]WorkItem `json:"items"`
}

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusDone       Status = "done"
	StatusSkipped    Status = "skipped"
)

type WorkItem struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Description        string            `json:"description"`
	AcceptanceCriteria []string          `json:"acceptanceCriteria"`
	Prompt             string            `json:"prompt"`
	ParentID           *string           `json:"parentId"`
	ChildIDs           []string          `json:"childIds"`
	Deps               []string          `json:"deps"`
	Status             Status            `json:"status"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	Notes              *string           `json:"notes,omitempty"`
	DepRationale       map[string]string `json:"depRationale,omitempty"`
}

func NewEmptyWorkGraph() WorkGraph {
	now := time.Now().UTC()
	_ = now // reserved for future root timestamps if we add them.

	return WorkGraph{
		SchemaVersion: SchemaVersion,
		Items:         map[string]WorkItem{},
	}
}
