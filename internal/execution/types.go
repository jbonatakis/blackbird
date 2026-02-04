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

type DecisionState string

const (
	DecisionStatePending          DecisionState = "pending"
	DecisionStateApprovedContinue DecisionState = "approved_continue"
	DecisionStateApprovedQuit     DecisionState = "approved_quit"
	DecisionStateChangesRequested DecisionState = "changes_requested"
	DecisionStateRejected         DecisionState = "rejected"
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

type ReviewSummary struct {
	Files    []string        `json:"files,omitempty"`
	DiffStat string          `json:"diffstat,omitempty"`
	Snippets []ReviewSnippet `json:"snippets,omitempty"`
}

type ReviewSnippet struct {
	File    string `json:"file,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

type RunRecord struct {
	ID                  string         `json:"id"`
	TaskID              string         `json:"taskId"`
	Provider            string         `json:"provider,omitempty"`
	ProviderSessionRef  string         `json:"provider_session_ref,omitempty"`
	StartedAt           time.Time      `json:"startedAt"`
	CompletedAt         *time.Time     `json:"completedAt,omitempty"`
	Status              RunStatus      `json:"status"`
	ExitCode            *int           `json:"exitCode,omitempty"`
	Stdout              string         `json:"stdout,omitempty"`
	Stderr              string         `json:"stderr,omitempty"`
	Context             ContextPack    `json:"context"`
	Error               string         `json:"error,omitempty"`
	DecisionRequired    bool           `json:"decision_required,omitempty"`
	DecisionState       DecisionState  `json:"decision_state,omitempty"`
	DecisionRequestedAt *time.Time     `json:"decision_requested_at,omitempty"`
	DecisionResolvedAt  *time.Time     `json:"decision_resolved_at,omitempty"`
	DecisionFeedback    string         `json:"decision_feedback,omitempty"`
	ReviewSummary       *ReviewSummary `json:"review_summary,omitempty"`
}
