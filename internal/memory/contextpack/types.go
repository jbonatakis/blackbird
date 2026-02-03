package contextpack

import (
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

const SchemaVersion = 1

type RunTimeLookup func(taskID, runID string) (time.Time, bool)

type Budget struct {
	TotalTokens            int `json:"totalTokens"`
	DecisionsTokens        int `json:"decisionsTokens"`
	ConstraintsTokens      int `json:"constraintsTokens"`
	ImplementedTokens      int `json:"implementedTokens"`
	OpenThreadsTokens      int `json:"openThreadsTokens"`
	ArtifactPointersTokens int `json:"artifactPointersTokens"`
}

type Usage struct {
	TotalTokens            int `json:"totalTokens"`
	GoalTokens             int `json:"goalTokens"`
	InstructionTokens      int `json:"instructionTokens"`
	DecisionsTokens        int `json:"decisionsTokens"`
	ConstraintsTokens      int `json:"constraintsTokens"`
	ImplementedTokens      int `json:"implementedTokens"`
	OpenThreadsTokens      int `json:"openThreadsTokens"`
	ArtifactPointersTokens int `json:"artifactPointersTokens"`
}

type Section struct {
	Items  []string `json:"items,omitempty"`
	Tokens int      `json:"tokens"`
}

type ContextPack struct {
	SchemaVersion int      `json:"schemaVersion"`
	SessionID     string   `json:"session_id,omitempty"`
	SessionGoal   string   `json:"session_goal,omitempty"`
	Instructions  []string `json:"instructions,omitempty"`

	Decisions   Section   `json:"decisions,omitempty"`
	Constraints Section   `json:"constraints,omitempty"`
	Implemented Section   `json:"implemented,omitempty"`
	OpenThreads Section   `json:"open_threads,omitempty"`
	ArtifactIDs Section   `json:"artifact_ids,omitempty"`
	Budget      Budget    `json:"budget,omitempty"`
	Usage       Usage     `json:"usage,omitempty"`
	GeneratedAt time.Time `json:"generated_at,omitempty"`
}

type BuildOptions struct {
	SessionID     string
	SessionGoal   string
	Artifacts     []artifact.Artifact
	Budget        Budget
	Instructions  []string
	RunTimeLookup RunTimeLookup
	Now           time.Time
}
