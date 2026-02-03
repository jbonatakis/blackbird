package canonical

const SchemaVersion = 1

type Log struct {
	SchemaVersion int      `json:"schemaVersion"`
	SessionID     string   `json:"session_id,omitempty"`
	TaskID        string   `json:"task_id,omitempty"`
	RunID         string   `json:"run_id,omitempty"`
	RequestIDs    []string `json:"request_ids,omitempty"`
	Metadata      Metadata `json:"metadata,omitempty"`
	Items         []Item   `json:"items,omitempty"`
}

type ItemType string

const (
	ItemMessage    ItemType = "message"
	ItemToolCall   ItemType = "tool_call"
	ItemToolResult ItemType = "tool_result"
)

type Item struct {
	Type       ItemType    `json:"type"`
	Message    *Message    `json:"message,omitempty"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

type Message struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	Provenance []ProvenanceSpan `json:"provenance,omitempty"`
}

type ToolCall struct {
	ID         string           `json:"id,omitempty"`
	Name       string           `json:"name,omitempty"`
	Arguments  string           `json:"arguments,omitempty"`
	Provenance []ProvenanceSpan `json:"provenance,omitempty"`
}

type ToolResult struct {
	ID         string           `json:"id,omitempty"`
	Result     string           `json:"result,omitempty"`
	Error      string           `json:"error,omitempty"`
	Provenance []ProvenanceSpan `json:"provenance,omitempty"`
}

type ProvenanceSpan struct {
	ContentStart int         `json:"content_start"`
	ContentEnd   int         `json:"content_end"`
	Trace        []TraceSpan `json:"trace"`
}

type TraceSpan struct {
	RequestID  string `json:"request_id,omitempty"`
	EventIndex int    `json:"event_index"`
	EventType  string `json:"event_type"`
	Seq        int    `json:"seq,omitempty"`
	ByteStart  int    `json:"byte_start"`
	ByteEnd    int    `json:"byte_end"`
}

type Metadata struct {
	RequestMethod string   `json:"request_method,omitempty"`
	RequestPath   string   `json:"request_path,omitempty"`
	Status        int      `json:"status,omitempty"`
	Model         string   `json:"model,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	MaxTokens     *int     `json:"max_tokens,omitempty"`
	Usage         Usage    `json:"usage,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}
