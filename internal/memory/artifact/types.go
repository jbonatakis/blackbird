package artifact

const (
	SchemaVersion  = 1
	BuilderVersion = "deterministic-v1"
)

type ArtifactType string

const (
	ArtifactOutcome    ArtifactType = "outcome"
	ArtifactDecision   ArtifactType = "decision"
	ArtifactConstraint ArtifactType = "constraint"
	ArtifactOpenThread ArtifactType = "open_thread"
	ArtifactTranscript ArtifactType = "transcript"
)

type Artifact struct {
	SchemaVersion  int          `json:"schemaVersion"`
	ArtifactID     string       `json:"artifact_id"`
	SessionID      string       `json:"session_id,omitempty"`
	TaskID         string       `json:"task_id,omitempty"`
	RunID          string       `json:"run_id,omitempty"`
	Type           ArtifactType `json:"type"`
	Content        Content      `json:"content"`
	Provenance     []Provenance `json:"provenance,omitempty"`
	BuilderVersion string       `json:"builder_version"`
	Supersedes     []string     `json:"supersedes,omitempty"`
	SupersededBy   []string     `json:"superseded_by,omitempty"`
}

type Content struct {
	Text      string          `json:"text,omitempty"`
	Role      string          `json:"role,omitempty"`
	Status    string          `json:"status,omitempty"`
	Summary   []string        `json:"summary,omitempty"`
	Files     []string        `json:"files,omitempty"`
	Commands  []CommandResult `json:"commands,omitempty"`
	Errors    []string        `json:"errors,omitempty"`
	Rationale string          `json:"rationale,omitempty"`
	Scope     string          `json:"scope,omitempty"`
}

type CommandResult struct {
	Command  string `json:"command"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type Provenance struct {
	ItemIndex    int              `json:"item_index"`
	ItemType     string           `json:"item_type"`
	Role         string           `json:"role,omitempty"`
	ContentStart int              `json:"content_start,omitempty"`
	ContentEnd   int              `json:"content_end,omitempty"`
	Spans        []ProvenanceSpan `json:"spans,omitempty"`
}

type ProvenanceSpan struct {
	ContentStart int         `json:"content_start"`
	ContentEnd   int         `json:"content_end"`
	Trace        []TraceSpan `json:"trace,omitempty"`
}

type TraceSpan struct {
	RequestID  string `json:"request_id,omitempty"`
	EventIndex int    `json:"event_index"`
	EventType  string `json:"event_type"`
	Seq        int    `json:"seq,omitempty"`
	ByteStart  int    `json:"byte_start"`
	ByteEnd    int    `json:"byte_end"`
}

// Store is the durable artifact store schema.
type Store struct {
	SchemaVersion int        `json:"schemaVersion"`
	Artifacts     []Artifact `json:"artifacts"`
}
