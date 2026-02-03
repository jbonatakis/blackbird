package provider

import (
	"net/http"
	"testing"

	"github.com/jbonatakis/blackbird/internal/config"
)

func TestSelectAdapter(t *testing.T) {
	codex := Select("codex")
	if codex == nil {
		t.Fatal("expected codex adapter")
	}
	if codex.ProviderID() != ProviderCodex {
		t.Fatalf("codex adapter id = %q, want %q", codex.ProviderID(), ProviderCodex)
	}

	claude := Select(" Claude ")
	if claude == nil {
		t.Fatal("expected claude adapter")
	}
	if claude.ProviderID() != ProviderClaude {
		t.Fatalf("claude adapter id = %q, want %q", claude.ProviderID(), ProviderClaude)
	}

	if got := Select("unknown"); got != nil {
		t.Fatalf("expected nil adapter for unknown provider, got %T", got)
	}
}

func TestAdapterEnabled(t *testing.T) {
	memory := config.DefaultResolvedConfig().Memory
	memory.Mode = config.MemoryModeProvider

	codex := Select("codex")
	if codex == nil {
		t.Fatal("expected codex adapter")
	}
	if !codex.Enabled(memory) {
		t.Fatal("expected codex adapter to be enabled in provider mode")
	}

	memory.Mode = config.MemoryModeOff
	if codex.Enabled(memory) {
		t.Fatal("expected codex adapter to be disabled in off mode")
	}

	claude := Select("claude")
	if claude == nil {
		t.Fatal("expected claude adapter")
	}
	if claude.Enabled(memory) {
		t.Fatal("expected claude adapter to be disabled")
	}
}

func TestCodexRoutingRules(t *testing.T) {
	adapter := CodexAdapter{}
	tests := []struct {
		name     string
		path     string
		headers  http.Header
		upstream Upstream
		wantPath string
	}{
		{
			name:     "api responses",
			path:     "/responses",
			headers:  http.Header{},
			upstream: UpstreamAPI,
			wantPath: "/v1/responses",
		},
		{
			name:     "api v1 pass-through",
			path:     "/v1/models",
			headers:  http.Header{},
			upstream: UpstreamAPI,
			wantPath: "/v1/models",
		},
		{
			name:     "chatgpt responses",
			path:     "/responses",
			headers:  http.Header{"Chatgpt-Account-Id": []string{"acct"}},
			upstream: UpstreamChatGPT,
			wantPath: "/backend-api/codex/responses",
		},
		{
			name:     "chatgpt wham",
			path:     "/wham/health",
			headers:  http.Header{"Session_id": []string{"sess"}},
			upstream: UpstreamChatGPT,
			wantPath: "/backend-api/wham/health",
		},
	}

	for _, test := range tests {
		route := adapter.Route(test.path, test.headers)
		if route.Upstream != test.upstream {
			t.Errorf("%s: upstream = %q, want %q", test.name, route.Upstream, test.upstream)
		}
		if route.Path != test.wantPath {
			t.Errorf("%s: path = %q, want %q", test.name, route.Path, test.wantPath)
		}
	}
}

func TestCodexBaseHeaders(t *testing.T) {
	adapter := CodexAdapter{}
	headers := adapter.BaseHeaders(RequestIDs{SessionID: "session", TaskID: "task", RunID: "run"})
	if got := headers.Get(HeaderBlackbirdSessionID); got != "session" {
		t.Fatalf("session header = %q, want %q", got, "session")
	}
	if got := headers.Get(HeaderBlackbirdTaskID); got != "task" {
		t.Fatalf("task header = %q, want %q", got, "task")
	}
	if got := headers.Get(HeaderBlackbirdRunID); got != "run" {
		t.Fatalf("run header = %q, want %q", got, "run")
	}

	empty := adapter.BaseHeaders(RequestIDs{})
	if len(empty) != 0 {
		t.Fatalf("expected no headers for empty ids, got %v", empty)
	}
}
