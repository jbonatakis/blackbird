package provider

import (
	"net/http"
	"strings"

	"github.com/jbonatakis/blackbird/internal/config"
)

type CodexAdapter struct{}

func (CodexAdapter) ProviderID() string {
	return ProviderCodex
}

func (CodexAdapter) Enabled(memory config.ResolvedMemory) bool {
	return !strings.EqualFold(strings.TrimSpace(memory.Mode), config.MemoryModeOff)
}

func (CodexAdapter) BaseURLPrefix() string {
	return ""
}

func (CodexAdapter) BaseHeaders(ids RequestIDs) http.Header {
	headers := http.Header{}
	if value := strings.TrimSpace(ids.SessionID); value != "" {
		headers.Set(HeaderBlackbirdSessionID, value)
	}
	if value := strings.TrimSpace(ids.TaskID); value != "" {
		headers.Set(HeaderBlackbirdTaskID, value)
	}
	if value := strings.TrimSpace(ids.RunID); value != "" {
		headers.Set(HeaderBlackbirdRunID, value)
	}
	return headers
}

func (CodexAdapter) RequestIDs(headers http.Header) RequestIDs {
	return RequestIDs{
		SessionID: strings.TrimSpace(headers.Get(HeaderBlackbirdSessionID)),
		TaskID:    strings.TrimSpace(headers.Get(HeaderBlackbirdTaskID)),
		RunID:     strings.TrimSpace(headers.Get(HeaderBlackbirdRunID)),
	}
}

func (CodexAdapter) Route(path string, headers http.Header) Route {
	if isChatGPTAuth(headers) {
		return Route{Upstream: UpstreamChatGPT, Path: rewriteChatGPTPath(path)}
	}
	return Route{Upstream: UpstreamAPI, Path: rewriteAPIPath(path)}
}

func isChatGPTAuth(headers http.Header) bool {
	return headerHasValue(headers, "Chatgpt-Account-Id") || headerHasValue(headers, "Session_id")
}

func headerHasValue(headers http.Header, name string) bool {
	for key, values := range headers {
		if !strings.EqualFold(key, name) {
			continue
		}
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				return true
			}
		}
	}
	return false
}

func rewriteAPIPath(path string) string {
	normalized := normalizePath(path)
	if hasPathPrefix(normalized, "/v1") {
		return normalized
	}
	return "/v1" + normalized
}

func rewriteChatGPTPath(path string) string {
	normalized := normalizePath(path)
	if hasPathPrefix(normalized, "/backend-api") {
		return normalized
	}
	if hasPathPrefix(normalized, "/responses") {
		return "/backend-api/codex" + normalized
	}
	if hasPathPrefix(normalized, "/api/codex") {
		return "/backend-api" + normalized
	}
	if hasPathPrefix(normalized, "/wham") {
		return "/backend-api" + normalized
	}
	return "/backend-api" + normalized
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

func hasPathPrefix(path string, prefix string) bool {
	if path == prefix {
		return true
	}
	if strings.HasPrefix(path, prefix+"/") {
		return true
	}
	return false
}
