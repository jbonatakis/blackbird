package provider

import (
	"net/http"

	"github.com/jbonatakis/blackbird/internal/config"
)

const (
	ProviderCodex  = "codex"
	ProviderClaude = "claude"

	HeaderBlackbirdSessionID = "X-Blackbird-Session-Id"
	HeaderBlackbirdTaskID    = "X-Blackbird-Task-Id"
	HeaderBlackbirdRunID     = "X-Blackbird-Run-Id"
)

type Upstream string

const (
	UpstreamUnknown Upstream = ""
	UpstreamAPI     Upstream = "api"
	UpstreamChatGPT Upstream = "chatgpt"
)

type Route struct {
	Upstream Upstream
	Path     string
}

type RequestIDs struct {
	SessionID string
	TaskID    string
	RunID     string
}

type Adapter interface {
	ProviderID() string
	Enabled(memory config.ResolvedMemory) bool
	BaseURLPrefix() string
	BaseHeaders(ids RequestIDs) http.Header
	RequestIDs(headers http.Header) RequestIDs
	Route(path string, headers http.Header) Route
}
