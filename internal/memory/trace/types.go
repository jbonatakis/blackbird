package trace

import "time"

const SchemaVersion = 1

const (
	EventRequestStart  = "request.start"
	EventRequestBody   = "request.body.chunk"
	EventRequestEnd    = "request.end"
	EventResponseStart = "response.start"
	EventResponseBody  = "response.body.chunk"
	EventResponseEnd   = "response.end"
	EventError         = "error"
)

type Event struct {
	SchemaVersion int       `json:"schemaVersion"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`

	SessionID string `json:"session_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`

	Method  string              `json:"method,omitempty"`
	Path    string              `json:"path,omitempty"`
	Status  int                 `json:"status,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`

	Seq  int    `json:"seq,omitempty"`
	Body []byte `json:"body,omitempty"`

	BodyBytes  int64 `json:"body_bytes,omitempty"`
	DurationMs int64 `json:"duration_ms,omitempty"`

	Error     string `json:"error,omitempty"`
	ErrorKind string `json:"error_kind,omitempty"`
}
