package provider

import (
	"net/http"

	"github.com/jbonatakis/blackbird/internal/config"
)

type ClaudeAdapter struct{}

func (ClaudeAdapter) ProviderID() string {
	return ProviderClaude
}

func (ClaudeAdapter) Enabled(memory config.ResolvedMemory) bool {
	return false
}

func (ClaudeAdapter) BaseURLPrefix() string {
	return ""
}

func (ClaudeAdapter) BaseHeaders(ids RequestIDs) http.Header {
	return http.Header{}
}

func (ClaudeAdapter) RequestIDs(headers http.Header) RequestIDs {
	return RequestIDs{}
}

func (ClaudeAdapter) Route(path string, headers http.Header) Route {
	return Route{Upstream: UpstreamUnknown, Path: path}
}
