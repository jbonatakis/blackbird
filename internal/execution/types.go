package execution

import (
	"encoding/json"
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

type RunType string

const (
	RunTypeExecute RunType = "execute"
	RunTypeReview  RunType = "review"
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
	SchemaVersion        int                          `json:"schemaVersion"`
	Task                 TaskContext                  `json:"task"`
	Dependencies         []DependencyContext          `json:"dependencies,omitempty"`
	ParentReview         *ParentReviewContext         `json:"parentReview,omitempty"`
	ParentReviewFeedback *ParentReviewFeedbackContext `json:"parentReviewFeedback,omitempty"`
	ProjectSnapshot      string                       `json:"projectSnapshot,omitempty"`
	Questions            []agent.Question             `json:"questions,omitempty"`
	Answers              []agent.Answer               `json:"answers,omitempty"`
	SystemPrompt         string                       `json:"systemPrompt,omitempty"`
}

type ParentReviewContext struct {
	ParentTaskID         string                     `json:"parentTaskId"`
	ParentTaskTitle      string                     `json:"parentTaskTitle,omitempty"`
	AcceptanceCriteria   []string                   `json:"acceptanceCriteria,omitempty"`
	ReviewerInstructions string                     `json:"reviewerInstructions,omitempty"`
	Children             []ParentReviewChildContext `json:"children,omitempty"`
}

type ParentReviewChildContext struct {
	ChildID          string   `json:"childId"`
	ChildTitle       string   `json:"childTitle,omitempty"`
	LatestRunID      string   `json:"latestRunId"`
	LatestRunSummary string   `json:"latestRunSummary"`
	ArtifactRefs     []string `json:"artifactRefs,omitempty"`
}

type ParentReviewFeedbackContext struct {
	ParentTaskID string `json:"parentTaskId,omitempty"`
	ReviewRunID  string `json:"reviewRunId,omitempty"`
	Feedback     string `json:"feedback,omitempty"`
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
	ID                              string         `json:"id"`
	TaskID                          string         `json:"taskId"`
	Type                            RunType        `json:"run_type,omitempty"`
	Provider                        string         `json:"provider,omitempty"`
	ProviderSessionRef              string         `json:"provider_session_ref,omitempty"`
	StartedAt                       time.Time      `json:"startedAt"`
	CompletedAt                     *time.Time     `json:"completedAt,omitempty"`
	Status                          RunStatus      `json:"status"`
	ExitCode                        *int           `json:"exitCode,omitempty"`
	Stdout                          string         `json:"stdout,omitempty"`
	Stderr                          string         `json:"stderr,omitempty"`
	Context                         ContextPack    `json:"context"`
	Error                           string         `json:"error,omitempty"`
	DecisionRequired                bool           `json:"decision_required,omitempty"`
	DecisionState                   DecisionState  `json:"decision_state,omitempty"`
	DecisionRequestedAt             *time.Time     `json:"decision_requested_at,omitempty"`
	DecisionResolvedAt              *time.Time     `json:"decision_resolved_at,omitempty"`
	DecisionFeedback                string         `json:"decision_feedback,omitempty"`
	ReviewSummary                   *ReviewSummary `json:"review_summary,omitempty"`
	ParentReviewPassed              *bool          `json:"parent_review_passed,omitempty"`
	ParentReviewResumeTaskIDs       []string       `json:"parent_review_resume_task_ids,omitempty"`
	ParentReviewFeedback            string         `json:"parent_review_feedback,omitempty"`
	ParentReviewCompletionSignature string         `json:"parent_review_completion_signature,omitempty"`
}

func (r RunRecord) MarshalJSON() ([]byte, error) {
	type runRecordAlias RunRecord
	alias := runRecordAlias(r)
	if alias.Type == "" {
		alias.Type = RunTypeExecute
	}
	return json.Marshal(alias)
}

func (r *RunRecord) UnmarshalJSON(data []byte) error {
	type runRecordAlias RunRecord
	var alias runRecordAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	if alias.Type == "" {
		alias.Type = RunTypeExecute
	}
	*r = RunRecord(alias)
	return nil
}
