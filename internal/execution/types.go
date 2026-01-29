package execution

import (
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

const ContextPackSchemaVersion = 1

type RunStatus string

const (
	RunStatusRunning     RunStatus = "running"
	RunStatusSuccess     RunStatus = "success"
	RunStatusFailed      RunStatus = "failed"
	RunStatusWaitingUser RunStatus = "waiting_user"
)

type TaskContext struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	AcceptanceCriteria []string `json:"acceptanceCriteria,omitempty"`
	Prompt             string   `json:"prompt,omitempty"`
}

type DependencyContext struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Status    string   `json:"status,omitempty"`
	Artifacts []string `json:"artifacts,omitempty"`
}

type ContextPack struct {
	SchemaVersion   int                 `json:"schemaVersion"`
	Task            TaskContext         `json:"task"`
	Dependencies    []DependencyContext `json:"dependencies,omitempty"`
	ProjectSnapshot string              `json:"projectSnapshot,omitempty"`
	Questions       []agent.Question    `json:"questions,omitempty"`
	Answers         []agent.Answer      `json:"answers,omitempty"`
	SystemPrompt    string              `json:"systemPrompt,omitempty"`
}

type RunRecord struct {
	ID          string      `json:"id"`
	TaskID      string      `json:"taskId"`
	Provider    string      `json:"provider,omitempty"`
	StartedAt   time.Time   `json:"startedAt"`
	CompletedAt *time.Time  `json:"completedAt,omitempty"`
	Status      RunStatus   `json:"status"`
	ExitCode    *int        `json:"exitCode,omitempty"`
	Stdout      string      `json:"stdout,omitempty"`
	Stderr      string      `json:"stderr,omitempty"`
	Context     ContextPack `json:"context"`
	Error       string      `json:"error,omitempty"`
}
